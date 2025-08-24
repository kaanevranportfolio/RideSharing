package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rideshare-platform/shared/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	// Generate UUID if not provided
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	query := `
		INSERT INTO users (id, email, phone, password_hash, first_name, last_name, user_type, status, profile_image_url, email_verified, phone_verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.Phone, user.PasswordHash,
		user.FirstName, user.LastName, user.UserType, user.Status,
		user.ProfileImageURL, user.EmailVerified, user.PhoneVerified,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUser(ctx context.Context, id string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, email, phone, password_hash, first_name, last_name, user_type, status, 
		       profile_image_url, email_verified, phone_verified, created_at, updated_at
		FROM users WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Phone, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.UserType, &user.Status,
		&user.ProfileImageURL, &user.EmailVerified, &user.PhoneVerified,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, email, phone, password_hash, first_name, last_name, user_type, status, 
		       profile_image_url, email_verified, phone_verified, created_at, updated_at
		FROM users WHERE email = $1`

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Phone, &user.PasswordHash,
		&user.FirstName, &user.LastName, &user.UserType, &user.Status,
		&user.ProfileImageURL, &user.EmailVerified, &user.PhoneVerified,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *UserRepository) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, phone, password_hash, first_name, last_name, user_type, status, 
		       profile_image_url, email_verified, phone_verified, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.Phone, &user.PasswordHash,
			&user.FirstName, &user.LastName, &user.UserType, &user.Status,
			&user.ProfileImageURL, &user.EmailVerified, &user.PhoneVerified,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET 
		    email = $2, phone = $3, password_hash = $4, first_name = $5, last_name = $6,
		    user_type = $7, status = $8, profile_image_url = $9, email_verified = $10,
		    phone_verified = $11, updated_at = $12
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		user.ID, user.Email, user.Phone, user.PasswordHash,
		user.FirstName, user.LastName, user.UserType, user.Status,
		user.ProfileImageURL, user.EmailVerified, user.PhoneVerified,
		user.UpdatedAt,
	).Scan(&user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
