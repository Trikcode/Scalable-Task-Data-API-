package models

import (
	"time"
)

// Task represents a time-series task data entry
type Task struct {
	ID          int       `json:"id" db:"id"`
	Title       string    `json:"title" db:"title" binding:"required"`
	Description string    `json:"description" db:"description"`
	Status      string    `json:"status" db:"status" binding:"required"`
	Priority    int       `json:"priority" db:"priority"`
	AssigneeID  *int      `json:"assignee_id" db:"assignee_id"`
	ProjectID   int       `json:"project_id" db:"project_id" binding:"required"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	DueDate     *time.Time `json:"due_date" db:"due_date"`
	EstimatedHours *float64 `json:"estimated_hours" db:"estimated_hours"`
	ActualHours    *float64 `json:"actual_hours" db:"actual_hours"`
	Tags        []string  `json:"tags"`
}

// TaskStatus represents valid task statuses
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username" binding:"required"`
	Email     string    `json:"email" db:"email" binding:"required,email"`
	FullName  string    `json:"full_name" db:"full_name"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Project represents a project in the system
type Project struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" binding:"required"`
	Description string    `json:"description" db:"description"`
	OwnerID     int       `json:"owner_id" db:"owner_id" binding:"required"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// TaskMetrics represents aggregated task metrics for time-series analysis
type TaskMetrics struct {
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	TotalTasks      int       `json:"total_tasks" db:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks" db:"completed_tasks"`
	InProgressTasks int       `json:"in_progress_tasks" db:"in_progress_tasks"`
	OverdueTasks    int       `json:"overdue_tasks" db:"overdue_tasks"`
	AvgCompletionTime *float64 `json:"avg_completion_time" db:"avg_completion_time"`
	ProjectID       *int      `json:"project_id" db:"project_id"`
}

// CreateTaskRequest represents the request payload for creating a task
type CreateTaskRequest struct {
	Title          string     `json:"title" binding:"required"`
	Description    string     `json:"description"`
	Status         string     `json:"status" binding:"required"`
	Priority       int        `json:"priority"`
	AssigneeID     *int       `json:"assignee_id"`
	ProjectID      int        `json:"project_id" binding:"required"`
	DueDate        *time.Time `json:"due_date"`
	EstimatedHours *float64   `json:"estimated_hours"`
	Tags           []string   `json:"tags"`
}

// UpdateTaskRequest represents the request payload for updating a task
type UpdateTaskRequest struct {
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	Status         *string    `json:"status"`
	Priority       *int       `json:"priority"`
	AssigneeID     *int       `json:"assignee_id"`
	DueDate        *time.Time `json:"due_date"`
	EstimatedHours *float64   `json:"estimated_hours"`
	ActualHours    *float64   `json:"actual_hours"`
	Tags           []string   `json:"tags"`
}

// TaskQuery represents query parameters for filtering tasks
type TaskQuery struct {
	Status     []string  `form:"status"`
	AssigneeID *int      `form:"assignee_id"`
	ProjectID  *int      `form:"project_id"`
	Priority   *int      `form:"priority"`
	FromDate   *time.Time `form:"from_date" time_format:"2006-01-02"`
	ToDate     *time.Time `form:"to_date" time_format:"2006-01-02"`
	Tags       []string  `form:"tags"`
	Limit      int       `form:"limit"`
	Offset     int       `form:"offset"`
	SortBy     string    `form:"sort_by"`
	SortOrder  string    `form:"sort_order"`
}

// MetricsQuery represents query parameters for metrics
type MetricsQuery struct {
	FromDate  time.Time `form:"from_date" binding:"required" time_format:"2006-01-02"`
	ToDate    time.Time `form:"to_date" binding:"required" time_format:"2006-01-02"`
	ProjectID *int      `form:"project_id"`
	Interval  string    `form:"interval"` // hour, day, week, month
}