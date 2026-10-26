package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func newTestMessageRepo(t *testing.T) (*messageRepo, *sql.DB) {
	t.Helper()
	conn, err := sql.Open("sqlite3", "file:"+filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	// Only the columns the lookup reads; the real table also has a foreign key to instances.
	if _, err := conn.Exec(`CREATE TABLE message_queue (
		id TEXT PRIMARY KEY, instance_id TEXT NOT NULL, recipient TEXT NOT NULL, type TEXT NOT NULL,
		payload TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (datetime('now')),
		whatsapp_id TEXT, delivered_at TEXT)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return NewMessageRepository(&DB{Conn: conn}), conn
}

func TestGetByWhatsAppIDForInstance(t *testing.T) {
	repo, conn := newTestMessageRepo(t)
	rows := [][]string{
		{"m1", "inst-a", "3EB0AAAAAAAAAAAAAAAAAA", "sent"},
		{"m2", "inst-a", "3EB0BBBBBBBBBBBBBBBBBB", "failed"},
		{"m3", "inst-a", "3EB0CCCCCCCCCCCCCCCCCC", "delivered"},
	}
	for _, r := range rows {
		if _, err := conn.Exec(`INSERT INTO message_queue (id, instance_id, recipient, type, payload, status, whatsapp_id) VALUES (?, ?, '5511', 'text', '{"text":"oi"}', ?, ?)`, r[0], r[1], r[3], r[2]); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	ctx := context.Background()

	if got, err := repo.GetByWhatsAppIDForInstance(ctx, "inst-a", "3EB0AAAAAAAAAAAAAAAAAA"); err != nil || got.ID != "m1" {
		t.Fatalf("sent row: got %+v err %v", got, err)
	}
	if got, err := repo.GetByWhatsAppIDForInstance(ctx, "inst-a", "3EB0CCCCCCCCCCCCCCCCCC"); err != nil || got.ID != "m3" {
		t.Fatalf("receipt status still counts as sent: got %+v err %v", got, err)
	}
	if _, err := repo.GetByWhatsAppIDForInstance(ctx, "inst-b", "3EB0AAAAAAAAAAAAAAAAAA"); err == nil {
		t.Fatal("another instance must not find it")
	}
	if _, err := repo.GetByWhatsAppIDForInstance(ctx, "inst-a", "3EB0BBBBBBBBBBBBBBBBBB"); err == nil {
		t.Fatal("a failed send must not be replayed")
	}
}
