package whatsapp

import (
	"regexp"
	"strings"

	"go.mau.fi/whatsmeow/types"
)

var nonDigits = regexp.MustCompile(`\D`)

// SanitizePhone formats phone numbers cleanly (e.g., 081234 -> 6281234)
func SanitizePhone(phone string) string {
	cleaned := nonDigits.ReplaceAllString(phone, "")
	if strings.HasPrefix(cleaned, "0") {
		cleaned = "62" + cleaned[1:]
	}
	return cleaned
}

// ToUserJID converts a phone number to WhatsApp user JID
func ToUserJID(phone string) types.JID {
	clean := SanitizePhone(phone)
	return types.NewJID(clean, types.DefaultUserServer)
}
