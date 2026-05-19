package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware provides JWT authentication for API routes.
// The middleware is fully functional for development using the configured
// JWT secret (from config.yaml or JWT_SECRET env var). In production, the
// JWT_SECRET must be set to match the main backend's signing key so that
// tokens issued by the main backend are accepted by this service.
// Full auth integration with the main backend's token refresh/validation
// endpoints will be implemented in a later phase.
type JWTAuthMiddleware struct {
	Secret string
}

// NewJWTAuthMiddleware creates a new JWTAuthMiddleware.
func NewJWTAuthMiddleware(secret string) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{Secret: secret}
}

// RequireAuth returns a Gin middleware handler that validates JWT tokens.
func (m *JWTAuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Missing authorization header",
			})
			return
		}

		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid authorization format, expected 'Bearer <token>'",
			})
			return
		}

		tokenString := parts[1]

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.Secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Invalid or expired token",
			})
			return
		}

		// Extract claims and set user info in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["sub"])
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}

// OptionalAuth is a middleware that checks auth but does not reject unauthenticated requests.
func (m *JWTAuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(m.Secret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				c.Set("user_id", claims["sub"])
				c.Set("username", claims["username"])
			}
		}

		c.Next()
	}
}
