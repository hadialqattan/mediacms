//go:build integration

package outboxrelaytest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/outboxrelay"
	"github.com/hadialqattan/mediacms/internal/outboxrelay/repository"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type testSuite struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
	cmsQueries  *sqlc.Queries
	relay       *outboxrelay.Relay
	cleanupFunc func()
}

func setupTestSuite(t *testing.T) *testSuite {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/mediacms?sslmode=disable")
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return nil
	}

	redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		pool.Close()
		t.Skipf("Redis not available: %v", err)
		return nil
	}

	cleanupTestData(ctx, t, pool)
	redisClient.FlushAll(ctx)

	cmsQueries := sqlc.New(pool)
	outboxRepo := repository.NewOutboxRepo(pool)
	asynqClient := repository.NewQueue("localhost:6379")
	relay := outboxrelay.NewRelay(outboxRepo, asynqClient, 100*time.Millisecond)

	return &testSuite{
		pool:        pool,
		redisClient: redisClient,
		cmsQueries:  cmsQueries,
		relay:       relay,
		cleanupFunc: func() {
			cleanupTestData(ctx, t, pool)
			pool.Close()
			redisClient.Close()
		},
	}
}

func createTestUser(t *testing.T, queries *sqlc.Queries, email string) sqlc.User {
	user, err := queries.CreateUser(context.Background(), sqlc.CreateUserParams{
		Email:        email,
		PasswordHash: "hash",
		Role:         string(domain.UserRoleEditor),
	})
	require.NoError(t, err)
	return user
}

func createTestProgram(t *testing.T, queries *sqlc.Queries, slug string, userID pgtype.UUID) sqlc.Program {
	program, err := queries.CreateProgram(context.Background(), sqlc.CreateProgramParams{
		Slug:        slug,
		Title:       "Test Program",
		Description: pgtypeText("Test program for relay"),
		Type:        "podcast",
		Language:    "en",
		DurationMs:  3600000,
		Tags:        []string{"test"},
		CreatedBy:   userID,
	})
	require.NoError(t, err)
	return program
}

func createTestOutboxEvent(t *testing.T, queries *sqlc.Queries, programID pgtype.UUID, slug string) sqlc.OutboxEvent {
	payload := map[string]interface{}{
		"program_id": uuid.UUID(programID.Bytes).String(),
		"slug":       slug,
		"title":      "Test Program",
	}
	payloadBytes, _ := json.Marshal(payload)

	event, err := queries.CreateOutboxEvent(context.Background(), sqlc.CreateOutboxEventParams{
		Type:      string(domain.OutboxEventTypeProgramUpsert),
		Payload:   payloadBytes,
		ProgramID: programID,
	})
	require.NoError(t, err)
	return event
}

func cleanupTestData(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	// TODO: Use testing containers and remove this function.
	pool.Exec(ctx, "DELETE FROM outbox_events WHERE program_id IN (SELECT id FROM programs WHERE slug LIKE 'test-relay-%')")
	pool.Exec(ctx, "DELETE FROM categorized_as WHERE program_id IN (SELECT id FROM programs WHERE slug LIKE 'test-relay-%')")
	pool.Exec(ctx, "DELETE FROM programs WHERE slug LIKE 'test-relay-%'")
	pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'test-relay-%@example.com'")
}

func pgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func pgtypeText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func TestBasicRelayFlow(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	suite.redisClient.FlushAll(context.Background())

	user := createTestUser(t, suite.cmsQueries, "test-relay-user@example.com")
	program := createTestProgram(t, suite.cmsQueries, "test-relay-basic", user.ID)
	event := createTestOutboxEvent(t, suite.cmsQueries, program.ID, "test-relay-basic")
	eventID := event.ID.String()

	relayCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go suite.relay.Start(relayCtx)

	time.Sleep(500 * time.Millisecond)

	var enqueued bool
	err := suite.pool.QueryRow(context.Background(), "SELECT enqueued FROM outbox_events WHERE id = $1", eventID).Scan(&enqueued)
	require.NoError(t, err)
	assert.True(t, enqueued)
}

func TestMultipleEvents(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	suite.redisClient.FlushAll(context.Background())

	user := createTestUser(t, suite.cmsQueries, "test-relay-multi-user@example.com")

	eventCount := 5
	eventIDs := make([]pgtype.UUID, eventCount)

	for i := 0; i < eventCount; i++ {
		slug := "test-relay-multi-" + string(rune('a'+i))
		program := createTestProgram(t, suite.cmsQueries, slug, user.ID)
		event := createTestOutboxEvent(t, suite.cmsQueries, program.ID, slug)
		eventIDs[i] = event.ID
	}

	relayCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go suite.relay.Start(relayCtx)

	time.Sleep(500 * time.Millisecond)

	var enqueuedCount int
	err := suite.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM outbox_events WHERE id = ANY($1) AND enqueued = true", eventIDs).Scan(&enqueuedCount)
	require.NoError(t, err)
	assert.Equal(t, eventCount, enqueuedCount)
}

func TestIdempotency(t *testing.T) {
	suite := setupTestSuite(t)
	if suite == nil {
		return
	}
	defer suite.cleanupFunc()

	suite.redisClient.FlushAll(context.Background())

	user := createTestUser(t, suite.cmsQueries, "test-relay-idempotent-user@example.com")
	program := createTestProgram(t, suite.cmsQueries, "test-relay-idempotent", user.ID)
	event := createTestOutboxEvent(t, suite.cmsQueries, program.ID, "test-relay-idempotent")
	eventID := event.ID.String()

	_, err := suite.pool.Exec(context.Background(), "UPDATE outbox_events SET enqueued = true WHERE id = $1", eventID)
	require.NoError(t, err)

	relayCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go suite.relay.Start(relayCtx)

	time.Sleep(300 * time.Millisecond)

	var enqueued bool
	err = suite.pool.QueryRow(context.Background(), "SELECT enqueued FROM outbox_events WHERE id = $1", eventID).Scan(&enqueued)
	require.NoError(t, err)
	assert.True(t, enqueued)
}
