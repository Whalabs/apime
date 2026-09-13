package webhook

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func mustJID(t *testing.T, s string) types.JID {
	t.Helper()
	jid, err := types.ParseJID(s)
	if err != nil {
		t.Fatalf("ParseJID(%q) failed: %v", s, err)
	}
	return jid
}

func TestBroadcastRecipientsEmitsBothIdentities(t *testing.T) {
	got := broadcastRecipients([]types.BroadcastRecipient{
		{PN: mustJID(t, "5511999999999@s.whatsapp.net"), LID: mustJID(t, "123456@lid")},
	})
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["pn"] != "5511999999999@s.whatsapp.net" {
		t.Errorf("pn = %q, want %q", got[0]["pn"], "5511999999999@s.whatsapp.net")
	}
	if got[0]["lid"] != "123456@lid" {
		t.Errorf("lid = %q, want %q", got[0]["lid"], "123456@lid")
	}
}

func TestBroadcastRecipientsPartialAndEmpty(t *testing.T) {
	cases := []struct {
		name       string
		recipients []types.BroadcastRecipient
		wantLen    int
		wantKeys   []string
	}{
		{"nil", nil, 0, nil},
		{"empty slice", []types.BroadcastRecipient{}, 0, nil},
		{
			"pn only",
			[]types.BroadcastRecipient{{PN: mustJID(t, "5511988887777@s.whatsapp.net")}},
			1, []string{"pn"},
		},
		{
			"lid only (username-only contact)",
			[]types.BroadcastRecipient{{LID: mustJID(t, "987654@lid")}},
			1, []string{"lid"},
		},
		{
			// An entry with neither identity is unaddressable, so it is dropped rather than
			// emitted as an empty object the consumer would have to filter itself.
			"both empty is dropped",
			[]types.BroadcastRecipient{{}},
			0, nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := broadcastRecipients(tc.recipients)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
			if tc.wantLen == 0 && got != nil {
				t.Errorf("got = %v, want nil so the key is omitted from the payload", got)
			}
			for _, key := range tc.wantKeys {
				if got[0][key] == "" {
					t.Errorf("missing key %q in %v", key, got[0])
				}
			}
		})
	}
}

func TestBroadcastRecipientsKeepsOrder(t *testing.T) {
	// The consumer fans the list out in order, so a reordering would silently change which
	// conversation each copy lands in if it ever paired by index.
	got := broadcastRecipients([]types.BroadcastRecipient{
		{PN: mustJID(t, "5511111111111@s.whatsapp.net")},
		{PN: mustJID(t, "5522222222222@s.whatsapp.net")},
		{PN: mustJID(t, "5533333333333@s.whatsapp.net")},
	})
	want := []string{
		"5511111111111@s.whatsapp.net",
		"5522222222222@s.whatsapp.net",
		"5533333333333@s.whatsapp.net",
	}
	for i, w := range want {
		if got[i]["pn"] != w {
			t.Errorf("index %d = %q, want %q", i, got[i]["pn"], w)
		}
	}
}
