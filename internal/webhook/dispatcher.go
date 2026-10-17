package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	waLog "go.mau.fi/whatsmeow/util/log"
)

type EventType string

const (
	EventSessionStatus   EventType = "session.status"
	EventSessionQR       EventType = "session.qr"
	EventMessageIncoming EventType = "message.incoming"
	EventMessageReceipt  EventType = "message.receipt"
)

type Payload struct {
	Event     EventType   `json:"event"`
	SessionID string      `json:"session_id"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type Dispatcher struct {
	url    string
	secret string
	client *http.Client
	log    waLog.Logger
}

func NewDispatcher(url, secret string, log waLog.Logger) *Dispatcher {
	return &Dispatcher{
		url:    url,
		secret: secret,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		log: log,
	}
}

func (d *Dispatcher) Send(event EventType, sessionID string, data interface{}) {
	if d.url == "" {
		return
	}

	payload := Payload{
		Event:     event,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	// Dispatch asynchronously so we never block WhatsApp socket routines
	go func() {
		body, err := json.Marshal(payload)
		if err != nil {
			if d.log != nil {
				d.log.Errorf("Failed to marshal webhook payload: %v", err)
			}
			return
		}

		req, err := http.NewRequest("POST", d.url, bytes.NewBuffer(body))
		if err != nil {
			if d.log != nil {
				d.log.Errorf("Failed to create webhook request: %v", err)
			}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		if d.secret != "" {
			req.Header.Set("X-Webhook-Secret", d.secret)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			if d.log != nil {
				d.log.Warnf("Failed to deliver webhook to %s: %v", d.url, err)
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if d.log != nil {
				d.log.Warnf("Webhook returned non-2xx status: %d", resp.StatusCode)
			}
		}
	}()
}
