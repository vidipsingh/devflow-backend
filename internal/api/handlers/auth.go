package handlers

import (
    "errors"
    "net/http"

    api "devflow-backend/internal/api/response"
    "devflow-backend/internal/service"

    "github.com/gin-gonic/gin"
)

type RegisterRequest struct {
    Name     string `json:"name"     binding:"required,min=2"`
    Username string `json:"username" binding:"required,min=3,max=30"`
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
    Token string `json:"token"`
    User  struct {
        ID       string `json:"id"`
        Name     string `json:"name"`
        Username string `json:"username"`
        Email    string `json:"email"`
        Plan     string `json:"plan"`
    } `json:"user"`
}

// Register
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := service.Register(c.Request.Context(), service.RegisterInput{
		Name:     req.Name,
        Username: req.Username,
        Email:    req.Email,
        Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		case errors.Is(err, service.ErrUsernameTaken):
			c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
		default:
			api.InternalError(c, "Registration failed")
		}
		return
	}

	var resp AuthResponse
    resp.Token = result.Token
    resp.User.ID       = result.User.ID.Hex()
    resp.User.Name     = result.User.Name
    resp.User.Username = result.User.Username
    resp.User.Email    = result.User.Email
    resp.User.Plan     = result.User.Plan

	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// Login
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCreds) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		api.InternalError(c, "Login Failed")
		return
	}

	var resp AuthResponse
    resp.Token = result.Token
    resp.User.ID       = result.User.ID.Hex()
    resp.User.Name     = result.User.Name
    resp.User.Username = result.User.Username
    resp.User.Email    = result.User.Email
    resp.User.Plan     = result.User.Plan

    api.OK(c, resp)
}