package service

import (
	"context"
	"errors"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

func (s *Service) CreateUser(ctx context.Context, params sqlc.CreateUserParams) (*domain.User, error) {
	return s.userRepo.Create(ctx, params)
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

type AuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (s *Service) Login(ctx context.Context, email, password string, passwordCheck func(hash, password string) error) (*domain.User, *AuthTokens, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := passwordCheck(user.PasswordHash, password); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	sessionID, _, err := s.sessionRepo.CreateSession(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(sessionID, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetAccessTokenTTL().Seconds()),
	}, nil
}

func (s *Service) Register(ctx context.Context, params sqlc.CreateUserParams) (*domain.User, *AuthTokens, error) {
	user, err := s.userRepo.Create(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	sessionID, _, err := s.sessionRepo.CreateSession(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(sessionID, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.jwtManager.GetAccessTokenTTL().Seconds()),
	}, nil
}

func (s *Service) RefreshAccessToken(ctx context.Context, sessionID, userID string) (string, error) {
	exists, err := s.sessionRepo.SessionExists(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", errors.New("invalid session")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	return s.jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.DeleteSession(ctx, sessionID)
}
