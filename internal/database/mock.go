package database

import (
        "database/sql"
        "fmt"
        "time"

        "github.com/DATA-DOG/go-sqlmock"
)

// NewMockConnection creates a mock database connection for testing
func NewMockConnection() (*sql.DB, sqlmock.Sqlmock, error) {
        db, mock, err := sqlmock.New()
        if err != nil {
                return nil, nil, fmt.Errorf("failed to create mock database: %w", err)
        }

        // Set up basic mock expectations for migrations
        mock.ExpectExec("CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE TABLE IF NOT EXISTS users").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE TABLE IF NOT EXISTS projects").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE TABLE IF NOT EXISTS tasks").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE TABLE IF NOT EXISTS task_metrics").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectExec("CREATE INDEX IF NOT EXISTS").WillReturnResult(sqlmock.NewResult(0, 0))
        mock.ExpectQuery("SELECT create_hypertable").WillReturnRows(sqlmock.NewRows([]string{"create_hypertable"}).AddRow("1"))

        return db, mock, nil
}

// SetupMockLogin sets up mock expectations for login
func SetupMockLogin(mock sqlmock.Sqlmock) {
        // Mock user login query
        mock.ExpectQuery("SELECT id, username, email, full_name, role, password_hash FROM users WHERE username").
                WithArgs("testuser").
                WillReturnRows(sqlmock.NewRows([]string{"id", "username", "email", "full_name", "role", "password_hash"}).
                        AddRow(1, "testuser", "test@example.com", "Test User", "admin", "$2a$10$N9qo8uLOickgx2ZMRZoMye6yIrM8/p0cQpZ9kDOBjE1zGOGvt.uAW")) // password123
}

// SetupMockTasks sets up mock expectations for task operations
func SetupMockTasks(mock sqlmock.Sqlmock) {
        // Mock task creation
        mock.ExpectQuery("INSERT INTO tasks").
                WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
                WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "status", "priority", "assignee_id", "project_id", "created_at", "updated_at", "completed_at", "due_date", "estimated_hours", "actual_hours", "tags"}).
                        AddRow(1, "Test Task", "A test task", "todo", 1, nil, 1, time.Now(), time.Now(), nil, nil, nil, nil, "{}"))

        // Mock task retrieval
        mock.ExpectQuery("SELECT id, title, description, status, priority, assignee_id, project_id, created_at, updated_at, completed_at, due_date, estimated_hours, actual_hours, tags FROM tasks").
                WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "status", "priority", "assignee_id", "project_id", "created_at", "updated_at", "completed_at", "due_date", "estimated_hours", "actual_hours", "tags"}).
                        AddRow(1, "Test Task", "A test task", "todo", 1, nil, 1, time.Now(), time.Now(), nil, nil, nil, nil, "{}"))

        // Mock metrics query
        mock.ExpectQuery("SELECT status, project_id, COUNT").
                WillReturnRows(sqlmock.NewRows([]string{"status", "project_id", "count"}).
                        AddRow("todo", 1, 5).
                        AddRow("in_progress", 1, 3).
                        AddRow("done", 1, 10))

        // Mock active tasks count
        mock.ExpectQuery("SELECT COUNT").
                WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))
}