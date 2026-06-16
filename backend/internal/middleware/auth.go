package middleware

import (
	"net/http"
	"strings"

	"github.com/acme/observability/internal/domain/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const ctxUserID = "user_id"
const ctxEmail = "email"

// Auth validates the Bearer access token and stores the user id in context.
func Auth(tokens *services.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Browsers cannot set headers on a WebSocket handshake, so we also
		// accept the access token via the ?token= query parameter.
		raw := ""
		header := c.GetHeader("Authorization")
		if parts := strings.SplitN(header, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			raw = parts[1]
		} else if q := c.Query("token"); q != "" {
			raw = q
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims, err := tokens.ParseAccess(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxEmail, claims.Email)
		c.Next()
	}
}

// UserID extracts the authenticated user id from the gin context.
func UserID(c *gin.Context) uuid.UUID {
	v, _ := c.Get(ctxUserID)
	id, _ := v.(uuid.UUID)
	return id
}
