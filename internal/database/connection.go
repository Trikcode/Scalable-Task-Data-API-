package database

import (
        "database/sql"
        "fmt"
        "scalable-task-api/internal/config"

        _ "github.com/lib/pq"
)

// NewConnection creates a new database connection
func NewConnection(cfg config.DatabaseConfig) (*sql.DB, error) {
        dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
                cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

        db, err := sql.Open("postgres", dsn)
        if err != nil {
                return nil, fmt.Errorf("failed to open database connection: %w", err)
        }

        // Configure connection pool
        db.SetMaxOpenConns(cfg.MaxOpenConns)
        db.SetMaxIdleConns(cfg.MaxIdleConns)
        db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

        // Test connection
        if err := db.Ping(); err != nil {
                return nil, fmt.Errorf("failed to ping database: %w", err)
        }

        return db, nil
}

// RunMigrations runs database migrations
func RunMigrations(db *sql.DB) error {
        migrations := []string{
                createExtensionsSQL,
                createUsersTableSQL,
                createProjectsTableSQL,
                createTasksTableSQL,
                createTaskMetricsTableSQL,
                createIndexesSQL,
                createHypertableSQL,
        }

        for i, migration := range migrations {
                if _, err := db.Exec(migration); err != nil {
                        return fmt.Errorf("failed to run migration %d: %w", i+1, err)
                }
        }

        return nil
}

const createExtensionsSQL = `
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
`

const createUsersTableSQL = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const createProjectsTableSQL = `
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id INTEGER REFERENCES users(id),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const createTasksTableSQL = `
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'todo',
    priority INTEGER DEFAULT 1,
    assignee_id INTEGER REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    due_date TIMESTAMPTZ,
    estimated_hours DECIMAL(10,2),
    actual_hours DECIMAL(10,2),
    tags TEXT[]
);
`

const createTaskMetricsTableSQL = `
CREATE TABLE IF NOT EXISTS task_metrics (
    timestamp TIMESTAMPTZ NOT NULL,
    total_tasks INTEGER NOT NULL DEFAULT 0,
    completed_tasks INTEGER NOT NULL DEFAULT 0,
    in_progress_tasks INTEGER NOT NULL DEFAULT 0,
    overdue_tasks INTEGER NOT NULL DEFAULT 0,
    avg_completion_time DECIMAL(10,2),
    project_id INTEGER REFERENCES projects(id)
);
`

const createIndexesSQL = `
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee ON tasks(assignee_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);
CREATE INDEX IF NOT EXISTS idx_tasks_tags ON tasks USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
`

const createHypertableSQL = `
SELECT create_hypertable('task_metrics', 'timestamp', if_not_exists => TRUE);
`