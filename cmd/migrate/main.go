package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-apime/apime/internal/config"
)

func main() {
	migrationsDir := flag.String("migrations", "db/migrations", "Diretório contendo arquivos *.up.sql")
	seedsDir := flag.String("seeds", "db/seeds", "Diretório contendo arquivos de seed (*.sql)")
	withSeeds := flag.Bool("with-seeds", false, "Executar seeds após as migrations")
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("migrate: falha ao conectar no banco: %v", err)
	}
	defer pool.Close()

	log.Println("migrate: conectado ao banco, garantindo tabela de controle...")
	if err := ensureSchemaMigrations(ctx, pool); err != nil {
		log.Fatalf("migrate: falha ao preparar schema_migrations: %v", err)
	}

	if err := applyMigrations(ctx, pool, *migrationsDir); err != nil {
		log.Fatalf("migrate: erro ao aplicar migrations: %v", err)
	}

	if *withSeeds {
		if err := runSeeds(ctx, pool, *seedsDir); err != nil {
			log.Fatalf("migrate: erro ao executar seeds: %v", err)
		}
	}

	log.Println("migrate: concluído com sucesso.")
}

func ensureSchemaMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := listSQLFiles(dir, ".up.sql")
	if err != nil {
		return fmt.Errorf("listar migrations: %w", err)
	}
	if len(files) == 0 {
		log.Printf("migrate: nenhum arquivo .up.sql encontrado em %s", dir)
		return nil
	}

	for _, file := range files {
		version := filepath.Base(file)
		applied, err := migrationApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		log.Printf("migrate: aplicando %s ...", version)
		sqlStmt, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("ler %s: %w", version, err)
		}

		if err := execSQL(ctx, pool, string(sqlStmt)); err != nil {
			return fmt.Errorf("executar %s: %w", version, err)
		}

		if _, err := pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("registrar %s: %w", version, err)
		}

		log.Printf("migrate: %s aplicado.", version)
	}
	return nil
}

func runSeeds(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	files, err := listSQLFiles(dir, ".sql")
	if err != nil {
		return fmt.Errorf("listar seeds: %w", err)
	}
	if len(files) == 0 {
		log.Printf("migrate: nenhum seed encontrado em %s", dir)
		return nil
	}

	for _, file := range files {
		sqlStmt, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("ler seed %s: %w", file, err)
		}
		rowsAffected, err := execSQLWithRowsAffected(ctx, pool, string(sqlStmt))
		if err != nil {
			return fmt.Errorf("executar seed %s: %w", file, err)
		}
		if rowsAffected > 0 {
			log.Printf("migrate: seed %s aplicado (%d linhas inseridas)", filepath.Base(file), rowsAffected)
		}
	}
	return nil
}

func execSQL(ctx context.Context, pool *pgxpool.Pool, statement string) error {
	sql := strings.TrimSpace(statement)
	if sql == "" {
		return nil
	}
	ctxExec, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	_, err := pool.Exec(ctxExec, sql)
	return err
}

func execSQLWithRowsAffected(ctx context.Context, pool *pgxpool.Pool, statement string) (int64, error) {
	sql := strings.TrimSpace(statement)
	if sql == "" {
		return 0, nil
	}
	ctxExec, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	result, err := pool.Exec(ctxExec, sql)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

func migrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar %s: %w", version, err)
	}
	return exists, nil
}

func listSQLFiles(dir, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if suffix != "" && !strings.HasSuffix(name, suffix) {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files, nil
}
