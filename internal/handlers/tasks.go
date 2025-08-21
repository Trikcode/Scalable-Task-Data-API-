package handlers

import (
        "database/sql"
        "net/http"
        "scalable-task-api/internal/models"
        "scalable-task-api/internal/monitoring"
        "strconv"
        "strings"

        "github.com/gin-gonic/gin"
        "github.com/lib/pq"
)

// TaskHandler handles task-related endpoints
type TaskHandler struct {
        db      *sql.DB
        metrics *monitoring.Metrics
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(db *sql.DB, metrics *monitoring.Metrics) *TaskHandler {
        return &TaskHandler{
                db:      db,
                metrics: metrics,
        }
}

// CreateTask creates a new task
// @Summary Create a new task
// @Description Create a new task with the provided information
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateTaskRequest true "Task information"
// @Success 201 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
        var req models.CreateTaskRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        // Insert task
        var task models.Task
        err := h.db.QueryRow(`
                INSERT INTO tasks (title, description, status, priority, assignee_id, project_id, due_date, estimated_hours, tags)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
                RETURNING id, title, description, status, priority, assignee_id, project_id, created_at, updated_at, completed_at, due_date, estimated_hours, actual_hours, tags
        `, req.Title, req.Description, req.Status, req.Priority, req.AssigneeID, req.ProjectID, req.DueDate, req.EstimatedHours, pq.Array(req.Tags)).Scan(
                &task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
                &task.AssigneeID, &task.ProjectID, &task.CreatedAt, &task.UpdatedAt,
                &task.CompletedAt, &task.DueDate, &task.EstimatedHours, &task.ActualHours, pq.Array(&task.Tags),
        )

        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
                return
        }

        // Update metrics
        h.updateTaskMetrics()

        c.JSON(http.StatusCreated, task)
}

// GetTasks retrieves tasks with filtering and pagination
// @Summary Get tasks
// @Description Get tasks with optional filtering and pagination
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param status query []string false "Filter by status"
// @Param assignee_id query int false "Filter by assignee ID"
// @Param project_id query int false "Filter by project ID"
// @Param priority query int false "Filter by priority"
// @Param from_date query string false "Filter from date (YYYY-MM-DD)"
// @Param to_date query string false "Filter to date (YYYY-MM-DD)"
// @Param tags query []string false "Filter by tags"
// @Param limit query int false "Limit results" default(50)
// @Param offset query int false "Offset results" default(0)
// @Param sort_by query string false "Sort by field" default(created_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {array} models.Task
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /tasks [get]
func (h *TaskHandler) GetTasks(c *gin.Context) {
        var query models.TaskQuery
        if err := c.ShouldBindQuery(&query); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        // Set defaults
        if query.Limit == 0 {
                query.Limit = 50
        }
        if query.SortBy == "" {
                query.SortBy = "created_at"
        }
        if query.SortOrder == "" {
                query.SortOrder = "desc"
        }

        // Build query
        whereClause, args := h.buildTaskWhereClause(query)
        orderClause := h.buildTaskOrderClause(query.SortBy, query.SortOrder)

        queryStr := `
                SELECT id, title, description, status, priority, assignee_id, project_id, 
                       created_at, updated_at, completed_at, due_date, estimated_hours, actual_hours, tags
                FROM tasks
        ` + whereClause + orderClause + ` LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

        args = append(args, query.Limit, query.Offset)

        rows, err := h.db.Query(queryStr, args...)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query tasks"})
                return
        }
        defer rows.Close()

        var tasks []models.Task
        for rows.Next() {
                var task models.Task
                err := rows.Scan(
                        &task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
                        &task.AssigneeID, &task.ProjectID, &task.CreatedAt, &task.UpdatedAt,
                        &task.CompletedAt, &task.DueDate, &task.EstimatedHours, &task.ActualHours, pq.Array(&task.Tags),
                )
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan task"})
                        return
                }
                tasks = append(tasks, task)
        }

        c.JSON(http.StatusOK, tasks)
}

// GetTask retrieves a single task by ID
// @Summary Get task by ID
// @Description Get a single task by its ID
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
        idStr := c.Param("id")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
                return
        }

        var task models.Task
        err = h.db.QueryRow(`
                SELECT id, title, description, status, priority, assignee_id, project_id, 
                       created_at, updated_at, completed_at, due_date, estimated_hours, actual_hours, tags
                FROM tasks WHERE id = $1
        `, id).Scan(
                &task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
                &task.AssigneeID, &task.ProjectID, &task.CreatedAt, &task.UpdatedAt,
                &task.CompletedAt, &task.DueDate, &task.EstimatedHours, &task.ActualHours, pq.Array(&task.Tags),
        )

        if err != nil {
                if err == sql.ErrNoRows {
                        c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
                        return
                }
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
                return
        }

        c.JSON(http.StatusOK, task)
}

// UpdateTask updates an existing task
// @Summary Update task
// @Description Update an existing task
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param request body models.UpdateTaskRequest true "Task update information"
// @Success 200 {object} models.Task
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
        idStr := c.Param("id")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
                return
        }

        var req models.UpdateTaskRequest
        if err := c.ShouldBindJSON(&req); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        // Build update query dynamically
        updateClause, args := h.buildTaskUpdateClause(req)
        if len(args) == 0 {
                c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
                return
        }

        // Add completion timestamp if status is changed to done
        if req.Status != nil && *req.Status == "done" {
                updateClause += ", completed_at = NOW()"
        }

        query := `
                UPDATE tasks SET ` + updateClause + `, updated_at = NOW()
                WHERE id = $` + strconv.Itoa(len(args)+1) + `
                RETURNING id, title, description, status, priority, assignee_id, project_id, 
                          created_at, updated_at, completed_at, due_date, estimated_hours, actual_hours, tags
        `

        args = append(args, id)

        var task models.Task
        err = h.db.QueryRow(query, args...).Scan(
                &task.ID, &task.Title, &task.Description, &task.Status, &task.Priority,
                &task.AssigneeID, &task.ProjectID, &task.CreatedAt, &task.UpdatedAt,
                &task.CompletedAt, &task.DueDate, &task.EstimatedHours, &task.ActualHours, pq.Array(&task.Tags),
        )

        if err != nil {
                if err == sql.ErrNoRows {
                        c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
                        return
                }
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
                return
        }

        // Update metrics
        h.updateTaskMetrics()

        c.JSON(http.StatusOK, task)
}

// DeleteTask deletes a task
// @Summary Delete task
// @Description Delete a task by ID
// @Tags tasks
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
        idStr := c.Param("id")
        id, err := strconv.Atoi(idStr)
        if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
                return
        }

        result, err := h.db.Exec("DELETE FROM tasks WHERE id = $1", id)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
                return
        }

        rowsAffected, err := result.RowsAffected()
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get affected rows"})
                return
        }

        if rowsAffected == 0 {
                c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
                return
        }

        // Update metrics
        h.updateTaskMetrics()

        c.Status(http.StatusNoContent)
}

// GetTaskMetrics retrieves task metrics for time-series analysis
// @Summary Get task metrics
// @Description Get aggregated task metrics for time-series analysis
// @Tags metrics
// @Produce json
// @Security BearerAuth
// @Param from_date query string true "From date (YYYY-MM-DD)"
// @Param to_date query string true "To date (YYYY-MM-DD)"
// @Param project_id query int false "Filter by project ID"
// @Param interval query string false "Aggregation interval (hour, day, week, month)" default(day)
// @Success 200 {array} models.TaskMetrics
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /tasks/metrics [get]
func (h *TaskHandler) GetTaskMetrics(c *gin.Context) {
        var query models.MetricsQuery
        if err := c.ShouldBindQuery(&query); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        if query.Interval == "" {
                query.Interval = "day"
        }

        // Build time bucket based on interval
        var timeBucket string
        switch query.Interval {
        case "hour":
                timeBucket = "time_bucket('1 hour', created_at)"
        case "day":
                timeBucket = "time_bucket('1 day', created_at)"
        case "week":
                timeBucket = "time_bucket('1 week', created_at)"
        case "month":
                timeBucket = "time_bucket('1 month', created_at)"
        default:
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval"})
                return
        }

        whereClause := "WHERE created_at >= $1 AND created_at <= $2"
        args := []interface{}{query.FromDate, query.ToDate}

        if query.ProjectID != nil {
                whereClause += " AND project_id = $3"
                args = append(args, *query.ProjectID)
        }

        queryStr := `
                SELECT 
                        ` + timeBucket + ` as timestamp,
                        COUNT(*) as total_tasks,
                        COUNT(*) FILTER (WHERE status = 'done') as completed_tasks,
                        COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress_tasks,
                        COUNT(*) FILTER (WHERE due_date < NOW() AND status != 'done') as overdue_tasks,
                        AVG(EXTRACT(EPOCH FROM (completed_at - created_at))/3600) FILTER (WHERE completed_at IS NOT NULL) as avg_completion_time
                FROM tasks 
                ` + whereClause + `
                GROUP BY ` + timeBucket + `
                ORDER BY timestamp
        `

        rows, err := h.db.Query(queryStr, args...)
        if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query metrics"})
                return
        }
        defer rows.Close()

        var metrics []models.TaskMetrics
        for rows.Next() {
                var metric models.TaskMetrics
                err := rows.Scan(
                        &metric.Timestamp, &metric.TotalTasks, &metric.CompletedTasks,
                        &metric.InProgressTasks, &metric.OverdueTasks, &metric.AvgCompletionTime,
                )
                if err != nil {
                        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan metrics"})
                        return
                }
                if query.ProjectID != nil {
                        metric.ProjectID = query.ProjectID
                }
                metrics = append(metrics, metric)
        }

        c.JSON(http.StatusOK, metrics)
}

// Helper functions

func (h *TaskHandler) buildTaskWhereClause(query models.TaskQuery) (string, []interface{}) {
        var conditions []string
        var args []interface{}
        argIndex := 1

        if len(query.Status) > 0 {
                placeholders := make([]string, len(query.Status))
                for i, status := range query.Status {
                        placeholders[i] = "$" + strconv.Itoa(argIndex)
                        args = append(args, status)
                        argIndex++
                }
                conditions = append(conditions, "status IN ("+strings.Join(placeholders, ",")+")")
        }

        if query.AssigneeID != nil {
                conditions = append(conditions, "assignee_id = $"+strconv.Itoa(argIndex))
                args = append(args, *query.AssigneeID)
                argIndex++
        }

        if query.ProjectID != nil {
                conditions = append(conditions, "project_id = $"+strconv.Itoa(argIndex))
                args = append(args, *query.ProjectID)
                argIndex++
        }

        if query.Priority != nil {
                conditions = append(conditions, "priority = $"+strconv.Itoa(argIndex))
                args = append(args, *query.Priority)
                argIndex++
        }

        if query.FromDate != nil {
                conditions = append(conditions, "created_at >= $"+strconv.Itoa(argIndex))
                args = append(args, *query.FromDate)
                argIndex++
        }

        if query.ToDate != nil {
                conditions = append(conditions, "created_at <= $"+strconv.Itoa(argIndex))
                args = append(args, *query.ToDate)
                argIndex++
        }

        if len(query.Tags) > 0 {
                conditions = append(conditions, "tags && $"+strconv.Itoa(argIndex))
                args = append(args, pq.Array(query.Tags))
                argIndex++
        }

        whereClause := ""
        if len(conditions) > 0 {
                whereClause = " WHERE " + strings.Join(conditions, " AND ")
        }

        return whereClause, args
}

func (h *TaskHandler) buildTaskOrderClause(sortBy, sortOrder string) string {
        validColumns := map[string]bool{
                "id":          true,
                "title":       true,
                "status":      true,
                "priority":    true,
                "created_at":  true,
                "updated_at":  true,
                "due_date":    true,
        }

        if !validColumns[sortBy] {
                sortBy = "created_at"
        }

        if sortOrder != "asc" && sortOrder != "desc" {
                sortOrder = "desc"
        }

        return " ORDER BY " + sortBy + " " + sortOrder
}

func (h *TaskHandler) buildTaskUpdateClause(req models.UpdateTaskRequest) (string, []interface{}) {
        var setParts []string
        var args []interface{}
        argIndex := 1

        if req.Title != nil {
                setParts = append(setParts, "title = $"+strconv.Itoa(argIndex))
                args = append(args, *req.Title)
                argIndex++
        }

        if req.Description != nil {
                setParts = append(setParts, "description = $"+strconv.Itoa(argIndex))
                args = append(args, *req.Description)
                argIndex++
        }

        if req.Status != nil {
                setParts = append(setParts, "status = $"+strconv.Itoa(argIndex))
                args = append(args, *req.Status)
                argIndex++
        }

        if req.Priority != nil {
                setParts = append(setParts, "priority = $"+strconv.Itoa(argIndex))
                args = append(args, *req.Priority)
                argIndex++
        }

        if req.AssigneeID != nil {
                setParts = append(setParts, "assignee_id = $"+strconv.Itoa(argIndex))
                args = append(args, *req.AssigneeID)
                argIndex++
        }

        if req.DueDate != nil {
                setParts = append(setParts, "due_date = $"+strconv.Itoa(argIndex))
                args = append(args, *req.DueDate)
                argIndex++
        }

        if req.EstimatedHours != nil {
                setParts = append(setParts, "estimated_hours = $"+strconv.Itoa(argIndex))
                args = append(args, *req.EstimatedHours)
                argIndex++
        }

        if req.ActualHours != nil {
                setParts = append(setParts, "actual_hours = $"+strconv.Itoa(argIndex))
                args = append(args, *req.ActualHours)
                argIndex++
        }

        if req.Tags != nil {
                setParts = append(setParts, "tags = $"+strconv.Itoa(argIndex))
                args = append(args, pq.Array(req.Tags))
                argIndex++
        }

        return strings.Join(setParts, ", "), args
}

func (h *TaskHandler) updateTaskMetrics() {
        // Query current task counts by status
        rows, err := h.db.Query(`
                SELECT status, project_id, COUNT(*) 
                FROM tasks 
                GROUP BY status, project_id
        `)
        if err != nil {
                return
        }
        defer rows.Close()

        for rows.Next() {
                var status string
                var projectID int
                var count int
                if err := rows.Scan(&status, &projectID, &count); err == nil {
                        h.metrics.UpdateTaskMetrics(status, projectID, float64(count))
                }
        }

        // Update active tasks count
        var activeCount int
        err = h.db.QueryRow(`
                SELECT COUNT(*) FROM tasks WHERE status IN ('todo', 'in_progress', 'review')
        `).Scan(&activeCount)
        if err == nil {
                h.metrics.UpdateActiveTasksMetric(float64(activeCount))
        }
}