package message

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

func TestBuildStatusTextAlwaysCarriesStyling(t *testing.T) {
	// The whole point of the status branch: a bare Conversation is a shape no real status has, and
	// a recipient on an older client renders it as "update WhatsApp to view this message". The
	// background and font must therefore always be present, even when the caller sends none.
	msg := buildStatusText(SendInput{Text: "olá"})

	if msg.GetText() != "olá" {
		t.Errorf("text = %q, want %q", msg.GetText(), "olá")
	}
	if msg.BackgroundArgb == nil {
		t.Error("backgroundArgb must be set, it is what makes it render as a status")
	}
	if msg.TextArgb == nil {
		t.Error("textArgb must be set")
	}
	if msg.Font == nil {
		t.Error("font must be set")
	}
	if msg.GetBackgroundArgb() != defaultStatusBackgroundArgb {
		t.Errorf("backgroundArgb = %#x, want %#x", msg.GetBackgroundArgb(), defaultStatusBackgroundArgb)
	}
}

func TestBuildStatusTextHonoursExplicitStyling(t *testing.T) {
	msg := buildStatusText(SendInput{
		Text:           "estilizado",
		BackgroundArgb: 0xFF112233,
		TextArgb:       0xFF445566,
		Font:           "calistoga_regular",
	})

	if msg.GetBackgroundArgb() != 0xFF112233 {
		t.Errorf("backgroundArgb = %#x, want %#x", msg.GetBackgroundArgb(), 0xFF112233)
	}
	if msg.GetTextArgb() != 0xFF445566 {
		t.Errorf("textArgb = %#x, want %#x", msg.GetTextArgb(), 0xFF445566)
	}
	if msg.GetFont() != waE2E.ExtendedTextMessage_CALISTOGA_REGULAR {
		t.Errorf("font = %v, want CALISTOGA_REGULAR", msg.GetFont())
	}
}

func TestStatusFontFallsBackToSystem(t *testing.T) {
	// An unknown name must not travel to the recipient: a font value their client does not know is
	// exactly the kind of thing that renders as an unsupported message.
	for _, name := range []string{"", "   ", "comic sans", "SYSTEM_ITALIC", "99"} {
		if got := statusFont(name); got != waE2E.ExtendedTextMessage_SYSTEM {
			t.Errorf("statusFont(%q) = %v, want SYSTEM", name, got)
		}
	}
}

func TestStatusFontIsCaseAndSpaceInsensitive(t *testing.T) {
	for _, name := range []string{"CALISTOGA_REGULAR", " calistoga_regular ", "Calistoga_Regular"} {
		if got := statusFont(name); got != waE2E.ExtendedTextMessage_CALISTOGA_REGULAR {
			t.Errorf("statusFont(%q) = %v, want CALISTOGA_REGULAR", name, got)
		}
	}
}

func TestEveryOfferedFontMapsToADistinctValue(t *testing.T) {
	seen := make(map[waE2E.ExtendedTextMessage_FontType]string, len(statusFonts))
	for name := range statusFonts {
		font := statusFont(name)
		if other, dup := seen[font]; dup {
			t.Errorf("fonts %q and %q map to the same value %v", name, other, font)
		}
		seen[font] = name
	}
}
