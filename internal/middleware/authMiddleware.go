package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"uid"`
	Admin  bool   `json:"adm"`
	jwt.RegisteredClaims
}

func Middleware(secret string, requireAdmin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(*Claims)
		if requireAdmin && !claims.Admin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Set("uid", claims.UserID)
		c.Set("adm", claims.Admin)
		c.Next()
	}
}

// UserMiddleware is a simpler middleware that just requires authentication (not admin)
func UserMiddleware(secret string) gin.HandlerFunc {
	return Middleware(secret, false)
}

// AdminMiddleware requires admin privileges
func AdminMiddleware(secret string) gin.HandlerFunc {
	return Middleware(secret, true)
}

func Issue(secret, userID string, admin bool, ttl time.Duration) (string, error) {
	claims := &Claims{UserID: userID, Admin: admin, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
