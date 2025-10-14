package repository

import (
	"backend/internal/models"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type AuthRepository interface {
	CreateUser(user *models.User) error
	GetUserByUsername(username string) (*models.User, error)
	CountUsers() (int, error)
}

type authRepository struct {
	db  *sqlx.DB
	log *logrus.Logger
}

func NewAuthRepository(db *sqlx.DB, log *logrus.Logger) AuthRepository {
	return &authRepository{db: db, log: log}
}

func (r *authRepository) CreateUser(user *models.User) error {
	query := `INSERT INTO users (username, password_hash, role, dk_encrypted) VALUES ($1, $2, $3, $4) RETURNING id, created_at`
	return r.db.QueryRowx(query, user.Username, user.PasswordHash, user.Role, user.DKEncrypted).StructScan(user)
}

func (r *authRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := `SELECT id, username, password_hash, role, dk_encrypted, created_at FROM users WHERE username = $1`
	err := r.db.Get(&user, query, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) CountUsers() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	err := r.db.Get(&count, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}
