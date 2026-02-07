//go:build integration

package e2etest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"golang.org/x/crypto/bcrypt"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/handler"
	cmsrepo "github.com/hadialqattan/mediacms/internal/cms/repository"
	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	cmsrouter "github.com/hadialqattan/mediacms/internal/cms/router"
	cmsservice "github.com/hadialqattan/mediacms/internal/cms/service"
	cmsdiscovery "github.com/hadialqattan/mediacms/internal/discovery"
	discoveryrepo "github.com/hadialqattan/mediacms/internal/discovery/repository"
	discoveryrouter "github.com/hadialqattan/mediacms/internal/discovery/router"
	"github.com/hadialqattan/mediacms/internal/outboxrelay"
	outboxrepo "github.com/hadialqattan/mediacms/internal/outboxrelay/repository"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type e2eTestSuite struct {
	pool            *pgxpool.Pool
	redisClient     *redis.Client
	typesenseClient *typesense.Client
	cmsRouter       http.Handler
	discoveryRouter http.Handler
	relay           *outboxrelay.Relay
	cmsService      *cmsservice.Service
}

func TestFullCMSToDiscoveryFlow(t *testing.T) {
	suite := setupE2ETest(t)
	if suite == nil {
		return
	}

	ctx := context.Background()
	relayCtx, relayCancel := context.WithTimeout(ctx, 10*time.Second)
	defer relayCancel()
	go suite.relay.Start(relayCtx)

	email := "e2e-test-user@example.com"
	userID := createE2ETestUser(t, suite.cmsService, email, "password123")
	loginResp := loginE2EUser(t, suite.cmsRouter, email, "password123")

	programSlug := fmt.Sprintf("test-e2e-%s", uuid.New().String()[:8])
	programID := createE2EProgram(t, suite.cmsRouter, loginResp.AccessToken, programSlug)

	publishE2EProgram(t, suite.cmsRouter, loginResp.AccessToken, programSlug, programID)
	verifyOutboxEvent(t, suite.pool, programID)
	waitForOutboxEnqueue(t, suite.pool, programID)
	searchAndVerifyProgram(t, suite.discoveryRouter, programID)

	t.Log("Cleaning up test data...")
	_, _ = suite.typesenseClient.Collection("programs").Document(programID).Delete(ctx)

	suite.pool.Exec(ctx, "DELETE FROM outbox_events WHERE program_id = $1", programID)
	suite.pool.Exec(ctx, "DELETE FROM categorized_as WHERE program_id = $1", programID)
	suite.pool.Exec(ctx, "DELETE FROM programs WHERE id = $1", programID)
	suite.pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)

	t.Log("Full CMS to Discovery flow test completed successfully!")
}

func setupE2ETest(t *testing.T) *e2eTestSuite {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable")
	if err != nil {
		t.Skipf("Database not available, skipping integration test: %v", err)
		return nil
	}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		pool.Close()
		t.Skipf("Redis not available, skipping integration test: %v", err)
		return nil
	}

	typesenseClient := typesense.NewClient(
		typesense.WithServer("http://localhost:8108"),
		typesense.WithAPIKey("xyz"),
	)

	pool.Exec(ctx, "DELETE FROM outbox_events WHERE program_id IN (SELECT id FROM programs WHERE slug LIKE 'test-e2e-%')")
	pool.Exec(ctx, "DELETE FROM categorized_as WHERE program_id IN (SELECT id FROM programs WHERE slug LIKE 'test-e2e-%')")
	pool.Exec(ctx, "DELETE FROM programs WHERE slug LIKE 'test-e2e-%'")
	pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'e2e-test%@example.com'")
	redisClient.FlushAll(ctx)

	schema := &api.CollectionSchema{
		Name: "programs",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "slug", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "description", Type: "string"},
			{Name: "type", Type: "string"},
			{Name: "language", Type: "string"},
			{Name: "duration_ms", Type: "int32"},
			{Name: "categories", Type: "string[]"},
			{Name: "published_at", Type: "int64"},
		},
	}
	_, _ = typesenseClient.Collections().Create(ctx, schema)

	programRepo := cmsrepo.NewProgramRepo(pool)
	categoryRepo := cmsrepo.NewCategoryRepo(pool)
	sourceRepo := cmsrepo.NewSourceRepo(pool)
	cmsOutboxRepo := cmsrepo.NewOutboxRepo(pool)
	userRepo := cmsrepo.NewUserRepo(pool)

	cfg := config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
	}

	sessionRepo := cmsrepo.NewSessionRepo(redisClient, cfg)
	jwtManager := auth.NewJWTManager(cfg)

	cmsService := cmsservice.NewService(programRepo, categoryRepo, sourceRepo, cmsOutboxRepo, userRepo, sessionRepo, jwtManager, pool)
	cmsRouter := cmsrouter.NewRouter(cmsService, cfg)

	searchIndex := discoveryrepo.NewSearchIndex(typesenseClient)
	discoveryService := cmsdiscovery.NewService(searchIndex)
	discoveryRouter := discoveryrouter.NewRouter(discoveryService)

	asynqClient := outboxrepo.NewQueue("localhost:6379")
	outboxRelayRepo := outboxrepo.NewOutboxRepo(pool)
	relay := outboxrelay.NewRelay(outboxRelayRepo, asynqClient, 100*time.Millisecond)

	suite := &e2eTestSuite{
		pool:            pool,
		redisClient:     redisClient,
		typesenseClient: typesenseClient,
		cmsRouter:       cmsRouter,
		discoveryRouter: discoveryRouter,
		relay:           relay,
		cmsService:      cmsService,
	}

	t.Cleanup(func() {
		pool.Close()
		redisClient.Close()
	})

	return suite
}

func createE2ETestUser(t *testing.T, svc *cmsservice.Service, email, password string) string {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user, err := svc.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
		Role:         string(domain.UserRoleEditor),
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	t.Logf("Created user: %s", user.ID)
	return user.ID
}

func loginE2EUser(t *testing.T, router http.Handler, email, password string) *handler.LoginResponse {
	loginReq := handler.LoginRequest{Email: email, Password: password}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReqHTTP)

	if loginW.Code != http.StatusOK {
		t.Fatalf("Failed to login: %d: %s", loginW.Code, loginW.Body.String())
	}

	var loginResp handler.LoginResponse
	json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	t.Logf("Logged in, got access token: %s...", loginResp.AccessToken[:20])
	return &loginResp
}

func createE2EProgram(t *testing.T, router http.Handler, accessToken, slug string) string {
	createReq := handler.CreateProgramRequest{
		Slug:        slug,
		Title:       "E2E Test Podcast",
		Description: "A podcast for end-to-end testing",
		Type:        "podcast",
		Language:    "en",
		DurationMs:  3600000,
	}
	createBody, _ := json.Marshal(createReq)
	createReqHTTP := httptest.NewRequest("POST", "/api/v1/programs", bytes.NewReader(createBody))
	createReqHTTP.Header.Set("Authorization", "Bearer "+accessToken)
	createW := httptest.NewRecorder()
	router.ServeHTTP(createW, createReqHTTP)

	if createW.Code != http.StatusCreated {
		t.Fatalf("Failed to create program: %d: %s", createW.Code, createW.Body.String())
	}

	var createdProgram map[string]interface{}
	json.Unmarshal(createW.Body.Bytes(), &createdProgram)
	programID := createdProgram["id"].(string)
	t.Logf("Created program: %s (slug: %s)", programID, slug)
	return programID
}

func publishE2EProgram(t *testing.T, router http.Handler, accessToken, slug, programID string) {
	publishReqHTTP := httptest.NewRequest("POST", "/api/v1/programs/"+slug+"/publish", nil)
	publishReqHTTP.Header.Set("Authorization", "Bearer "+accessToken)
	publishW := httptest.NewRecorder()
	router.ServeHTTP(publishW, publishReqHTTP)

	if publishW.Code != http.StatusOK {
		t.Fatalf("Failed to publish program: %d: %s", publishW.Code, publishW.Body.String())
	}
	t.Logf("Published program: %s", programID)
}

func verifyOutboxEvent(t *testing.T, pool *pgxpool.Pool, programID string) int {
	ctx := context.Background()
	var outboxEventCount int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE program_id = $1", programID).Scan(&outboxEventCount)
	if err != nil {
		t.Fatalf("Failed to query outbox events: %v", err)
	}

	if outboxEventCount == 0 {
		t.Fatal("Expected outbox event to be created")
	}
	t.Logf("Outbox events created: %d", outboxEventCount)
	return outboxEventCount
}

func waitForOutboxEnqueue(t *testing.T, pool *pgxpool.Pool, programID string) {
	ctx := context.Background()
	t.Log("Waiting for outbox relay to process events...")
	time.Sleep(3 * time.Second)

	var enqueuedCount int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events WHERE program_id = $1 AND enqueued = true", programID).Scan(&enqueuedCount)
	if err != nil {
		t.Fatalf("Failed to query enqueued outbox events: %v", err)
	}

	if enqueuedCount == 0 {
		t.Fatal("Expected outbox event to be enqueued")
	}
	t.Logf("Outbox events enqueued: %d", enqueuedCount)
}

func searchAndVerifyProgram(t *testing.T, router http.Handler, programID string) {
	t.Log("Waiting for search indexer to process...")
	time.Sleep(3 * time.Second)

	searchReq := httptest.NewRequest("GET", "/api/v1/programs/search?q=E2E", nil)
	searchW := httptest.NewRecorder()
	router.ServeHTTP(searchW, searchReq)

	if searchW.Code != http.StatusOK {
		t.Fatalf("Search failed: %d: %s", searchW.Code, searchW.Body.String())
	}

	var searchResult map[string]interface{}
	json.Unmarshal(searchW.Body.Bytes(), &searchResult)
	programs := searchResult["programs"].([]interface{})

	if len(programs) == 0 {
		t.Fatal("Expected to find program in search results")
	}
	t.Logf("Found %d programs in search", len(programs))

	getReq := httptest.NewRequest("GET", "/api/v1/programs/"+programID, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("Failed to get program by ID: %d: %s", getW.Code, getW.Body.String())
	}

	var program map[string]interface{}
	json.Unmarshal(getW.Body.Bytes(), &program)

	if program["id"] != programID {
		t.Fatalf("Expected program ID %s, got %v", programID, program["id"])
	}
	t.Logf("Successfully retrieved program from Discovery: %v", program["title"])
}
