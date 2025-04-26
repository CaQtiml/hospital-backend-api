package middleware

import (
	"hospital-middleware/internal/services"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyClaims is the key used to store validated JWT claims in the Gin context.
	ContextKeyClaims = "userClaims"
)

// AuthRequired is a middleware function to verify JWT token.
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Println("Auth middleware: Missing Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// Expecting "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Println("Auth middleware: Invalid Authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		tokenString := parts[1]
		claims, err := services.ValidateToken(tokenString)
		if err != nil {
			log.Printf("Auth middleware: Token validation failed - %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()}) // e.g., "token is expired" or "invalid token"
			return
		}

		// Store claims in context for use by subsequent handlers
		c.Set(ContextKeyClaims, claims)
		log.Printf("Auth middleware: User %s (ID: %d, Hospital: %d) authorized", claims.Username, claims.UserID, claims.HospitalID)

		c.Next() // Proceed to the next handler
	}
}
