package service

import (
	"thmanyah.com/content-platform/internal/cms/auth"
	"thmanyah.com/content-platform/internal/cms/port"
)

type Service struct {
	programRepo  port.ProgramRepo
	categoryRepo port.CategoryRepo
	sourceRepo   port.SourceRepo
	outboxRepo   port.OutboxRepo
	userRepo     port.UserRepo
	sessionRepo  port.SessionRepo
	jwtManager   *auth.JWTManager
}

func NewService(
	programRepo port.ProgramRepo,
	categoryRepo port.CategoryRepo,
	sourceRepo port.SourceRepo,
	outboxRepo port.OutboxRepo,
	userRepo port.UserRepo,
	sessionRepo port.SessionRepo,
	jwtManager *auth.JWTManager,
) *Service {
	return &Service{
		programRepo:  programRepo,
		categoryRepo: categoryRepo,
		sourceRepo:   sourceRepo,
		outboxRepo:   outboxRepo,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		jwtManager:   jwtManager,
	}
}
