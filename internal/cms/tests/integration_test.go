//go:build integration

package cmstest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"thmanyah.com/content-platform/config"
	"thmanyah.com/content-platform/internal/cms/auth"
	"thmanyah.com/content-platform/internal/cms/handler"
	"thmanyah.com/content-platform/internal/cms/port"
	"thmanyah.com/content-platform/internal/cms/repository"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/cms/router"
	"thmanyah.com/content-platform/internal/cms/service"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type testSuite struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
	svc         *service.Service
	sourceRepo  port.SourceRepo
	testRouter  *chi.Mux
	cleanupFunc func()
}

func setupTestSuite(t *testing.T) *testSuite {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/thmanyah?sslmode=disable")
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
	categoryRepo := repository.NewCategoryRepo(pool)
	sourceRepo := repository.NewSourceRepo(pool)
	outboxRepo := repository.NewOutboxRepo(pool)
	userRepo := repository.NewUserRepo(pool)

	cfg := config.JWTConfig{
		Secret:          "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 720 * time.Hour,
	}

	sessionRepo := repository.NewSessionRepo(redisClient, cfg)
	jwtManager := auth.NewJWTManager(cfg)
	svc := service.NewService(programRepo, categoryRepo, sourceRepo, outboxRepo, userRepo, sessionRepo, jwtManager, pool)
	testRouter := router.NewRouter(svc, cfg)

	return &testSuite{
		pool:        pool,
		redisClient: redisClient,
		svc:         svc,
		sourceRepo:  sourceRepo,
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

func createTestCategory(t *testing.T, testRouter http.Handler, accessToken, name, description string) map[string]interface{} {
	createReq := handler.CreateCategoryRequest{Name: name, Description: description}
	body, _ := json.Marshal(createReq)

	w := makeRequest("POST", "/api/v1/categories", body, accessToken, testRouter)
	require.Equal(t, http.StatusCreated, w.Code)

	var category map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &category))
	return category
}

func createTestProgram(t *testing.T, testRouter http.Handler, accessToken, slug string) map[string]interface{} {
	createReq := handler.CreateProgramRequest{
		Slug:        slug,
		Title:       "Test Program",
		Description: "A test program",
		Type:        "podcast",
		Language:    "en",
		DurationMs:  3600000,
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
	pool.Exec(ctx, "DELETE FROM categorized_as")
	pool.Exec(ctx, "DELETE FROM programs")
	pool.Exec(ctx, "DELETE FROM categories")
	pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'test-%@example.com'")
	pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'editor-test%@example.com'")
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

	t.Run("POST /api/v1/auth/register - New user registration", func(t *testing.T) {
		registerReq := handler.LoginRequest{Email: "test-user-3-auth@example.com", Password: "newpassword123"}
		body, _ := json.Marshal(registerReq)

		w := makeRequest("POST", "/api/v1/auth/register", body, "", suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var registerResp handler.LoginResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &registerResp))
		assert.NotEmpty(t, registerResp.AccessToken)
		assert.NotEmpty(t, registerResp.RefreshToken)
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
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)

		assert.Equal(t, slug, program["slug"])
		assert.Equal(t, "Test Program", program["title"])
		assert.NotEmpty(t, program["id"])
	})

	t.Run("GET /api/v1/programs - List programs", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-crud@example.com", "password123")
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, "test-program-crud-2")

		w := makeRequest("GET", "/api/v1/programs", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var programs []map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &programs))
		assert.GreaterOrEqual(t, len(programs), 1)
	})

	t.Run("GET /api/v1/programs/{slug} - Get program by slug", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-3-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-3-crud@example.com", "password123")
		slug := "test-program-crud-3"
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)

		w := makeRequest("GET", "/api/v1/programs/"+slug, nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var program map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &program))
		assert.Equal(t, slug, program["slug"])
	})

	t.Run("PUT /api/v1/programs/{slug} - Update program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-4-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-4-crud@example.com", "password123")
		slug := "test-program-crud-4"
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)

		updateReq := map[string]interface{}{"title": "Updated Title", "description": "Updated description"}
		body, _ := json.Marshal(updateReq)

		w := makeRequest("PUT", "/api/v1/programs/"+slug, body, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var program map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &program))
		assert.Equal(t, "Updated Title", program["title"])
	})

	t.Run("DELETE /api/v1/programs/{slug} - Delete unpublished program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-5-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-5-crud@example.com", "password123")
		slug := "test-program-crud-5-delete"
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)
		programID := program["id"].(string)

		w := makeRequest("DELETE", "/api/v1/programs/"+slug, nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusNoContent, w.Code)

		var eventCount int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE program_id = $1", programID).Scan(&eventCount)
		assert.Equal(t, 0, eventCount)
	})

	t.Run("DELETE /api/v1/programs/{slug} - Delete published program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-6-crud@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-6-crud@example.com", "password123")
		slug := "test-program-crud-6-delete-published"
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)
		programID := program["id"].(string)

		makeRequest("POST", "/api/v1/programs/"+slug+"/publish", nil, loginResp.AccessToken, suite.testRouter)

		w := makeRequest("DELETE", "/api/v1/programs/"+slug, nil, loginResp.AccessToken, suite.testRouter)
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

	t.Run("POST /api/v1/programs/{slug}/publish - Publish program", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-publish@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-publish@example.com", "password123")
		slug := "test-program-publish-1"
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)
		programID := program["id"].(string)

		w := makeRequest("POST", "/api/v1/programs/"+slug+"/publish", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var publishedProgram map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &publishedProgram))
		assert.NotNil(t, publishedProgram["published_at"])

		var eventCount int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE type = 'program.upsert' AND program_id = $1", programID).Scan(&eventCount)
		assert.Greater(t, eventCount, 0)
	})

	t.Run("POST /api/v1/programs/{slug}/publish - Republish", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-publish@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-publish@example.com", "password123")
		slug := "test-program-publish-2"
		program := createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)
		programID := program["id"].(string)

		makeRequest("POST", "/api/v1/programs/"+slug+"/publish", nil, loginResp.AccessToken, suite.testRouter)

		var eventCountFirst int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE type = 'program.upsert' AND program_id = $1", programID).Scan(&eventCountFirst)

		makeRequest("POST", "/api/v1/programs/"+slug+"/publish", nil, loginResp.AccessToken, suite.testRouter)

		var eventCountSecond int
		suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE type = 'program.upsert' AND program_id = $1", programID).Scan(&eventCountSecond)

		assert.Equal(t, eventCountFirst+1, eventCountSecond)
	})
}

func TestCategoryEndpoints(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("POST /api/v1/categories - Create category", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-category@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-category@example.com", "password123")

		category := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-1", "Test category description")

		assert.Equal(t, "test-category-1", category["Name"])
		assert.NotNil(t, category["ID"])
	})

	t.Run("GET /api/v1/categories - List categories", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-category@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-category@example.com", "password123")

		createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-2-a", "Category A")
		createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-2-b", "Category B")

		w := makeRequest("GET", "/api/v1/categories", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var categories []map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &categories))
		assert.GreaterOrEqual(t, len(categories), 2)
	})

	t.Run("GET /api/v1/categories/{id} - Get category by ID", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-3-category@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-3-category@example.com", "password123")

		category := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-3", "Test category 3")
		categoryID := category["ID"].(string)

		w := makeRequest("GET", "/api/v1/categories/"+categoryID, nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var fetchedCategory map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fetchedCategory))
		assert.Equal(t, "test-category-3", fetchedCategory["Name"])
	})
}

func TestProgramCategoryAssignment(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("PUT /api/v1/programs/{slug}/categories - Assign categories", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-1-assign@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-assign@example.com", "password123")

		category1 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-1", "Category 1")
		category2 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-2", "Category 2")

		slug := "test-program-assign-1"
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)

		assignReq := handler.AssignCategoriesRequest{
			CategoryIDs: []string{category1["ID"].(string), category2["ID"].(string)},
		}
		body, _ := json.Marshal(assignReq)

		w := makeRequest("PUT", "/api/v1/programs/"+slug+"/categories", body, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("GET /api/v1/programs/{slug}/categories - Get assigned categories", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-assign@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-assign@example.com", "password123")

		category1 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-3", "Category 3")
		category2 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-4", "Category 4")

		slug := "test-program-assign-2"
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)

		assignReq := handler.AssignCategoriesRequest{
			CategoryIDs: []string{category1["ID"].(string), category2["ID"].(string)},
		}
		body, _ := json.Marshal(assignReq)
		makeRequest("PUT", "/api/v1/programs/"+slug+"/categories", body, loginResp.AccessToken, suite.testRouter)

		w := makeRequest("GET", "/api/v1/programs/"+slug+"/categories", nil, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusOK, w.Code)

		var categories []map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &categories))
		assert.Equal(t, 2, len(categories))
	})

	t.Run("PUT /api/v1/programs/{slug}/categories - Update categories", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-3-assign@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-3-assign@example.com", "password123")

		category1 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-5", "Category 5")
		category2 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-6", "Category 6")
		category3 := createTestCategory(t, suite.testRouter, loginResp.AccessToken, "test-category-assign-7", "Category 7")

		slug := "test-program-assign-3"
		createTestProgram(t, suite.testRouter, loginResp.AccessToken, slug)

		assignReq1 := handler.AssignCategoriesRequest{
			CategoryIDs: []string{category1["ID"].(string), category2["ID"].(string)},
		}
		body1, _ := json.Marshal(assignReq1)
		makeRequest("PUT", "/api/v1/programs/"+slug+"/categories", body1, loginResp.AccessToken, suite.testRouter)

		assignReq2 := handler.AssignCategoriesRequest{
			CategoryIDs: []string{category3["ID"].(string)},
		}
		body2, _ := json.Marshal(assignReq2)
		makeRequest("PUT", "/api/v1/programs/"+slug+"/categories", body2, loginResp.AccessToken, suite.testRouter)

		w := makeRequest("GET", "/api/v1/programs/"+slug+"/categories", nil, loginResp.AccessToken, suite.testRouter)

		var categories []map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &categories))
		assert.Equal(t, 3, len(categories))
	})
}

func TestImportEndpoint(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	t.Run("POST /api/v1/import - Import program with valid source", func(t *testing.T) {
		userID := createTestUser(t, suite.svc, "test-user-1-import@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-1-import@example.com", "password123")

		metadata := map[string]interface{}{"name": "Test YouTube Source"}
		metadataBytes, _ := json.Marshal(metadata)
		source, err := suite.sourceRepo.Create(context.Background(), sqlc.CreateSourceParams{
			Type:     "youtube",
			Metadata: metadataBytes,
		})
		require.NoError(t, err)

		importReq := handler.ImportRequest{
			SourceType: "youtube",
			Metadata: map[string]interface{}{
				"slug":         fmt.Sprintf("test-import-%s", userID[:8]),
				"title":        "Imported Program",
				"description":  "A program imported from YouTube",
				"type":         "podcast",
				"language":     "en",
				"duration_ms":  3600000,
				"source_id":    source.ID,
				"external_id":  "yt-12345",
				"external_url": "https://youtube.com/watch?v=12345",
			},
		}
		body, _ := json.Marshal(importReq)

		w := makeRequest("POST", "/api/v1/import", body, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusCreated, w.Code)

		var program map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &program))
		assert.Equal(t, "Imported Program", program["title"])
		assert.NotEmpty(t, program["id"])
	})

	t.Run("POST /api/v1/import - Invalid source type", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-2-import@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-2-import@example.com", "password123")

		importReq := handler.ImportRequest{
			SourceType: "invalid_source",
			Metadata: map[string]interface{}{
				"slug":        "test-import-invalid",
				"title":       "Invalid Source Program",
				"description": "A program with invalid source type",
				"type":        "podcast",
				"language":    "en",
				"duration_ms": 3600000,
			},
		}
		body, _ := json.Marshal(importReq)

		w := makeRequest("POST", "/api/v1/import", body, loginResp.AccessToken, suite.testRouter)
		assert.NotEqual(t, http.StatusCreated, w.Code)
	})

	t.Run("POST /api/v1/import - Verify imported program in DB", func(t *testing.T) {
		createTestUser(t, suite.svc, "test-user-3-import@example.com", "password123")
		loginResp := loginUser(t, suite.testRouter, "test-user-3-import@example.com", "password123")

		importReq := handler.ImportRequest{
			SourceType: "youtube",
			Metadata: map[string]interface{}{
				"title":        "Imported YouTube Program",
				"description":  "A program imported from YouTube",
				"type":         "podcast",
				"language":     "en",
				"duration_ms":  1800000,
				"external_id":  "yt-67890",
				"external_url": "https://youtube.com/watch?v=67890",
			},
		}
		body, _ := json.Marshal(importReq)

		w := makeRequest("POST", "/api/v1/import", body, loginResp.AccessToken, suite.testRouter)
		assert.Equal(t, http.StatusCreated, w.Code)

		var program map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &program))

		slug := program["slug"].(string)
		dbProgram, err := suite.svc.GetProgramBySlug(context.Background(), slug)
		require.NoError(t, err)
		assert.Equal(t, "Imported YouTube Program", dbProgram.Title)
	})
}
