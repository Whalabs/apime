package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/open-apime/apime/internal/storage"
	"github.com/open-apime/apime/internal/storage/model"
)

var (
	ErrInvalidEmail            = errors.New("email inválido")
	ErrInvalidPassword         = errors.New("senha deve ter pelo menos 8 caracteres")
	ErrInvalidRole             = errors.New("papel inválido")
	ErrDeleteLastAdmin         = errors.New("não é possível remover o último administrador")
	ErrTokenServiceUnavailable = errors.New("serviço de tokens indisponível")
)

type TokenManager interface {
	Create(ctx context.Context, userID, name string, expiresAt *time.Time) (model.APIToken, string, error)
}

type InstanceManager interface {
	ListByUser(ctx context.Context, userID string, userRole string) ([]model.Instance, error)
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repo         storage.UserRepository
	tokenService TokenManager
	instanceSvc  InstanceManager
}

func NewService(repo storage.UserRepository, tokenService TokenManager, instanceSvc InstanceManager) *Service {
	return &Service{
		repo:         repo,
		tokenService: tokenService,
		instanceSvc:  instanceSvc,
	}
}

type CreateInput struct {
	Email    string
	Password string
	Role     string
}

func (s *Service) List(ctx context.Context) ([]model.User, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Exists(ctx context.Context, id string) (bool, error) {
	_, err := s.repo.GetByID(ctx, id)
	if err == nil {
		return true, nil
	}
	if err == storage.ErrNotFound {
		return false, nil
	}
	return false, err
}

func (s *Service) Create(ctx context.Context, input CreateInput) (model.User, string, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	if email == "" || !strings.Contains(email, "@") {
		return model.User{}, "", ErrInvalidEmail
	}
	if len(input.Password) < 8 {
		return model.User{}, "", ErrInvalidPassword
	}
	role := strings.TrimSpace(input.Role)
	if role == "" {
		role = "admin"
	}

	hash, err := hashPassword(input.Password)
	if err != nil {
		return model.User{}, "", err
	}

	user := model.User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	}

	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		return model.User{}, "", err
	}

	var tokenString string
	if s.tokenService != nil {
		// Gera um token padrão para facilitar o primeiro acesso do usuário.
		_, tokenString, err = s.tokenService.Create(ctx, createdUser.ID, "Default Token", nil)
		if err != nil {
			// Se falhar criar token, prosseguimos, mas sem retornar token.
			// O usuário pode criar depois via dashboard.
		}
	}

	return createdUser, tokenString, nil
}

func (s *Service) UpdatePassword(ctx context.Context, id, password string) error {
	if len(password) < 8 {
		return ErrInvalidPassword
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, id, hash)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if user.Role == "admin" {
		users, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		adminCount := 0
		for _, u := range users {
			if u.Role == "admin" {
				adminCount++
			}
		}
		if adminCount <= 1 {
			return ErrDeleteLastAdmin
		}
	}

	if s.instanceSvc != nil {
		instances, err := s.instanceSvc.ListByUser(ctx, id, "user")
		if err == nil {
			for _, inst := range instances {
				_ = s.instanceSvc.Delete(ctx, inst.ID)
			}
		}
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) RotateAPIToken(ctx context.Context, id string) (string, error) {
	if s.tokenService == nil {
		return "", ErrTokenServiceUnavailable
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return "", err
	}
	_, token, err := s.tokenService.Create(ctx, id, "Rotated Token", nil)
	if err != nil {
		return "", err
	}
	return token, nil
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
