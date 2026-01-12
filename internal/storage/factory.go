package storage

import (
	"go.uber.org/zap"

	"github.com/open-apime/apime/internal/config"
	"github.com/open-apime/apime/internal/storage/memory"
	"github.com/open-apime/apime/internal/storage/postgres"
	"github.com/open-apime/apime/internal/storage/redis"
)

// Repositories agrupa todos os repositórios.
type Repositories struct {
	Instance     InstanceRepository
	Message      MessageRepository
	EventLog     EventLogRepository
	User         UserRepository
	APIToken     APITokenRepository
	DeviceConfig DeviceConfigRepository
	RedisClient  *redis.Client
	WebhookQueue *redis.Queue
}

// NewRepositories cria repositórios baseado no driver configurado.
func NewRepositories(cfg config.Config, log *zap.Logger) (*Repositories, error) {
	log.Info("inicializando repositórios",
		zap.String("driver", cfg.Storage.Driver),
	)

	// Inicializar Redis (sempre necessário para webhooks)
	log.Debug("criando conexão com Redis")
	redisClient, err := redis.New(cfg.Redis, log)
	if err != nil {
		log.Error("erro ao conectar com Redis", zap.Error(err))
		return nil, err
	}
	webhookQueue := redis.NewQueue(redisClient, "webhook:events")
	log.Info("Redis conectado e fila de webhooks criada")

	switch cfg.Storage.Driver {
	case "postgres":
		log.Debug("criando conexão com PostgreSQL")
		db, err := postgres.New(cfg.DB, log)
		if err != nil {
			log.Error("erro ao conectar com PostgreSQL", zap.Error(err))
			return nil, err
		}

		log.Info("repositórios PostgreSQL criados com sucesso")
		return &Repositories{
			Instance:     postgres.NewInstanceRepository(db),
			Message:      postgres.NewMessageRepository(db),
			EventLog:     postgres.NewEventLogRepository(db),
			User:         postgres.NewUserRepository(db),
			APIToken:     postgres.NewAPITokenRepository(db),
			DeviceConfig: postgres.NewDeviceConfigRepository(db),
			RedisClient:  redisClient,
			WebhookQueue: webhookQueue,
		}, nil

	case "memory", "":
		log.Info("usando repositórios em memória")
		return &Repositories{
			Instance:     memory.NewInstanceStore(),
			Message:      memory.NewMessageStore(),
			EventLog:     memory.NewEventLogStore(),
			User:         memory.NewUserStore(),
			APIToken:     memory.NewAPITokenStore(),
			DeviceConfig: memory.NewDeviceConfigStore(),
			RedisClient:  redisClient,
			WebhookQueue: webhookQueue,
		}, nil

	default:
		log.Error("driver de storage desconhecido",
			zap.String("driver", cfg.Storage.Driver),
		)
		return nil, &ErrUnknownDriver{Driver: cfg.Storage.Driver}
	}
}

type ErrUnknownDriver struct {
	Driver string
}

func (e *ErrUnknownDriver) Error() string {
	return "storage: driver desconhecido: " + e.Driver
}
