package whatsapp

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"seruwit-wa/internal/database"
	"seruwit-wa/internal/webhook"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type Manager struct {
	db         *database.Database
	dispatcher *webhook.Dispatcher
	log        waLog.Logger
	sessions   map[string]*Session
	mu         sync.RWMutex
}

func NewManager(db *database.Database, dispatcher *webhook.Dispatcher, log waLog.Logger) *Manager {
	return &Manager{
		db:         db,
		dispatcher: dispatcher,
		log:        log,
		sessions:   make(map[string]*Session),
	}
}

// LoadSavedSessions loads and reconnects all previously authenticated sessions from database
func (m *Manager) LoadSavedSessions(ctx context.Context) error {
	records, err := m.db.GetAllSessions()
	if err != nil {
		return err
	}

	for _, rec := range records {
		if rec.JID == "" || rec.Status != string(StatusConnected) {
			continue
		}

		jid, err := types.ParseJID(rec.JID)
		if err != nil {
			m.log.Warnf("Invalid JID for session %s: %v", rec.SessionID, err)
			continue
		}

		device, err := m.db.Container.GetDevice(ctx, jid)
		if err != nil || device == nil {
			m.log.Warnf("Device store not found for session %s (%s)", rec.SessionID, rec.JID)
			continue
		}

		client := whatsmeow.NewClient(device, waLog.Stdout(fmt.Sprintf("WA-%s", rec.SessionID), "WARN", true))
		s := &Session{
			SessionID: rec.SessionID,
			Client:    client,
			Device:    device,
			Status:    StatusConnecting,
			JID:       jid,
			Phone:     rec.Phone,
			PushName:  rec.PushName,
		}

		m.registerEventHandlers(s)
		m.mu.Lock()
		m.sessions[rec.SessionID] = s
		m.mu.Unlock()

		go func(sess *Session) {
			if err := sess.Client.Connect(); err != nil {
				m.log.Warnf("Failed to auto-reconnect session %s: %v", sess.SessionID, err)
				sess.SetStatus(StatusDisconnected)
				_ = m.db.UpdateStatus(sess.SessionID, string(StatusDisconnected))
			} else {
				m.log.Infof("Auto-reconnected session %s (%s)", sess.SessionID, sess.Phone)
			}
		}(s)
	}

	return nil
}

// StartSession initializes pairing and generates QR code for the given sessionID
func (m *Manager) StartSession(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.Lock()
	existing, found := m.sessions[sessionID]
	if found && existing.Client != nil && existing.Client.IsLoggedIn() {
		m.mu.Unlock()
		return existing, nil
	}

	device := m.db.Container.NewDevice()
	client := whatsmeow.NewClient(device, waLog.Stdout(fmt.Sprintf("WA-%s", sessionID), "WARN", true))

	sessCtx, cancel := context.WithCancel(context.Background())
	s := &Session{
		SessionID: sessionID,
		Client:    client,
		Device:    device,
		Status:    StatusScanQR,
		CancelQR:  cancel,
	}

	m.registerEventHandlers(s)
	m.sessions[sessionID] = s
	m.mu.Unlock()

	qrChan, err := client.GetQRChannel(sessCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR channel: %w", err)
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect client: %w", err)
	}

	// Wait for the first QR code to be generated before returning to HTTP caller
	firstQRReceived := make(chan bool, 1)

	go func() {
		isFirst := true
		for item := range qrChan {
			switch item.Event {
			case "code":
				png, err := qrcode.Encode(item.Code, qrcode.Medium, 256)
				if err == nil {
					base64Img := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
					s.SetQR(base64Img)
					m.dispatcher.Send(webhook.EventSessionQR, sessionID, map[string]string{
						"qr_base64": base64Img,
						"qr_code":   item.Code,
					})
				}
				if isFirst {
					isFirst = false
					firstQRReceived <- true
				}
			case "success":
				s.SetStatus(StatusConnected)
				s.SetQR("")
				if client.Store.ID != nil {
					jid := *client.Store.ID
					s.JID = jid
					s.Phone = jid.User
					s.PushName = client.Store.PushName

					_ = m.db.UpsertSession(&database.SessionRecord{
						SessionID: sessionID,
						JID:       jid.String(),
						Phone:     jid.User,
						PushName:  s.PushName,
						Status:    string(StatusConnected),
					})
				}
				m.dispatcher.Send(webhook.EventSessionStatus, sessionID, map[string]interface{}{
					"status":    string(StatusConnected),
					"phone":     s.Phone,
					"push_name": s.PushName,
					"jid":       s.JID.String(),
				})
			case "timeout":
				s.SetStatus(StatusDisconnected)
				s.SetQR("")
				_ = m.db.UpdateStatus(sessionID, string(StatusDisconnected))
			}
		}
	}()

	select {
	case <-firstQRReceived:
	case <-time.After(5 * time.Second):
		// Proceed even if QR took more than 5 seconds, client can poll status
	}

	return s, nil
}

// PairWithCode initiates pairing using an 8-digit code instead of QR
func (m *Manager) PairWithCode(ctx context.Context, sessionID, phone string) (string, error) {
	m.mu.Lock()
	device := m.db.Container.NewDevice()
	client := whatsmeow.NewClient(device, waLog.Stdout(fmt.Sprintf("WA-%s", sessionID), "WARN", true))

	s := &Session{
		SessionID: sessionID,
		Client:    client,
		Device:    device,
		Status:    StatusConnecting,
		Phone:     phone,
	}

	m.registerEventHandlers(s)
	m.sessions[sessionID] = s
	m.mu.Unlock()

	if err := client.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}

	// Allow websocket handshake to settle
	time.Sleep(1 * time.Second)

	cleanPhone := SanitizePhone(phone)
	code, err := client.PairPhone(ctx, cleanPhone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("failed to generate pairing code: %w", err)
	}

	s.SetPairCode(code)
	return code, nil
}

func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	s, found := m.sessions[sessionID]
	m.mu.RUnlock()

	if found {
		return s, nil
	}

	// Try reading from DB
	rec, err := m.db.GetSession(sessionID)
	if err != nil {
		return nil, errors.New("session not found")
	}

	return &Session{
		SessionID: rec.SessionID,
		Status:    SessionStatus(rec.Status),
		Phone:     rec.Phone,
		PushName:  rec.PushName,
	}, nil
}

func (m *Manager) Logout(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	s, found := m.sessions[sessionID]
	if found {
		delete(m.sessions, sessionID)
	}
	m.mu.Unlock()

	if found && s.Client != nil {
		if s.CancelQR != nil {
			s.CancelQR()
		}
		_ = s.Client.Logout(ctx)
		s.Client.Disconnect()
		if s.Device != nil {
			_ = m.db.Container.DeleteDevice(ctx, s.Device)
		}
	}

	_ = m.db.DeleteSession(sessionID)
	m.dispatcher.Send(webhook.EventSessionStatus, sessionID, map[string]interface{}{
		"status": "LOGGED_OUT",
	})

	return nil
}

func (m *Manager) SendTextMessage(ctx context.Context, sessionID, recipient, text string) (string, error) {
	m.mu.RLock()
	s, found := m.sessions[sessionID]
	m.mu.RUnlock()

	if !found || s.Client == nil {
		return "", errors.New("session not found or not initialized")
	}

	if !s.Client.IsConnected() || !s.Client.IsLoggedIn() {
		return "", errors.New("session is not connected or logged in to WhatsApp")
	}

	targetJID := ToUserJID(recipient)
	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	resp, err := s.Client.SendMessage(ctx, targetJID, msg)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return resp.ID, nil
}
