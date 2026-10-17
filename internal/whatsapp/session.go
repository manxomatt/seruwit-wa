package whatsapp

import (
	"context"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/types"
)

type SessionStatus string

const (
	StatusIdle         SessionStatus = "IDLE"
	StatusScanQR       SessionStatus = "SCAN_QR"
	StatusConnecting   SessionStatus = "CONNECTING"
	StatusConnected    SessionStatus = "CONNECTED"
	StatusDisconnected SessionStatus = "DISCONNECTED"
)

type Session struct {
	SessionID  string
	Client     *whatsmeow.Client
	Device     *store.Device
	Status     SessionStatus
	QRBase64   string
	PairCode   string
	CancelQR   context.CancelFunc
	JID        types.JID
	Phone      string
	PushName   string
	mu         sync.RWMutex
}

func (s *Session) SetStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}

func (s *Session) GetStatus() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

func (s *Session) SetQR(qr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.QRBase64 = qr
}

func (s *Session) GetQR() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.QRBase64
}

func (s *Session) SetPairCode(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PairCode = code
}

func (s *Session) GetPairCode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PairCode
}
