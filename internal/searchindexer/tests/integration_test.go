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
	redisClient.FlushAll(ctx)

	typesenseClient := typesense.NewClient(
		typesense.WithServer("http://localhost:8108"),
		typesense.WithAPIKey("xyz"),
	)

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

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: "localhost:6379"})

	time.Sleep(2 * time.Second)

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
		CreatedBy:   "test-user",
		PublishedAt: &now,
	}
}

func TestUpsertPublishedProgram(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	suite.redisClient.FlushAll(context.Background())

	program := createPublishedProgram("test-indexer-published", "Test Indexer Published")

	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramUpsert, program)

	time.Sleep(3 * time.Second)

	doc, err := suite.typesenseClient.Collection("programs").Document(program.ID).Retrieve(context.Background())
	require.NoError(t, err)
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

	suite.redisClient.FlushAll(context.Background())

	program := createPublishedProgram("test-indexer-delete", "Test Indexer Delete")

	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramUpsert, program)

	time.Sleep(3 * time.Second)

	doc, err := suite.typesenseClient.Collection("programs").Document(program.ID).Retrieve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, program.ID, doc["id"])

	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramDelete, program)

	time.Sleep(1 * time.Second)

	_, err = suite.typesenseClient.Collection("programs").Document(program.ID).Retrieve(context.Background())
	assert.Error(t, err)
}

func TestDoNotIndexDeletedProgram(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	suite.redisClient.FlushAll(context.Background())

	now := time.Now()
	program := createPublishedProgram("test-indexer-deleted", "Test Indexer Deleted")
	program.DeletedAt = &now

	enqueueTask(t, suite.asynqClient, domain.OutboxEventTypeProgramUpsert, program)

	time.Sleep(1 * time.Second)

	_, err := suite.typesenseClient.Collection("programs").Document(program.ID).Retrieve(context.Background())
	assert.Error(t, err)
}
