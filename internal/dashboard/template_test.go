package dashboard

import "testing"

// The templates are parsed with template.Must at boot, so a broken one panics
// the service instead of failing a request. This catches it in CI instead.
func TestTemplatesParse(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("templates failed to parse: %v", r)
		}
	}()
	_ = HTMLTemplate()
}
