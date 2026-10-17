package whatsapp

import (
	"seruwit-wa/internal/database"
	"seruwit-wa/internal/webhook"

	"go.mau.fi/whatsmeow/types/events"
)

func (m *Manager) registerEventHandlers(s *Session) {
	s.Client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			s.SetStatus(StatusConnected)
			_ = m.db.UpdateStatus(s.SessionID, string(StatusConnected))
			m.dispatcher.Send(webhook.EventSessionStatus, s.SessionID, map[string]interface{}{
				"status":    string(StatusConnected),
				"phone":     s.Phone,
				"push_name": s.PushName,
				"jid":       s.JID.String(),
			})

		case *events.Disconnected:
			s.SetStatus(StatusDisconnected)
			_ = m.db.UpdateStatus(s.SessionID, string(StatusDisconnected))
			m.dispatcher.Send(webhook.EventSessionStatus, s.SessionID, map[string]interface{}{
				"status": string(StatusDisconnected),
			})

		case *events.LoggedOut:
			s.SetStatus(StatusDisconnected)
			_ = m.db.DeleteSession(s.SessionID)
			m.dispatcher.Send(webhook.EventSessionStatus, s.SessionID, map[string]interface{}{
				"status": "LOGGED_OUT",
				"reason": v.Reason.String(),
			})

		case *events.PairSuccess:
			s.SetStatus(StatusConnected)
			s.SetQR("")
			s.SetPairCode("")

			if s.Client.Store.ID != nil {
				jid := *s.Client.Store.ID
				s.JID = jid
				s.Phone = jid.User
				s.PushName = s.Client.Store.PushName

				_ = m.db.UpsertSession(&database.SessionRecord{
					SessionID: s.SessionID,
					JID:       jid.String(),
					Phone:     jid.User,
					PushName:  s.PushName,
					Status:    string(StatusConnected),
				})
			}

			m.dispatcher.Send(webhook.EventSessionStatus, s.SessionID, map[string]interface{}{
				"status":    string(StatusConnected),
				"phone":     s.Phone,
				"push_name": s.PushName,
				"jid":       s.JID.String(),
			})

		case *events.Receipt:
			receiptType := "delivered"
			if v.Type == events.ReceiptTypeRead || v.Type == events.ReceiptTypeReadSelf {
				receiptType = "read"
			} else if v.Type == events.ReceiptTypeDelivered {
				receiptType = "delivered"
			}

			m.dispatcher.Send(webhook.EventMessageReceipt, s.SessionID, map[string]interface{}{
				"message_ids": v.MessageIDs,
				"receipt":     receiptType,
				"timestamp":   v.Timestamp.Unix(),
			})

		case *events.Message:
			// Handle incoming message (e.g. tenant's customer replies to notification)
			if v.Info.IsFromMe {
				return
			}

			senderPhone := v.Info.Sender.User
			text := ""
			if v.Message != nil {
				if conv := v.Message.GetConversation(); conv != "" {
					text = conv
				} else if ext := v.Message.GetExtendedTextMessage(); ext != nil {
					text = ext.GetText()
				} else if img := v.Message.GetImageMessage(); img != nil {
					text = img.GetCaption()
				} else if doc := v.Message.GetDocumentMessage(); doc != nil {
					text = doc.GetCaption()
				}
			}

			m.dispatcher.Send(webhook.EventMessageIncoming, s.SessionID, map[string]interface{}{
				"message_id": v.Info.ID,
				"sender":     senderPhone,
				"text":       text,
				"timestamp":  v.Info.Timestamp.Unix(),
				"push_name":  v.Info.PushName,
			})
		}
	})
}
