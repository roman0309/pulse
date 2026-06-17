package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/acme/observability/internal/domain/services"
	"github.com/acme/observability/internal/remote"
	"github.com/acme/observability/internal/repositories/postgres"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func badRequest(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func serverError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// handleDomainError maps domain/repository errors to HTTP status codes.
func handleDomainError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, postgres.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, remote.ErrBadTarget):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		serverError(c, err)
	}
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return uuid.Nil, false
	}
	return id, true
}

// parseTimeRange reads ?from and ?to (RFC3339) query params, defaulting to the
// last `defaultWindow`.
func parseTimeRange(c *gin.Context, defaultWindow time.Duration) (time.Time, time.Time) {
	now := time.Now()
	from := now.Add(-defaultWindow)
	to := now
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			from = t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			to = t
		}
	}
	return from, to
}

func queryInt(c *gin.Context, name string, def int) int {
	if v := c.Query(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func parseTimestamp(s string) time.Time {
	if s == "" {
		return time.Now()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Now()
}
