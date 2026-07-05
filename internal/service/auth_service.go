package service

import (
	"context"
	"errors"
	"os"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Errors
var (
	ErrEmailTaken = errors.New("email already registered")
	ErrUsernameTaken = errors.New("username already taken")
    ErrInvalidCreds  = errors.New("invalid email or password")
    ErrUserNotFound  = errors.New("user not found")
)

// JWT Claims
type Claims struct {
    UserID   string `json:"userId"`
    Username string `json:"username"`
    Email    string `json:"email"`
    Plan     string `json:"plan"`
    jwt.RegisteredClaims
}

// Token Generation
func generateToken(user *models.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	claims := Claims{
		UserID: user.ID.Hex(),
		Username: user.Username,
		Email: user.Email,
		Plan: user.Plan,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7 days
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   user.ID.Hex(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Register
type RegisterInput struct {
    Name     string
    Username string
    Email    string
    Password string
}

type AuthResult struct {
    Token string
    User  *models.User
}

func Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	// Check email not already taken
	existing, err := repository.FindUserByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	// Check username not already taken
	existingByUsername, err := repository.FindUserByUsername(ctx, input.Username)
	if err != nil {
		return nil, err
	}
	if existingByUsername != nil {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
        Name:         input.Name,
        Username:     input.Username,
        Email:        input.Email,
        PasswordHash: string(hash),
    }
	if err := repository.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, User: user}, nil
}

// Login
func Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := repository.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCreds // don't reveal whether email exists
	}

	// Compare hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCreds
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, User: user}, nil
}