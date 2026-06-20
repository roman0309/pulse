package middleware

import (
	"net/http"
	"strings"

	"github.com/acme/observability/internal/domain/repositories"
	"github.com/acme/observability/pkg/hash"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ctxProjectID = "ingest_project_id"
	ctxKeyID     = "ingest_key_id"
)

// IngestAuth authenticates ingestion requests via a per-project ingest key.
// The key is read from `X-Pulse-Key` or `Authorization: Bearer <key>` and
// resolved to a project. Only the SHA-256 hash is compared.
func IngestAuth(keys repositories.IngestKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-Pulse-Key")
		if raw == "" {
			if parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2); len(parts) == 2 {
				raw = parts[1]
			}
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing ingest key"})
			return
		}
		projectID, keyID, err := keys.ResolveProject(c.Request.Context(), hash.SHA256(raw))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ingest key"})
			return
		}
		c.Set(ctxProjectID, projectID)
		c.Set(ctxKeyID, keyID)
		c.Next()
	}
}

// IngestProjectID returns the project resolved from the ingest key.
func IngestProjectID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ctxProjectID)
	id, _ := v.(uuid.UUID)
	return id
}

// IngestKeyID returns the ingest key id the request authenticated with.
func IngestKeyID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ctxKeyID)
	id, _ := v.(uuid.UUID)
	return id
}
