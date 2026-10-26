package message

import (
	"context"
	"errors"
	"regexp"

	"go.mau.fi/whatsmeow/types"

	"github.com/open-apime/apime/internal/storage/model"
)

// ErrInvalidWhatsAppID rejects a caller id that WhatsApp clients would not recognize as one of ours.
var ErrInvalidWhatsAppID = errors.New("whatsappId inválido: use hexadecimal maiúsculo com 16 a 32 caracteres")

// Same alphabet as whatsmeow GenerateMessageID ("3EB0" + uppercase hex, 22 chars, what acloud sends);
// 32 covers the longest ids the official apps generate.
var whatsAppIDPattern = regexp.MustCompile(`^[0-9A-F]{16,32}$`)

// ValidWhatsAppID reports whether id can be used as a caller-chosen WhatsApp message id.
func ValidWhatsAppID(id string) bool {
	return whatsAppIDPattern.MatchString(id)
}

// alreadySent finds a successful send of this instance under the caller id, so a retry replays it.
func (s *Service) alreadySent(ctx context.Context, input SendInput) (model.Message, bool) {
	if input.WhatsAppID == "" || s.repo == nil {
		return model.Message{}, false
	}
	prev, err := s.repo.GetByWhatsAppIDForInstance(ctx, input.InstanceID, input.WhatsAppID)
	if err != nil {
		return model.Message{}, false
	}
	return prev, true
}

// resolveSendID picks the id once, before the retry loop, so every attempt reuses it.
func resolveSendID(callerID string, generate func() types.MessageID) types.MessageID {
	if callerID != "" {
		return types.MessageID(callerID)
	}
	return generate()
}
