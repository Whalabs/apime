package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/open-apime/apime/internal/storage/model"
)

var ErrNotFound = errors.New("registro não encontrado")

type DeviceConfigStore struct {
	mu     sync.RWMutex
	config model.DeviceConfig
}

func NewDeviceConfigStore() *DeviceConfigStore {
	return &DeviceConfigStore{
		config: model.DeviceConfig{
			ID:           "00000000-0000-0000-0000-000000000001",
			PlatformType: "DESKTOP",
			OSName:       "ApiMe",
			PushName:     "ApiMe Server",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
}

func (s *DeviceConfigStore) Get(ctx context.Context) (model.DeviceConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config, nil
}

func (s *DeviceConfigStore) Update(ctx context.Context, config model.DeviceConfig) (model.DeviceConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	if config.ID == "" {
		config.ID = "00000000-0000-0000-0000-000000000001"
	}
	s.config = config
	return s.config, nil
}

type InstanceStore struct {
	mu   sync.RWMutex
	data map[string]model.Instance
}

func NewInstanceStore() *InstanceStore {
	return &InstanceStore{data: make(map[string]model.Instance)}
}

func (s *InstanceStore) Create(ctx context.Context, instance model.Instance) (model.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if instance.ID == "" {
		instance.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	instance.CreatedAt = now
	instance.UpdatedAt = now
	s.data[instance.ID] = instance
	return instance, nil
}

func (s *InstanceStore) GetByID(ctx context.Context, id string) (model.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	instance, ok := s.data[id]
	if !ok {
		return model.Instance{}, ErrNotFound
	}
	return instance, nil
}

func (s *InstanceStore) GetByTokenHash(ctx context.Context, tokenHash string) (model.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, instance := range s.data {
		if instance.TokenHash == tokenHash {
			return instance, nil
		}
	}
	return model.Instance{}, ErrNotFound
}

func (s *InstanceStore) List(ctx context.Context) ([]model.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Instance, 0, len(s.data))
	for _, inst := range s.data {
		out = append(out, inst)
	}
	return out, nil
}

func (s *InstanceStore) Update(ctx context.Context, instance model.Instance) (model.Instance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[instance.ID]; !ok {
		return model.Instance{}, ErrNotFound
	}
	instance.UpdatedAt = time.Now().UTC()
	s.data[instance.ID] = instance
	return instance, nil
}

func (s *InstanceStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

func (s *InstanceStore) ListByOwner(ctx context.Context, ownerUserID string) ([]model.Instance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []model.Instance
	for _, instance := range s.data {
		if instance.OwnerUserID == ownerUserID {
			result = append(result, instance)
		}
	}
	return result, nil
}

type MessageStore struct {
	mu   sync.RWMutex
	data map[string][]model.Message
}

func NewMessageStore() *MessageStore {
	return &MessageStore{data: make(map[string][]model.Message)}
}

func (s *MessageStore) Create(ctx context.Context, message model.Message) (model.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if message.ID == "" {
		message.ID = uuid.NewString()
	}
	message.CreatedAt = time.Now().UTC()
	if message.Status == "" {
		message.Status = "queued"
	}
	s.data[message.InstanceID] = append(s.data[message.InstanceID], message)
	return message, nil
}

func (s *MessageStore) ListByInstance(ctx context.Context, instanceID string) ([]model.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.data[instanceID]
	out := make([]model.Message, len(list))
	copy(out, list)
	return out, nil
}

func (s *MessageStore) Update(ctx context.Context, msg model.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	messages := s.data[msg.InstanceID]
	for i, m := range messages {
		if m.ID == msg.ID {
			messages[i] = msg
			s.data[msg.InstanceID] = messages
			return nil
		}
	}
	return ErrNotFound
}

func (s *MessageStore) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, instanceID)
	return nil
}

type EventLogStore struct {
	mu   sync.RWMutex
	data map[string][]model.EventLog
}

func NewEventLogStore() *EventLogStore {
	return &EventLogStore{data: make(map[string][]model.EventLog)}
}

func (s *EventLogStore) Create(ctx context.Context, eventLog model.EventLog) (model.EventLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if eventLog.ID == "" {
		eventLog.ID = uuid.NewString()
	}
	eventLog.CreatedAt = time.Now().UTC()
	s.data[eventLog.InstanceID] = append(s.data[eventLog.InstanceID], eventLog)
	return eventLog, nil
}

func (s *EventLogStore) ListByInstance(ctx context.Context, instanceID string) ([]model.EventLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.data[instanceID]
	out := make([]model.EventLog, len(list))
	copy(out, list)
	return out, nil
}

func (s *EventLogStore) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, instanceID)
	return nil
}

type UserStore struct {
	mu   sync.RWMutex
	data map[string]model.User
}

func NewUserStore() *UserStore {
	return &UserStore{data: make(map[string]model.User)}
}

func (s *UserStore) Create(ctx context.Context, user model.User) (model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	user.CreatedAt = time.Now().UTC()
	s.data[user.ID] = user
	return user, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[id]
	if !ok {
		return model.User{}, ErrNotFound
	}
	return val, nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.data {
		if user.Email == email {
			return user, nil
		}
	}
	return model.User{}, ErrNotFound
}

func (s *UserStore) List(ctx context.Context) ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.User, 0, len(s.data))
	for _, user := range s.data {
		out = append(out, user)
	}
	return out, nil
}

func (s *UserStore) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.data[id]
	if !ok {
		return ErrNotFound
	}
	user.PasswordHash = passwordHash
	s.data[id] = user
	return nil
}

func (s *UserStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[id]; !ok {
		return ErrNotFound
	}
	delete(s.data, id)
	return nil
}

type APITokenStore struct {
	mu        sync.RWMutex
	data      map[string]model.APIToken
	hashIndex map[string]string // tokenHash -> id
}

func NewAPITokenStore() *APITokenStore {
	return &APITokenStore{
		data:      make(map[string]model.APIToken),
		hashIndex: make(map[string]string),
	}
}

func (s *APITokenStore) Create(ctx context.Context, token model.APIToken) (model.APIToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token.ID == "" {
		token.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	token.CreatedAt = now
	token.UpdatedAt = now
	s.data[token.ID] = token
	s.hashIndex[token.TokenHash] = token.ID
	return token, nil
}

func (s *APITokenStore) GetByID(ctx context.Context, id string) (model.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[id]
	if !ok {
		return model.APIToken{}, ErrNotFound
	}
	return val, nil
}

func (s *APITokenStore) GetByTokenHash(ctx context.Context, tokenHash string) (model.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.hashIndex[tokenHash]
	if !ok {
		return model.APIToken{}, ErrNotFound
	}
	val, ok := s.data[id]
	if !ok {
		return model.APIToken{}, ErrNotFound
	}
	return val, nil
}

func (s *APITokenStore) ListByUser(ctx context.Context, userID string) ([]model.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.APIToken, 0)
	for _, token := range s.data {
		if token.UserID == userID {
			out = append(out, token)
		}
	}
	return out, nil
}

func (s *APITokenStore) Update(ctx context.Context, token model.APIToken) (model.APIToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[token.ID]; !ok {
		return model.APIToken{}, ErrNotFound
	}
	token.UpdatedAt = time.Now().UTC()
	s.data[token.ID] = token
	return token, nil
}

func (s *APITokenStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.data[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.hashIndex, token.TokenHash)
	delete(s.data, id)
	return nil
}
