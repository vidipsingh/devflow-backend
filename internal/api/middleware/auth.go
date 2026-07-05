package middleware

import (
    "net/http"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   string `json:"userId"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Plan     string `json:"plan"`
    jwt.RegisteredClaims
}

// Validate Bearer token in Authorization header
func RequireAuth(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
		c.Abort()
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	secret := os.Getenv("JWT_SECRET")

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		c.Abort()
		return
	}

	claims := token.Claims.(*Claims)
	c.Set("userID",   claims.UserID)
    c.Set("username", claims.Username)
    c.Set("email",    claims.Email)
    c.Set("plan",     claims.Plan)

    c.Next()
}