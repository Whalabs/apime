package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/open-apime/apime/internal/storage/model"
)

type eventLogRepo struct {
	db *DB
}

// NewEventLogRepository cria um novo repositório de logs de eventos.
func NewEventLogRepository(db *DB) *eventLogRepo {
	return &eventLogRepo{db: db}
}

func (r *eventLogRepo) Create(ctx context.Context, eventLog model.EventLog) (model.EventLog, error) {
	if eventLog.ID == "" {
		eventLog.ID = uuid.New().String()
	}
	eventLog.CreatedAt = time.Now()

	// Converter payload para JSONB
	payloadJSON, err := json.Marshal(eventLog.Payload)
	if err != nil {
		return model.EventLog{}, err
	}

	query := `
		INSERT INTO event_logs (id, instance_id, type, payload, delivered_at, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6)
		RETURNING id, instance_id, type, payload, delivered_at, created_at
	`

	var payloadBytes []byte
	err = r.db.Pool.QueryRow(ctx, query,
		eventLog.ID, eventLog.InstanceID, eventLog.Type, payloadJSON, eventLog.DeliveredAt, eventLog.CreatedAt,
	).Scan(
		&eventLog.ID, &eventLog.InstanceID, &eventLog.Type, &payloadBytes, &eventLog.DeliveredAt, &eventLog.CreatedAt,
	)

	if err != nil {
		return model.EventLog{}, err
	}

	// Converter JSONB de volta para string
	var payloadMap interface{}
	if err := json.Unmarshal(payloadBytes, &payloadMap); err == nil {
		if str, ok := payloadMap.(string); ok {
			eventLog.Payload = str
		} else {
			eventLog.Payload = string(payloadBytes)
		}
	} else {
		eventLog.Payload = string(payloadBytes)
	}

	return eventLog, nil
}

func (r *eventLogRepo) ListByInstance(ctx context.Context, instanceID string) ([]model.EventLog, error) {
	query := `
		SELECT id, instance_id, type, payload, delivered_at, created_at
		FROM event_logs
		WHERE instance_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := r.db.Pool.Query(ctx, query, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var eventLogs []model.EventLog
	for rows.Next() {
		var eventLog model.EventLog
		var payloadBytes []byte
		if err := rows.Scan(
			&eventLog.ID, &eventLog.InstanceID, &eventLog.Type, &payloadBytes, &eventLog.DeliveredAt, &eventLog.CreatedAt,
		); err != nil {
			return nil, err
		}

		// Converter JSONB de volta para string
		var payloadMap interface{}
		if err := json.Unmarshal(payloadBytes, &payloadMap); err == nil {
			if str, ok := payloadMap.(string); ok {
				eventLog.Payload = str
			} else {
				eventLog.Payload = string(payloadBytes)
			}
		} else {
			eventLog.Payload = string(payloadBytes)
		}

		eventLogs = append(eventLogs, eventLog)
	}

	return eventLogs, rows.Err()
}

func (r *eventLogRepo) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	query := `DELETE FROM event_logs WHERE instance_id = $1`
	_, err := r.db.Pool.Exec(ctx, query, instanceID)
	return err
}
