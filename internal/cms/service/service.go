package service

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/port"
)

type Service struct {
	programRepo port.ProgramRepo
	outboxRepo   port.OutboxRepo
	userRepo     port.UserRepo
	sessionRepo  port.SessionRepo
	jwtManager   *auth.JWTManager
	pool         *pgxpool.Pool
}

func NewService(
	programRepo port.ProgramRepo,
	outboxRepo port.OutboxRepo,
	userRepo port.UserRepo,
	sessionRepo port.SessionRepo,
	jwtManager *auth.JWTManager,
	pool *pgxpool.Pool,
) *Service {
	return &Service{
		programRepo: programRepo,
		outboxRepo:  outboxRepo,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		jwtManager:  jwtManager,
		pool:        pool,
	}
}
