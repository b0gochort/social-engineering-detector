package handler

import (
	"errors"
	"net/http"

	"backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler interface {
	RegisterParent(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	// TODO: Add ChangePassword handlers
}

func (h *authHandler) Logout(c *gin.Context) {
	username := c.MustGet("username").(string)

	err := h.authService.Logout(username)
	if err != nil {
		h.logger.Error("Failed to logout user", zap.String("username", username), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

type authHandler struct {
	authService service.AuthService
	logger      *zap.Logger
}

func NewAuthHandler(authService service.AuthService, logger *zap.Logger) AuthHandler {
	return &authHandler{authService: authService, logger: logger}
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *authHandler) RegisterParent(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON for registration", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.RegisterParent(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		h.logger.Error("Failed to register parent", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Parent registered successfully",
		"username": user.Username,
		"id":       user.ID,
	})
}

func (h *authHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON for login", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenString, expirationTime, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		h.logger.Error("Failed to login user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Login successful",
		"token":      tokenString,
		"expires_at": expirationTime,
	})
}
