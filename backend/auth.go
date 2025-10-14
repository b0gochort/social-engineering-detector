package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterParent handles the registration of the first (and only) parent user.
func RegisterParent(db *sqlx.DB, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Errorf("Failed to bind JSON for registration: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Implement logic to check if a user already exists
		// TODO: Implement password hashing (e.g., Argon2 or BCrypt)
		// TODO: Implement Data Key (DK) generation and encryption (DKenc)
		// TODO: Save user to database

		log.Info("User registration endpoint hit (logic not fully implemented yet)")
		c.JSON(http.StatusCreated, gin.H{"message": "Registration endpoint hit, logic to be implemented"})
	}
}
