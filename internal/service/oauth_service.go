package service

import (
    "context"
    "fmt"
    "strings"

    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"
)

type OAuthUserInfo struct {
    Provider  string
    OAuthID   string
    Email     string
    Name      string
    Username  string
    AvatarURL string
}

func HandleOAuthUser(ctx context.Context, info OAuthUserInfo) (*AuthResult, error) {
	// Try to find by OAuth provider + ID
	user, err := repository.FindUserByOAuth(ctx, info.Provider, info.OAuthID)
	if err != nil {
		return nil, err
	}

	// If not found by OAuth ID, try by email
	if user == nil && info.Email != "" {
		user, err = repository.FindUserByEmail(ctx, info.Email)
		if err != nil {
			return nil, err
		}
		
		// If found by email, update their record to link this OAuth provider
		if user != nil {
			user.OAuthProvider = info.Provider
			user.OAuthID = info.OAuthID
			if user.Avatar == "" {
				user.Avatar = info.AvatarURL
			}
			if err := repository.UpdateUser(ctx, user); err != nil {
				return nil, err
			}
		}
	}

	 // If still not found then create new user
	if user == nil {
		username := sanitizeUsername(info.Username)
		// ensure username is unique
		username, err = ensureUniqueUsername(ctx, username)
		if err != nil {
			return nil, err
		}

		user = &models.User{
            Name:          info.Name,
            Username:      username,
            Email:         info.Email,
            OAuthProvider: info.Provider,
            OAuthID:       info.OAuthID,
            Avatar:        info.AvatarURL,
            IsVerified:    true,
            IsActive:      true,
        }
		if err := repository.CreateUser(ctx, user); err != nil {
			return nil, err
		}
	}

	token, err := generateToken(user)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, User: user}, nil
}

// Removes characters that aren't alphanumeric or hyphens.
func sanitizeUsername(raw string) string {
    var b strings.Builder
    for _, r := range strings.ToLower(raw) {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
            b.WriteRune(r)
        }
    }
    result := b.String()
    if result == "" {
        return "user"
    }
    return result
}

// Appends a number if the username is taken.
func ensureUniqueUsername(ctx context.Context, base string) (string, error) {
    candidate := base
    for i := 1; i <= 99; i++ {
        existing, err := repository.FindUserByUsername(ctx, candidate)
        if err != nil {
            return "", err
        }
        if existing == nil {
            return candidate, nil // available
        }
        candidate = fmt.Sprintf("%s%d", base, i)
    }
    return base, ErrUsernameTaken
}