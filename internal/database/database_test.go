package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Mock database interface for testing
type mockDB struct {
	queryRowFunc func(query string, args ...interface{}) *sql.Row
	queryFunc    func(query string, args ...interface{}) (*sql.Rows, error)
	execFunc     func(query string, args ...interface{}) (sql.Result, error)
	prepareFunc  func(query string) (*sql.Stmt, error)
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(query, args...)
	}
	return nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(query, args...)
	}
	return nil, sql.ErrNoRows
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(query, args...)
	}
	return nil, nil
}

func (m *mockDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if m.prepareFunc != nil {
		return m.prepareFunc(query)
	}
	return nil, nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		db   DBTX
	}{
		{
			name: "create queries with mock db",
			db:   &mockDB{},
		},
		{
			name: "create queries with nil db",
			db:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queries := New(tt.db)

			if queries == nil {
				t.Errorf("New() returned nil")
			}

			if queries.db != tt.db {
				t.Errorf("New() db field = %v, want %v", queries.db, tt.db)
			}
		})
	}
}

func TestQueries_WithTx(t *testing.T) {
	mockDb := &mockDB{}
	queries := New(mockDb)

	// Create a mock transaction (nil is fine for testing structure)
	var mockTx *sql.Tx

	newQueries := queries.WithTx(mockTx)

	if newQueries == nil {
		t.Errorf("WithTx() returned nil")
	}

	if newQueries.db != mockTx {
		t.Errorf("WithTx() db field = %v, want %v", newQueries.db, mockTx)
	}

	// Original queries should remain unchanged
	if queries.db != mockDb {
		t.Errorf("Original queries.db changed unexpectedly")
	}
}

func TestCreateUserParams(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		hashedPassword string
	}{
		{
			name:           "valid user params",
			email:          "test@example.com",
			hashedPassword: "hashed_password_123",
		},
		{
			name:           "empty email",
			email:          "",
			hashedPassword: "hashed_password_123",
		},
		{
			name:           "empty password",
			email:          "test@example.com",
			hashedPassword: "",
		},
		{
			name:           "both empty",
			email:          "",
			hashedPassword: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := CreateUserParams{
				Email:          tt.email,
				HashedPassword: tt.hashedPassword,
			}

			if params.Email != tt.email {
				t.Errorf("CreateUserParams.Email = %q, want %q", params.Email, tt.email)
			}

			if params.HashedPassword != tt.hashedPassword {
				t.Errorf("CreateUserParams.HashedPassword = %q, want %q", params.HashedPassword, tt.hashedPassword)
			}
		})
	}
}

func TestCreateMessageParams(t *testing.T) {
	testUUID := uuid.New()

	tests := []struct {
		name   string
		body   string
		userID uuid.UUID
	}{
		{
			name:   "valid message params",
			body:   "Hello, world!",
			userID: testUUID,
		},
		{
			name:   "empty body",
			body:   "",
			userID: testUUID,
		},
		{
			name:   "nil UUID",
			body:   "Hello, world!",
			userID: uuid.Nil,
		},
		{
			name:   "long message body",
			body:   "This is a very long message that exceeds normal length expectations for testing purposes",
			userID: testUUID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := CreateMessageParams{
				Body:   tt.body,
				UserID: tt.userID,
			}

			if params.Body != tt.body {
				t.Errorf("CreateMessageParams.Body = %q, want %q", params.Body, tt.body)
			}

			if params.UserID != tt.userID {
				t.Errorf("CreateMessageParams.UserID = %v, want %v", params.UserID, tt.userID)
			}
		})
	}
}

func TestUser(t *testing.T) {
	testUUID := uuid.New()
	testTime := time.Now()

	tests := []struct {
		name           string
		id             uuid.UUID
		createdAt      time.Time
		updatedAt      time.Time
		email          string
		hashedPassword string
	}{
		{
			name:           "valid user",
			id:             testUUID,
			createdAt:      testTime,
			updatedAt:      testTime,
			email:          "test@example.com",
			hashedPassword: "hashed_password_123",
		},
		{
			name:           "user with zero values",
			id:             uuid.Nil,
			createdAt:      time.Time{},
			updatedAt:      time.Time{},
			email:          "",
			hashedPassword: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := User{
				ID:             tt.id,
				CreatedAt:      tt.createdAt,
				UpdatedAt:      tt.updatedAt,
				Email:          tt.email,
				HashedPassword: tt.hashedPassword,
			}

			if user.ID != tt.id {
				t.Errorf("User.ID = %v, want %v", user.ID, tt.id)
			}

			if user.Email != tt.email {
				t.Errorf("User.Email = %q, want %q", user.Email, tt.email)
			}

			if user.HashedPassword != tt.hashedPassword {
				t.Errorf("User.HashedPassword = %q, want %q", user.HashedPassword, tt.hashedPassword)
			}

			if !user.CreatedAt.Equal(tt.createdAt) {
				t.Errorf("User.CreatedAt = %v, want %v", user.CreatedAt, tt.createdAt)
			}

			if !user.UpdatedAt.Equal(tt.updatedAt) {
				t.Errorf("User.UpdatedAt = %v, want %v", user.UpdatedAt, tt.updatedAt)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	testUUID := uuid.New()
	userUUID := uuid.New()
	testTime := time.Now()

	tests := []struct {
		name      string
		id        uuid.UUID
		createdAt time.Time
		updatedAt time.Time
		body      string
		userID    uuid.UUID
	}{
		{
			name:      "valid message",
			id:        testUUID,
			createdAt: testTime,
			updatedAt: testTime,
			body:      "Hello, world!",
			userID:    userUUID,
		},
		{
			name:      "message with zero values",
			id:        uuid.Nil,
			createdAt: time.Time{},
			updatedAt: time.Time{},
			body:      "",
			userID:    uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := Message{
				ID:        tt.id,
				CreatedAt: tt.createdAt,
				UpdatedAt: tt.updatedAt,
				Body:      tt.body,
				UserID:    tt.userID,
			}

			if message.ID != tt.id {
				t.Errorf("Message.ID = %v, want %v", message.ID, tt.id)
			}

			if message.Body != tt.body {
				t.Errorf("Message.Body = %q, want %q", message.Body, tt.body)
			}

			if message.UserID != tt.userID {
				t.Errorf("Message.UserID = %v, want %v", message.UserID, tt.userID)
			}

			if !message.CreatedAt.Equal(tt.createdAt) {
				t.Errorf("Message.CreatedAt = %v, want %v", message.CreatedAt, tt.createdAt)
			}

			if !message.UpdatedAt.Equal(tt.updatedAt) {
				t.Errorf("Message.UpdatedAt = %v, want %v", message.UpdatedAt, tt.updatedAt)
			}
		})
	}
}

// Integration test for database interface compliance
func TestDBTXInterface(t *testing.T) {
	// Test that mockDB implements DBTX interface
	var _ DBTX = &mockDB{}

	// Test that *sql.DB would implement DBTX interface (compile-time check)
	var _ DBTX = (*sql.DB)(nil)

	// Test that *sql.Tx would implement DBTX interface (compile-time check)
	var _ DBTX = (*sql.Tx)(nil)
}
