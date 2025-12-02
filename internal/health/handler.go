package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sangkips/campaign-dispatch-service/internal/queue"
)

type Handler struct {
	db       *sql.DB
	rabbitMQ *queue.RabbitMQ
}

func NewHandler(db *sql.DB, rabbitMQ *queue.RabbitMQ) *Handler {
	return &Handler{
		db:       db,
		rabbitMQ: rabbitMQ,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string           `json:"status"`
	Checks    map[string]Check `json:"checks"`
	Timestamp time.Time        `json:"timestamp"`
}

// Check represents a single health check
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Health performs health checks on database and RabbitMQ
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]Check)
	overallHealthy := true

	// Check database connectivity
	dbCheck := h.checkDatabase(ctx)
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallHealthy = false
	}

	// Check RabbitMQ connectivity
	queueCheck := h.checkQueue()
	checks["queue"] = queueCheck
	if queueCheck.Status != "healthy" {
		overallHealthy = false
	}

	// Determine overall status
	status := "healthy"
	if !overallHealthy {
		status = "unhealthy"
	}

	response := HealthResponse{
		Status:    status,
		Checks:    checks,
		Timestamp: time.Now(),
	}

	// Set HTTP status code
	statusCode := http.StatusOK
	if !overallHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// checkDatabase checks if the database is accessible
func (h *Handler) checkDatabase(ctx context.Context) Check {
	if h.db == nil {
		return Check{
			Status:  "unhealthy",
			Message: "database connection is nil",
		}
	}

	// Try to ping the database
	err := h.db.PingContext(ctx)
	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: "database connection failed: " + err.Error(),
		}
	}

	// Try a simple query to verify database is actually working
	var result int
	err = h.db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: "database query failed: " + err.Error(),
		}
	}

	return Check{
		Status:  "healthy",
		Message: "database is accessible",
	}
}

// checkQueue checks if RabbitMQ is accessible
func (h *Handler) checkQueue() Check {
	if h.rabbitMQ == nil {
		return Check{
			Status:  "unhealthy",
			Message: "queue connection is nil",
		}
	}

	if err := h.rabbitMQ.Ping(); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: "queue connection failed: " + err.Error(),
		}
	}

	return Check{
		Status:  "healthy",
		Message: "queue is accessible",
	}
}
