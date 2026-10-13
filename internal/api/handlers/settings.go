package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
)

// GET /settings/profile
func GetProfile(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := repository.FindUserByIDRaw(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":          user.ID.Hex(),
			"name":        user.Name,
			"username":    user.Username,
			"email":       user.Email,
			"avatar":      user.Avatar,
			"bio":         user.Bio,
			"plan":        user.Plan,
			"isVerified":  user.IsVerified,
			"stats":       user.Stats,
			"preferences": user.Preferences,
			"subscription": user.Subscription,
			"aiUsage":     user.AIUsage,
			"wallet":      user.Wallet,
			"createdAt":   user.CreatedAt,
		},
	})
}

// PATCH /settings/profile
type UpdateProfileRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Bio      string `json:"bio"`
	Avatar   string `json:"avatar"`
}

func UpdateProfile(c *gin.Context) {
	userID := c.GetString("userID")
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If username is changing, check it isn't taken by someone else
	if req.Username != "" {
		existing, err := repository.FindUserByUsername(c.Request.Context(), req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check username"})
			return
		}
		if existing != nil && existing.ID.Hex() != userID {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
			return
		}
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Bio != "" || req.Bio == "" { // allow clearing bio
		updates["bio"] = req.Bio
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := repository.UpdateProfile(c.Request.Context(), userID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Profile updated"})
}

// PATCH /settings/preferences
func UpdatePreferences(c *gin.Context) {
	userID := c.GetString("userID")
	var prefs models.UserPreferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate allowed enum values
	if prefs.Theme != "" && prefs.Theme != "dark" && prefs.Theme != "light" && prefs.Theme != "system" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme value"})
		return
	}
	if prefs.DefaultVisibility != "" && prefs.DefaultVisibility != "public" && prefs.DefaultVisibility != "private" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid defaultVisibility value"})
		return
	}
	if err := repository.UpdatePreferences(c.Request.Context(), userID, prefs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Preferences saved"})
}

// POST /settings/security/change-password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword"     binding:"required,min=8"`
}

func ChangePassword(c *gin.Context) {
	userID := c.GetString("userID")
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := repository.FindUserByIDRaw(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// OAuth-only users have no password
	if user.PasswordHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot set password for OAuth accounts"})
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := repository.UpdatePassword(c.Request.Context(), userID, string(newHash)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password changed successfully"})
}

// GET /settings/security/sessions
func ListSessions(c *gin.Context) {
	userID := c.GetString("userID")
	user, err := repository.FindUserByIDRaw(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"sessions": user.Sessions}})
}

// DELETE /settings/security/sessions/:sessionId
func RevokeSession(c *gin.Context) {
	userID := c.GetString("userID")
	sessionID := c.Param("sessionId")
	if strings.TrimSpace(sessionID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId required"})
		return
	}
	if err := repository.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Session revoked"})
}

// DELETE /settings/security/sessions
func RevokeAllSessions(c *gin.Context) {
	userID := c.GetString("userID")
	// current session token is in X-Keep-Session header (sent by frontend)
	keepSession := c.GetHeader("X-Keep-Session")
	if err := repository.RevokeAllOtherSessions(c.Request.Context(), userID, keepSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke sessions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "All other sessions revoked"})
}

// DELETE /settings/account
type DeleteAccountRequest struct {
	Password     string `json:"password"`     // required for password-auth users
	Confirmation string `json:"confirmation"  binding:"required"` // must == "DELETE"
}

func DeleteAccount(c *gin.Context) {
	userID := c.GetString("userID")
	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Confirmation != "DELETE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Type DELETE to confirm account deletion"})
		return
	}

	user, err := repository.FindUserByIDRaw(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// For password accounts, verify password before deleting
	if user.PasswordHash != "" {
		if req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password required to delete account"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect password"})
			return
		}
	}

	if err := repository.DeleteAccount(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Account deleted permanently"})
}

func RecordLoginSession(c *gin.Context, userID string) {
	session := models.UserSession{
		SessionID: uuid.NewString(),
		UserAgent: c.Request.UserAgent(),
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}
	_ = repository.AddSession(c.Request.Context(), userID, session)
}