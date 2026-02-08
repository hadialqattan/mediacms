//go:build integration

package searchindexertest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"

	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type testSuite struct {
	redisClient     *redis.Client
	typesenseClient *typesense.Client
	asynqClient     *asynq.Client
	cleanupFunc     func()
}

func setupTestSuite(t *testing.T) *testSuite {
	ctx := context.Background()

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
		return nil
	}

	typesenseClient := typesense.NewClient(
		typesense.WithServer("http://localhost:8108"),
		typesense.WithAPIKey("xyz"),
	)

	facetTrue := true
	pubField := "published_at"
	schema := &api.CollectionSchema{
		Name: "programs",
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "slug", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "description", Type: "string"},
			{Name: "type", Type: "string", Facet: &facetTrue},
			{Name: "language", Type: "string", Facet: &facetTrue},
			{Name: "tags", Type: "string[]", Facet: &facetTrue},
			{Name: "duration_ms", Type: "int32"},
			{Name: "published_at", Type: "int64"},
			{Name: "created_at", Type: "int64"},
		},
		DefaultSortingField: &pubField,
	}
	_, err := typesenseClient.Collections().Create(ctx, schema)
	if err != nil {
		t.Logf("Warning: Failed to create Typesense collection: %v", err)
	}

	redisClient.FlushAll(ctx)
	time.Sleep(500 * time.Millisecond)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})

	return &testSuite{
		redisClient:     redisClient,
		typesenseClient: typesenseClient,
		asynqClient:     asynqClient,
		cleanupFunc: func() {
			redisClient.Close()
		},
	}
}

func enqueueTask(t *testing.T, client *asynq.Client, eventType domain.OutboxEventType, program domain.Program) {
	payload, err := json.Marshal(program)
	require.NoError(t, err)

	task := asynq.NewTask(string(eventType), payload)
	_, err = client.Enqueue(task)
	require.NoError(t, err)
}

func createPublishedProgram(slug, title string) domain.Program {
	now := time.Now()
	return domain.Program{
		ID:          uuid.New().String(),
		Slug:        slug,
		Title:       title,
		Description: "Test program",
		Type:        domain.ProgramTypePodcast,
		Language:    domain.LanguageEn,
		DurationMs:  3600000,
		Tags:        []string{"test", "integration"},
		CreatedAt:   now,
		CreatedBy:   "test-user",
		PublishedAt: &now,
	}
}

func waitForDocument(t *testing.T, client *typesense.Client, programID string, timeout time.Duration) map[string]interface{} {
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		doc, err := client.Collection("programs").Document(programID).Retrieve(ctx)
		if err == nil {
			return doc
		}
		<-ticker.C
	}
	t.Fatalf("Timeout waiting for document %s to appear in Typesense", programID)
	return nil
}

func waitForDocumentDeletion(t *testing.T, client *typesense.Client, programID string, timeout time.Duration) {
	ctx := context.Background()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		_, err := client.Collection("programs").Document(programID).Retrieve(ctx)
		if err != nil {
			return // Document is gone
		}
		<-ticker.C
	}
	t.Fatalf("Timeout waiting for document %s to be deleted from Typesense", programID)
}

func TestUpsertPublishedProgram(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	program := createPublishedProgram("test-indexer-published", "Test Indexer Published")
	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramUpsert, program)

	doc := waitForDocument(t, suite.typesenseClient, program.ID, 3*time.Second)
	assert.Equal(t, program.ID, doc["id"])
	assert.Equal(t, program.Title, doc["title"])

	t.Cleanup(func() {
		_, _ = suite.typesenseClient.Collection("programs").Document(program.ID).Delete(context.Background())
	})
}

func TestDeleteProgramFromIndex(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	program := createPublishedProgram("test-indexer-delete", "Test Indexer Delete")
	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramUpsert, program)
	waitForDocument(t, suite.typesenseClient, program.ID, 3*time.Second)

	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramDelete, program)
	waitForDocumentDeletion(t, suite.typesenseClient, program.ID, 2*time.Second)
}

func TestDoNotIndexDeletedProgram(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	now := time.Now()
	program := createPublishedProgram("test-indexer-deleted", "Test Indexer Deleted")
	program.DeletedAt = &now

	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramUpsert, program)

	time.Sleep(500 * time.Millisecond)
	_, err := suite.typesenseClient.Collection("programs").Document(program.ID).Retrieve(context.Background())
	assert.Error(t, err, "Deleted program should not be indexed")
}
