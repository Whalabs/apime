package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/open-apime/apime/internal/storage/model"
)

type deviceConfigRepo struct {
	db *DB
}

// NewDeviceConfigRepository cria um novo repositório de configurações de dispositivo.
func NewDeviceConfigRepository(db *DB) *deviceConfigRepo {
	return &deviceConfigRepo{db: db}
}

func (r *deviceConfigRepo) Get(ctx context.Context) (model.DeviceConfig, error) {
	query := `
		SELECT id, platform_type, os_name, push_name, created_at, updated_at
		FROM device_config
		ORDER BY created_at ASC
		LIMIT 1
	`

	var config model.DeviceConfig
	err := r.db.Pool.QueryRow(ctx, query).Scan(
		&config.ID, &config.PlatformType,
		&config.OSName, &config.PushName, &config.CreatedAt, &config.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		// Se não existe, retornar valores padrão
		return model.DeviceConfig{
			ID:           uuid.NewString(),
			PlatformType: "DESKTOP",
			OSName:       "ApiMe",
			PushName:     "ApiMe Server",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}, nil
	}

	if err != nil {
		return model.DeviceConfig{}, err
	}

	return config, nil
}

func (r *deviceConfigRepo) Update(ctx context.Context, config model.DeviceConfig) (model.DeviceConfig, error) {
	config.UpdatedAt = time.Now()

	// Usar UPSERT (INSERT ... ON CONFLICT UPDATE)
	query := `
		INSERT INTO device_config (id, platform_type, os_name, push_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET platform_type = EXCLUDED.platform_type,
		    os_name = EXCLUDED.os_name,
		    push_name = EXCLUDED.push_name,
		    updated_at = EXCLUDED.updated_at
		RETURNING id, platform_type, os_name, push_name, created_at, updated_at
	`

	if config.ID == "" {
		config.ID = "00000000-0000-0000-0000-000000000001"
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	err := r.db.Pool.QueryRow(ctx, query,
		config.ID, config.PlatformType,
		config.OSName, config.PushName, config.CreatedAt, config.UpdatedAt,
	).Scan(
		&config.ID, &config.PlatformType,
		&config.OSName, &config.PushName, &config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		return model.DeviceConfig{}, err
	}

	return config, nil
}
