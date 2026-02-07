//go:build integration

package discoverytest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"

	"thmanyah.com/content-platform/internal/discovery"
	"thmanyah.com/content-platform/internal/discovery/repository"
	"thmanyah.com/content-platform/internal/discovery/router"
	"thmanyah.com/content-platform/internal/shared/domain"
)

func TestDiscoveryEndpoints(t *testing.T) {
	client := typesense.NewClient(
		typesense.WithServer("http://localhost:8108"),
		typesense.WithAPIKey("xyz"),
	)

	ctx := context.Background()

	idField := "id"
	slugField := "slug"
	titleField := "title"
	descField := "description"
	typeField := "type"
	langField := "language"
	durField := "duration_ms"
	catField := "categories"
	pubField := "published_at"

	schema := &api.CollectionSchema{
		Name: "programs",
		Fields: []api.Field{
			{Name: idField, Type: "string"},
			{Name: slugField, Type: "string"},
			{Name: titleField, Type: "string"},
			{Name: descField, Type: "string"},
			{Name: typeField, Type: "string"},
			{Name: langField, Type: "string"},
			{Name: durField, Type: "int32"},
			{Name: catField, Type: "string[]"},
			{Name: pubField, Type: "int64"},
		},
	}

	collection, err := client.Collections().Create(ctx, schema)
	if err != nil {
		t.Fatalf("Failed to create collection: %v", err)
	}
	t.Logf("Created collection: %v", collection.Name)

	t.Cleanup(func() {
		_, _ = client.Collection("programs").Delete(ctx)
	})

	searchIndex := repository.NewSearchIndex(client)
	discoveryService := discovery.NewService(searchIndex)
	testRouter := router.NewRouter(discoveryService)

	testProgram := domain.Program{
		ID:          "test-id-123",
		Slug:        "test-podcast-discovery",
		Title:       "Test Discovery Podcast",
		Description: "A test podcast for discovery",
		Type:        domain.ProgramTypePodcast,
		Language:    domain.LanguageEn,
		DurationMs:  3600000,
		Categories:  []domain.Category{},
	}

	publishedAt := time.Unix(1640995200, 0)
	testProgram.PublishedAt = &publishedAt

	if err := searchIndex.UpsertProgram(ctx, testProgram); err != nil {
		t.Fatalf("Failed to index test program: %v", err)
	}

	t.Run("Search programs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/programs/search?q=podcast", nil)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		programs := result["programs"].([]interface{})
		if len(programs) == 0 {
			t.Fatal("Expected at least one program in search results")
		}

		t.Logf("Found %d programs", len(programs))
	})

	t.Run("Get program by ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/programs/test-id-123", nil)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var program map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &program); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if program["id"] != "test-id-123" {
			t.Fatalf("Expected program id test-id-123, got %v", program["id"])
		}

		t.Logf("Got program: %v", program["title"])
	})
}
