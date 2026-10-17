package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// initialAdmin reads ADMIN_EMAIL and ADMIN_PASSWORD. With both set, the first admin gets them instead
// of the seed's default password, so a fresh install exposed to the internet never runs on admin123.
func initialAdmin() (email, hash string, ok bool, err error) {
	email = strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" || password == "" {
		return "", "", false, nil
	}
	if len(password) < 8 {
		return "", "", false, fmt.Errorf("ADMIN_PASSWORD precisa de pelo menos 8 caracteres")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", false, err
	}
	return email, string(h), true, nil
}

// Runs before the seeds: their insert only happens when no admin exists, so it then does nothing.
func ensureAdminSQLite(ctx context.Context, db *sql.DB) error {
	email, hash, ok, err := initialAdmin()
	if err != nil || !ok {
		return err
	}
	res, err := db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, role, created_at)
		SELECT ?, ?, ?, 'admin', datetime('now')
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE role = 'admin')`, uuid.NewString(), email, hash)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("migrate: admin inicial criado (%s)", email)
	}
	return nil
}

func ensureAdminPostgres(ctx context.Context, pool *pgxpool.Pool) error {
	email, hash, ok, err := initialAdmin()
	if err != nil || !ok {
		return err
	}
	tag, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, created_at)
		SELECT $1, $2, $3, 'admin', now()
		WHERE NOT EXISTS (SELECT 1 FROM users WHERE role = 'admin')`, uuid.NewString(), email, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		log.Printf("migrate: admin inicial criado (%s)", email)
	}
	return nil
}
