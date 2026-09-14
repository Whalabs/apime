package message

import (
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

// Defaults for a text status. The colours are opaque ARGB; the background is WhatsApp's own green
// and the text white, which is what the composer offers first.
const (
	defaultStatusBackgroundArgb uint32 = 0xFF25D366
	defaultStatusTextArgb       uint32 = 0xFFFFFFFF
)

// statusFonts maps the font names the API accepts to the proto enum. Only the fonts the status
// composer actually offers are listed, so an unknown name falls back to the system font rather
// than sending a value the recipient's client may not know.
var statusFonts = map[string]waE2E.ExtendedTextMessage_FontType{
	"system":                waE2E.ExtendedTextMessage_SYSTEM,
	"system_text":           waE2E.ExtendedTextMessage_SYSTEM_TEXT,
	"system_bold":           waE2E.ExtendedTextMessage_SYSTEM_BOLD,
	"fb_script":             waE2E.ExtendedTextMessage_FB_SCRIPT,
	"morningbreeze_regular": waE2E.ExtendedTextMessage_MORNINGBREEZE_REGULAR,
	"calistoga_regular":     waE2E.ExtendedTextMessage_CALISTOGA_REGULAR,
	"exo2_extrabold":        waE2E.ExtendedTextMessage_EXO2_EXTRABOLD,
	"courierprime_bold":     waE2E.ExtendedTextMessage_COURIERPRIME_BOLD,
}

// buildStatusText assembles the ExtendedTextMessage of a text status, filling in the styling the
// official composer always sets.
func buildStatusText(input SendInput) *waE2E.ExtendedTextMessage {
	background := input.BackgroundArgb
	if background == 0 {
		background = defaultStatusBackgroundArgb
	}
	text := input.TextArgb
	if text == 0 {
		text = defaultStatusTextArgb
	}
	return &waE2E.ExtendedTextMessage{
		Text:           proto.String(input.Text),
		BackgroundArgb: proto.Uint32(background),
		TextArgb:       proto.Uint32(text),
		Font:           statusFont(input.Font).Enum(),
	}
}

func statusFont(name string) waE2E.ExtendedTextMessage_FontType {
	if font, ok := statusFonts[strings.ToLower(strings.TrimSpace(name))]; ok {
		return font
	}
	return waE2E.ExtendedTextMessage_SYSTEM
}
