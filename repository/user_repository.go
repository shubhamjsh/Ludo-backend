package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"Ludo/database"
	"Ludo/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrDuplicateEmail    = errors.New("email already exists")
	ErrDuplicatePhone    = errors.New("phone already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: database.DB}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (name, phone, email, password, avatar_url, coins, is_guest, created_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err := r.db.QueryRow(
		query,
		user.Name,
		user.Phone,
		user.Email,
		user.Password,
		user.AvatarURL,
		user.Coins,
		user.IsGuest,
		user.CreatedAt,
		user.LastLoginAt,
	).Scan(&user.ID)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
			return ErrDuplicateEmail
		}
		if err.Error() == `pq: duplicate key value violates unique constraint "users_phone_key"` {
			return ErrDuplicatePhone
		}
		return fmt.Errorf("error creating user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetUserByID(id int64) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password, avatar_url, coins, is_guest, created_at, last_login_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.Password,
		&user.AvatarURL,
		&user.Coins,
		&user.IsGuest,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password, avatar_url, coins, is_guest, created_at, last_login_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.Password,
		&user.AvatarURL,
		&user.Coins,
		&user.IsGuest,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByPhone(phone string) (*models.User, error) {
	query := `
		SELECT id, name, phone, email, password, avatar_url, coins, is_guest, created_at, last_login_at
		FROM users
		WHERE phone = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, phone).Scan(
		&user.ID,
		&user.Name,
		&user.Phone,
		&user.Email,
		&user.Password,
		&user.AvatarURL,
		&user.Coins,
		&user.IsGuest,
		&user.CreatedAt,
		&user.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) UpdateLastLogin(userID int64) error {
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`
	_, err := r.db.Exec(query, time.Now(), userID)
	return err
}

func (r *UserRepository) UpdateProfile(userID int64, name, avatarURL *string) error {
	query := `UPDATE users SET `
	args := []interface{}{}
	argCount := 1

	if name != nil {
		query += fmt.Sprintf("name = $%d, ", argCount)
		args = append(args, *name)
		argCount++
	}

	if avatarURL != nil {
		query += fmt.Sprintf("avatar_url = $%d, ", argCount)
		args = append(args, *avatarURL)
		argCount++
	}

	// Remove trailing comma and space
	query = query[:len(query)-2]
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, userID)

	_, err := r.db.Exec(query, args...)
	return err
}

