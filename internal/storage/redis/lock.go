package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Lock representa um lock distribuído.
type Lock struct {
	client *Client
	key    string
	value  string
	ttl    time.Duration
}

// Acquire tenta adquirir o lock. Retorna true se adquirido com sucesso.
func (l *Lock) Acquire(ctx context.Context) (bool, error) {
	l.value = uuid.New().String()
	acquired, err := l.client.rdb.SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("lock acquire: %w", err)
	}
	return acquired, nil
}

// Release libera o lock.
func (l *Lock) Release(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := l.client.rdb.Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("lock release: %w", err)
	}
	return nil
}

// NewLock cria um novo lock distribuído.
func NewLock(client *Client, key string, ttl time.Duration) *Lock {
	return &Lock{
		client: client,
		key:    key,
		ttl:    ttl,
	}
}
