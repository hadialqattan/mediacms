//go:build integration

package discoverytest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"

	"github.com/hadialqattan/mediacms/internal/discovery"
	"github.com/hadialqattan/mediacms/internal/discovery/repository"
	"github.com/hadialqattan/mediacms/internal/discovery/router"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type testSuite struct {
	client      *typesense.Client
	testRouter  *chi.Mux
	cleanupFunc func()
}

func setupTestSuite(t *testing.T) *testSuite {
	ctx := context.Background()

	client := typesense.NewClient(
		typesense.WithServer("http://localhost:8108"),
		typesense.WithAPIKey("xyz"),
	)

	slugField := "slug"
	titleField := "title"
	descField := "description"
	typeField := "type"
	langField := "language"
	createdField := "created_at"
	tagsField := "tags"
	pubField := "published_at"
	durationField := "duration_ms"

	facetTrue := true
	schema := &api.CollectionSchema{
		Name: "programs",
		Fields: []api.Field{
			{Name: slugField, Type: "string"},
			{Name: titleField, Type: "string"},
			{Name: descField, Type: "string"},
			{Name: typeField, Type: "string", Facet: &facetTrue},
			{Name: langField, Type: "string", Facet: &facetTrue},
			{Name: tagsField, Type: "string[]", Facet: &facetTrue},
			{Name: pubField, Type: "int64"},
			{Name: createdField, Type: "int64"},
			{Name: durationField, Type: "int32"},
		},
		DefaultSortingField: &pubField,
	}

	_, _ = client.Collection("programs").Delete(ctx)

	_, err := client.Collections().Create(ctx, schema)
	require.NoError(t, err, "Failed to create Typesense collection")

	searchIndex := repository.NewSearchIndex(client)
	discoveryService := discovery.NewService(searchIndex)
	testRouter := router.NewRouter(discoveryService)

	return &testSuite{
		client:     client,
		testRouter: testRouter,
		cleanupFunc: func() {
			_, _ = client.Collection("programs").Delete(ctx)
		},
	}
}

func createTestProgram(t *testing.T, client *typesense.Client, program domain.Program) {
	ctx := context.Background()
	searchIndex := repository.NewSearchIndex(client)

	err := searchIndex.UpsertProgram(ctx, program)
	require.NoError(t, err, "Failed to index test program")
}

func makeRequest(method, path string, body []byte, testRouter http.Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func assertProgramEqual(t *testing.T, expected, actual map[string]interface{}) {
	assert.Equal(t, expected["id"], actual["id"])
	assert.Equal(t, expected["slug"], actual["slug"])
	assert.Equal(t, expected["title"], actual["title"])
	assert.Equal(t, expected["type"], actual["type"])
	assert.Equal(t, expected["language"], actual["language"])
}

func TestHealthEndpoint(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.cleanupFunc()

	t.Run("GET /health - Health check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		suite.testRouter.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, "ok", response["status"])
		assert.Equal(t, "discovery-api", response["service"])
		assert.NotNil(t, response["timestamp"])
	})
}

func TestSearchEndpoints(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.cleanupFunc()

	now := time.Now()
	baseTime := now.Add(-24 * time.Hour)

	testPrograms := []domain.Program{
		{
			ID:          "tech-podcast-1",
			Slug:        "tech-podcast-1",
			Title:       "The Future of AI",
			Description: "Exploring artificial intelligence trends",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageEn,
			DurationMs:  3600000,
			Tags:        []string{"tech", "ai", "future"},
			CreatedAt:   baseTime,
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "news-podcast-1",
			Slug:        "news-podcast-1",
			Title:       "Daily News Briefing",
			Description: "Your daily news update",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageEn,
			DurationMs:  1800000,
			Tags:        []string{"news", "politics"},
			CreatedAt:   baseTime.Add(time.Hour),
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "sports-doc-1",
			Slug:        "sports-doc-1",
			Title:       "World Cup Documentary",
			Description: "Behind the scenes of the World Cup",
			Type:        domain.ProgramTypeDocumentary,
			Language:    domain.LanguageAr,
			DurationMs:  7200000,
			Tags:        []string{"sports", "football"},
			CreatedAt:   baseTime.Add(2 * time.Hour),
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "tech-doc-1",
			Slug:        "tech-doc-1",
			Title:       "The AI Revolution",
			Description: "How AI is changing the world",
			Type:        domain.ProgramTypeDocumentary,
			Language:    domain.LanguageEn,
			DurationMs:  5400000,
			Tags:        []string{"tech", "ai", "documentary"},
			CreatedAt:   baseTime.Add(3 * time.Hour),
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "politics-podcast-1",
			Slug:        "politics-podcast-1",
			Title:       "Political Analysis",
			Description: "Deep dive into current events",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageAr,
			DurationMs:  2700000,
			Tags:        []string{"news", "politics", "analysis"},
			CreatedAt:   baseTime.Add(4 * time.Hour),
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
	}

	for _, prog := range testPrograms {
		createTestProgram(t, suite.client, prog)
	}

	// Give Typesense time to index
	time.Sleep(500 * time.Millisecond)

	t.Run("GET /api/v1/programs - Empty search returns all programs", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results, ok := response["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.GreaterOrEqual(t, len(results), 5, "should return at least 5 programs")

		pagination, ok := response["pagination"].(map[string]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, int(pagination["total"].(float64)), 5)
	})

	t.Run("GET /api/v1/programs?q=query - Search by query", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs?q=AI", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 2, "should find programs with 'AI' in title/description")
	})

	t.Run("GET /api/v1/programs?type=podcast - Filter by type", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs?type=podcast", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 3, "should find at least 3 podcasts")
	})

	t.Run("GET /api/v1/programs?language=en - Filter by language", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs?language=en", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 3, "should find at least 3 English programs")
	})

	t.Run("GET /api/v1/programs?tags=tech - Filter by tags", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs?tags=tech", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 2, "should find programs with 'tech' tag")
	})

	t.Run("GET /api/v1/programs?sort=recent - Sort by recent", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs?sort=recent", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 1)
	})

	t.Run("GET /api/v1/programs?page=1&per_page=2 - Pagination", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs?page=1&per_page=2", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.LessOrEqual(t, len(results), 2, "should return at most 2 programs per page")

		pagination := response["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(2), pagination["per_page"])
		assert.Greater(t, pagination["total"].(float64), float64(2))
	})

}

func TestGetProgramEndpoint(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.cleanupFunc()

	baseTime := time.Now().Add(-24 * time.Hour)

	testProgram := domain.Program{
		ID:          "get-test-1",
		Slug:        "get-test-slug",
		Title:       "Test Program for Get",
		Description: "Testing the get endpoint",
		Type:        domain.ProgramTypePodcast,
		Language:    domain.LanguageEn,
		DurationMs:  3600000,
		Tags:        []string{"test", "get"},
		CreatedAt:   baseTime,
		PublishedAt: &baseTime,
		CreatedBy:   "test-user",
	}

	createTestProgram(t, suite.client, testProgram)
	time.Sleep(500 * time.Millisecond)

	t.Run("GET /api/v1/programs/{id} - Valid ID returns program", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/get-test-1", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		assert.Equal(t, "get-test-1", response["id"])
		assert.Equal(t, "get-test-slug", response["slug"])
		assert.Equal(t, "Test Program for Get", response["title"])
		assert.Equal(t, "podcast", response["type"])
		assert.Equal(t, "en", response["language"])
		assert.Equal(t, float64(3600000), response["duration_ms"])
		assert.NotEmpty(t, response["tags"])
		assert.NotNil(t, response["published_at"])
		assert.NotNil(t, response["created_at"])
	})

	t.Run("GET /api/v1/programs/{id} - Invalid ID returns 404", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/nonexistent-id", nil, suite.testRouter)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GET /api/v1/programs/{id} - Verify ETag header", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/get-test-1", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)
		etag := w.Header().Get("ETag")
		assert.NotEmpty(t, etag, "ETag header should be present")
		assert.Contains(t, etag, "get-test-1")
	})

	t.Run("GET /api/v1/programs/{id} - Verify cache headers", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/get-test-1", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)
		cacheControl := w.Header().Get("Cache-Control")
		assert.Contains(t, cacheControl, "public")
		assert.Contains(t, cacheControl, "max-age=300")
	})
}

func TestRecentProgramsEndpoint(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.cleanupFunc()

	now := time.Now()

	// Create programs with different published times
	recentPrograms := []domain.Program{
		{
			ID:          "recent-1",
			Slug:        "recent-podcast-1",
			Title:       "Recent Tech Podcast",
			Description: "Latest tech news",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageEn,
			DurationMs:  3600000,
			Tags:        []string{"tech"},
			CreatedAt:   now.Add(-1 * time.Hour),
			PublishedAt: func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
			CreatedBy:   "test-user",
		},
		{
			ID:          "recent-2",
			Slug:        "recent-doc-1",
			Title:       "Recent Documentary",
			Description: "Latest documentary film",
			Type:        domain.ProgramTypeDocumentary,
			Language:    domain.LanguageAr,
			DurationMs:  7200000,
			Tags:        []string{"documentary"},
			CreatedAt:   now.Add(-2 * time.Hour),
			PublishedAt: func() *time.Time { t := now.Add(-2 * time.Hour); return &t }(),
			CreatedBy:   "test-user",
		},
		{
			ID:          "recent-3",
			Slug:        "recent-podcast-2",
			Title:       "Recent Sports Podcast",
			Description: "Latest sports news",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageEn,
			DurationMs:  2700000,
			Tags:        []string{"sports"},
			CreatedAt:   now.Add(-3 * time.Hour),
			PublishedAt: func() *time.Time { t := now.Add(-3 * time.Hour); return &t }(),
			CreatedBy:   "test-user",
		},
	}

	for _, prog := range recentPrograms {
		createTestProgram(t, suite.client, prog)
	}

	time.Sleep(500 * time.Millisecond)

	t.Run("GET /api/v1/programs/recent - Returns recent programs", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/recent", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results, ok := response["results"].([]interface{})
		assert.True(t, ok, "results should be an array")
		assert.GreaterOrEqual(t, len(results), 3)

		pagination, ok := response["pagination"].(map[string]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, int(pagination["total"].(float64)), 3)
	})

	t.Run("GET /api/v1/programs/recent?type=podcast - Filter by type", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/recent?type=podcast", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 2, "should find at least 2 recent podcasts")
	})

	t.Run("GET /api/v1/programs/recent?language=en - Filter by language", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/recent?language=en", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.GreaterOrEqual(t, len(results), 2, "should find at least 2 recent English programs")
	})

	t.Run("GET /api/v1/programs/recent?page=1&per_page=2 - Pagination", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/recent?page=1&per_page=2", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		results := response["results"].([]interface{})
		assert.LessOrEqual(t, len(results), 2)

		pagination := response["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(2), pagination["per_page"])
	})

}

func TestFacetsEndpoint(t *testing.T) {
	suite := setupTestSuite(t)
	defer suite.cleanupFunc()

	baseTime := time.Now().Add(-24 * time.Hour)

	facetPrograms := []domain.Program{
		{
			ID:          "facet-1",
			Slug:        "facet-podcast-1",
			Title:       "Tech Podcast",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageEn,
			DurationMs:  3600000,
			Tags:        []string{"tech", "ai", "programming"},
			CreatedAt:   baseTime,
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "facet-2",
			Slug:        "facet-doc-1",
			Title:       "Sports Documentary",
			Type:        domain.ProgramTypeDocumentary,
			Language:    domain.LanguageAr,
			DurationMs:  7200000,
			Tags:        []string{"sports", "football"},
			CreatedAt:   baseTime,
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "facet-3",
			Slug:        "facet-podcast-2",
			Title:       "News Podcast",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageEn,
			DurationMs:  1800000,
			Tags:        []string{"news", "politics"},
			CreatedAt:   baseTime,
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "facet-4",
			Slug:        "facet-doc-2",
			Title:       "Tech Documentary",
			Type:        domain.ProgramTypeDocumentary,
			Language:    domain.LanguageEn,
			DurationMs:  5400000,
			Tags:        []string{"tech", "science"},
			CreatedAt:   baseTime,
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
		{
			ID:          "facet-5",
			Slug:        "facet-podcast-3",
			Title:       "Arabic Podcast",
			Type:        domain.ProgramTypePodcast,
			Language:    domain.LanguageAr,
			DurationMs:  2700000,
			Tags:        []string{"culture", "society"},
			CreatedAt:   baseTime,
			PublishedAt: &baseTime,
			CreatedBy:   "test-user",
		},
	}

	for _, prog := range facetPrograms {
		createTestProgram(t, suite.client, prog)
	}

	time.Sleep(500 * time.Millisecond)

	t.Run("GET /api/v1/programs/facets - Returns all facets", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/facets", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		facets, ok := response["facets"].(map[string]interface{})
		assert.True(t, ok, "facets should be an object")

		// Check type facet
		typeFacet, ok := facets["type"].(map[string]interface{})
		assert.True(t, ok, "type facet should exist")
		assert.NotEmpty(t, typeFacet, "type facet should have values")

		// Check language facet
		languageFacet, ok := facets["language"].(map[string]interface{})
		assert.True(t, ok, "language facet should exist")
		assert.NotEmpty(t, languageFacet, "language facet should have values")

		// Check tags facet
		tagsFacet, ok := facets["tags"].(map[string]interface{})
		assert.True(t, ok, "tags facet should exist")
		assert.NotEmpty(t, tagsFacet, "tags facet should have values")
	})

	t.Run("GET /api/v1/programs/facets - Verify facet structure", func(t *testing.T) {
		w := makeRequest("GET", "/api/v1/programs/facets", nil, suite.testRouter)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

		facets := response["facets"].(map[string]interface{})

		typeFacet := facets["type"].(map[string]interface{})
		assert.Contains(t, typeFacet, "podcast")
		assert.Contains(t, typeFacet, "documentary")

		languageFacet := facets["language"].(map[string]interface{})
		assert.Contains(t, languageFacet, "en")
		assert.Contains(t, languageFacet, "ar")

		tagsFacet := facets["tags"].(map[string]interface{})
		assert.Contains(t, tagsFacet, "tech")
	})

}
