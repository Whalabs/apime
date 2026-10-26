package handler

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Moving the send routes into a sub-group must not change a single path.
func TestSendRoutesKeepTheirPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewMessageHandler(nil, nil).Register(r.Group("/api"))

	var got []string
	for _, route := range r.Routes() {
		got = append(got, route.Method+" "+route.Path)
	}
	sort.Strings(got)

	want := []string{
		"GET /api/instances/:id/messages",
		"POST /api/instances/:id/messages",
		"POST /api/instances/:id/messages/audio",
		"POST /api/instances/:id/messages/contact",
		"POST /api/instances/:id/messages/document",
		"POST /api/instances/:id/messages/location",
		"POST /api/instances/:id/messages/media",
		"POST /api/instances/:id/messages/text",
		"POST /api/instances/:id/stories/media",
		"POST /api/instances/:id/stories/text",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("rotas registradas: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rota %d: got %q want %q", i, got[i], want[i])
		}
	}
}

// The allowlist and the error message must stay in sync: a type accepted here but missing from the
// message (or the reverse) is exactly the drift the map replaced a comparison chain to avoid.
func TestMediaAllowlistCoversTheDocumentedTypes(t *testing.T) {
	want := []string{"image", "video", "gif", "sticker"}
	if len(allowedMediaTypes) != len(want) {
		t.Fatalf("allowlist tem %d tipos, esperava %d", len(allowedMediaTypes), len(want))
	}
	for _, mediaType := range want {
		if !allowedMediaTypes[mediaType] {
			t.Fatalf("tipo %q deveria ser aceito", mediaType)
		}
	}
	for _, refused := range []string{"", "audio", "document", "GIF", "image/webp"} {
		if allowedMediaTypes[refused] {
			t.Fatalf("tipo %q não deveria ser aceito", refused)
		}
	}
}

// A malformed caller id is answered with 400 before the service is reached (the service is nil here).
func TestSendTextRejectsMalformedWhatsAppID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("authType", "instance_token")
		c.Set("instanceID", "inst-a")
	})
	NewMessageHandler(nil, nil).Register(r.Group("/api"))

	body := `{"to":"5511999999999","text":"oi","whatsappId":"3eb0-not-hex"}`
	req := httptest.NewRequest(http.MethodPost, "/api/instances/inst-a/messages/text", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, body %s", w.Code, w.Body.String())
	}
}
