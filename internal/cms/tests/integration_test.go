//go:build integration

package cmstest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/hadialqattan/mediacms/config"
	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/handler"
	"github.com/hadialqattan/mediacms/internal/cms/repository"
	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/cms/router"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type testSuite struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
	svc         *service.Service
	testRouter  *chi.Mux
	cleanupFunc func()
}

func setupTestSuite(t *testing.T) *testSuite {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable")
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		pool.Close()
		t.Skipf("Redis not available: %v", err)
		return nil
	}

	cleanupTestData(ctx, pool)
	redisClient.FlushDB(ctx)

	programRepo := repository.NewProgramRepo(pool)
	outboxRepo := repository.NewOutboxRepo(pool)
	userRepo := repository.NewUserRepo(pool)

	cfg := config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
	}

	sessionRepo := repository.NewSessionRepo(redisClient, cfg)
	jwtManager := auth.NewJWTManager(cfg)
	svc := service.NewService(programRepo, outboxRepo, userRepo, sessionRepo, jwtManager, pool)
	testRouter := router.NewRouter(svc, cfg)

	return &testSuite{
		pool:        pool,
		redisClient: redisClient,
		svc:         svc,
		testRouter:  testRouter,
		cleanupFunc: func() {
			cleanupTestData(ctx, pool)
			redisClient.FlushDB(ctx)
			pool.Close()
			redisClient.Close()
		},
	}
}

func createTestUser(t *testing.T, svc *service.Service, email, password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user, err := svc.CreateUser(context.Background(), sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
		Role:         string(domain.UserRoleEditor),
	})
	require.NoError(t, err)
	return user.ID
}

func loginUser(t *testing.T, testRouter http.Handler, email, password string) *handler.LoginResponse {
	loginReq := handler.LoginRequest{Email: email, Password: password}
	body, _ := json.Marshal(loginReq)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var loginResp handler.LoginResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	require.NotEmpty(t, loginResp.AccessToken)
	require.NotEmpty(t, loginResp.RefreshToken)

	return &loginResp
}

func createTestProgram(t *testing.T, testRouter http.Handler, accessToken, slug string, tags []string) map[string]interface{} {
	createReq := map[string]interface{}{
		"slug":        slug,
		"title":       "Test Program",
		"description": "A test program",
		"type":        "podcast",
		"language":    "en",
		"duration_ms": 3600000,
		"tags":        tags,
	}
	body, _ := json.Marshal(createReq)

	w := makeRequest("POST", "/api/v1/programs", body, accessToken, testRouter)
	require.Equal(t, http.StatusCreated, w.Code)

	var program map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &program))
	return program
}

func cleanupTestData(ctx context.Context, pool *pgxpool.Pool) {
	// TODO: Use testing containers and remove this function.
	pool.Exec(ctx, "DELETE FROM outbox_events")
	pool.Exec(ctx, "DELETE FROM programs")
	pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'test-%@example.com'")
}

func makeRequest(method, path string, body []byte, accessToken string, testRouter http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func TestHealthEndpoint(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("GET /health - Health check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		suite.testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "ok", response["status"])
		assert.Equal(t, "cms-api", response["service"])
		assert.NotNil(t, response["timestamp"])
	})
}

func TestAuthEndpoints(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("POST /api/v1/auth/login - Valid credentials", func(t *testing.T) {
		email := "test-user-1-auth@example.com"
		password := "password123"
		createTestUser(t, suite.svc, email, password)

		loginResp := loginUser(t, suite.testRouter, email, password)
		assert.NotEmpty(t, loginResp.AccessToken)
		assert.NotEmpty(t, loginResp.RefreshToken)
	})

	t.Run("POST /api/v1/auth/login - Invalid credentials", func(t *testing.T) {
		loginReq := handler.LoginRequest{Email: "nonexistent@example.com", Password: "wrongpassword"}
		body, _ := json.Marshal(loginReq)

		w := makeRequest("POST", "/api/v1/auth/login", body, "", suite.testRouter)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST /api/v1/auth/refresh - Valid refresh token", func(t *testing.T) {
		email := "test-user-4-auth@example.com"
		password := "password123"
		createTestUser(t, suite.svc, email, password)
		loginResp := loginUser(t, suite.testRouter, email, password)

		refreshReq := map[string]string{"refresh_token": loginResp.RefreshToken}
		body, _ := json.Marshal(refreshReq)

		w := makeRequest("POST", "/api/v1/auth/refresh", body, "", suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var refreshResp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &refreshResp))
		accessToken, ok := refreshResp["access_token"].(string)
		assert.True(t, ok && accessToken != "")
	})

	t.Run("POST /api/v1/auth/refresh - Invalid refresh token", func(t *testing.T) {
		refreshReq := map[string]string{"refresh_token": "invalid-token"}
		body, _ := json.Marshal(refreshReq)

		w := makeRequest("POST", "/api/v1/auth/refresh", body, "", suite.testRouter)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST /api/v1/auth/logout - Valid logout", func(t *testing.T) {
		email := "test-user-6-auth@example.com"
		password := "password123"
		createTestUser(t, suite.svc, email, password)
		loginResp := loginUser(t, suite.testRouter, email, password)

		logoutReq := map[string]string{"refresh_token": loginResp.RefreshToken}
		body, _ := json.Marshal(logoutReq)

		w := makeRequest("POST", "/api/v1/auth/logout", body, "", suite.testRouter)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("POST /api/v1/auth/refresh after logout - Should fail", func(t *testing.T) {
		email := "test-user-7-auth@example.com"
		password := "password123"
		createTestUser(t, suite.svc, email, password)
		loginResp := loginUser(t, suite.testRouter, email, password)

		logoutReq := map[string]string{"refresh_token": loginResp.RefreshToken}
		logoutBody, _ := json.Marshal(logoutReq)
		makeRequest("POST", "/api/v1/auth/logout", logoutBody, "", suite.testRouter)

		refreshReq := map[string]string{"refresh_token": loginResp.RefreshToken}
		body, _ := json.Marshal(refreshReq)

		w := makeRequest("POST", "/api/v1/auth/refresh", body, "", suite.testRouter)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestProgramCRUD(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("POST /api/v1/programs - Create program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-crud@example.com", "password123")

		slug := "test-program-crud-1"
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug, []string{"news", "politics"})

		assert.Equal(t, slug, program["slug"])
		assert.Equal(t, "Test Program", program["title"])
		assert.NotEmpty(t, program["id"])
		assert.NotEmpty(t, program["tags"])
	})

	t.Run("GET /api/v1/programs - List programs", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-crud@example.com", "password123")
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-crud-2", []string{"tech"})

		w := makeRequest("GET", "/api/v1/programs", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(data), 1)
	})

	t.Run("GET /api/v1/programs/{id} - Get program by UUID", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-3-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-3-crud@example.com", "password123")
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-crud-3", []string{"sports"})

		programID := program["id"].(string)

		w := makeRequest("GET", "/api/v1/programs/"+programID, nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "test-program-crud-3", response["slug"])
		assert.NotEmpty(t, response["created_by"])
	})

	t.Run("PUT /api/v1/programs/{id} - Update program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-4-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-4-crud@example.com", "password123")
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-crud-4", []string{"music"})
		programID := program["id"].(string)

		updateReq := map[string]interface{}{"title": "Updated Title", "description": "Updated description", "tags": []string{"music", "jazz"}}
		body, _ := json.Marshal(updateReq)

		w := makeRequest("PUT", "/api/v1/programs/"+programID, body, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var updatedProgram map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updatedProgram))
		assert.Equal(t, "Updated Title", updatedProgram["title"])
	})

	t.Run("DELETE /api/v1/programs/{id} - Delete unpublished program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-6-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-6-crud@example.com", "password123")
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-crud-6-delete", []string{"health"})
		programID := program["id"].(string)

		w := makeRequest("DELETE", "/api/v1/programs/"+programID, nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusNoContent, w.Code)

		var eventCount int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE program_id = $1", programID).Scan(&eventCount)
		assert.Equal(t, 0, eventCount)
	})

	t.Run("DELETE /api/v1/programs/{id} - Delete published program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-7-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-7-crud@example.com", "password123")
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-crud-7-delete-published", []string{"nature"})
		programID := program["id"].(string)

		makeRequest("POST", "/api/v1/programs/"+programID+"/publish", nil, loginResp.AccessToken, suite.testRouter)

		w := makeRequest("DELETE", "/api/v1/programs/"+programID, nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusNoContent, w.Code)

		var eventCount int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE program_id = $1 AND type = 'program.delete'", programID).Scan(&eventCount)
		assert.Equal(t, 1, eventCount)
	})
}

func TestProgramPublishing(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("PATCH /api/v1/programs/{id}/publish - Publish program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-publish@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-publish@example.com", "password123")
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-publish-1", []string{"business"})
		programID := program["id"].(string)

		w := makeRequest("POST", "/api/v1/programs/"+programID+"/publish", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var publishedProgram map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &publishedProgram))
		assert.NotNil(t, publishedProgram["published_at"])

		var eventCount int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE type = 'program.upsert' AND program_id = $1", programID).Scan(&eventCount)
		assert.Greater(t, eventCount, 0)
	})

	t.Run("PATCH /api/v1/programs/{id}/publish - Republish", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-publish@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-publish@example.com", "password123")
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-publish-2", []string{"finance"})
		programID := program["id"].(string)

		makeRequest("POST", "/api/v1/programs/"+programID+"/publish", nil, loginResp.AccessToken, suite.testRouter)

		var eventCountFirst int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE type = 'program.upsert' AND program_id = $1", programID).Scan(&eventCountFirst)

		makeRequest("POST", "/api/v1/programs/"+programID+"/publish", nil, loginResp.AccessToken, suite.testRouter)

		var eventCountSecond int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE type = 'program.upsert' AND program_id = $1", programID).Scan(&eventCountSecond)

		assert.Equal(t, eventCountFirst+1, eventCountSecond)
	})
}

func TestProgramPagination(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("GET /api/v1/programs?page=1&limit=5 - Paginated list", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-pagination@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-pagination@example.com", "password123")

		for i := 0; i < 10; i++ {
			createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-pagination-"+strconv.Itoa(i), []string{"test"})
		}

		w := makeRequest("GET", "/api/v1/programs?page=1&limit=5", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		data, ok := response["data"].([]interface{})
		assert.True(t, ok)
		assert.LessOrEqual(t, len(data), 5)

		meta := response["meta"].(map[string]interface{})
		assert.Equal(t, float64(1), meta["page"])
		assert.Equal(t, float64(5), meta["limit"])
	})
}

func TestBulkOperations(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("POST /api/v1/programs/bulk - Bulk create programs", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-bulk@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-bulk@example.com", "password123")

		bulkReq := map[string]interface{}{
			"programs": []map[string]interface{}{
				{
					"slug":        "bulk-program-1",
					"title":       "Bulk Program 1",
					"description": "First bulk program",
					"type":        "podcast",
					"language":    "en",
					"duration_ms": 3600000,
					"tags":        []string{"news"},
				},
				{
					"slug":        "bulk-program-2",
					"title":       "Bulk Program 2",
					"description": "Second bulk program",
					"type":        "podcast",
					"language":    "en",
					"duration_ms": 3600000,
					"tags":        []string{"sports"},
				},
			},
		}
		body, _ := json.Marshal(bulkReq)

		w := makeRequest("POST", "/api/v1/programs/bulk", body, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusMultiStatus, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		created := response["created"].([]interface{})
		assert.Equal(t, 2, len(created))
	})

}
