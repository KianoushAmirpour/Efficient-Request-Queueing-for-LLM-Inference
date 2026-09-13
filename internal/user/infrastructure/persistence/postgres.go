package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"efficient-request-queueing-for-llm-inference/internal/shared/tx/postgres"
	"efficient-request-queueing-for-llm-inference/internal/user/domain"
)

type UserRepository struct {
	Db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{Db: db}
}

func (r *UserRepository) Save(ctx context.Context, userID string) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `INSERT INTO users (user_id) VALUES ($1)`
	_, err := db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, userUUID string) (*domain.User, error) {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `SELECT user_id, role, tier FROM users WHERE user_id = $1`
	var user domain.User
	err := db.QueryRow(ctx, query, userUUID).Scan(&user.UserID, &user.Role, &user.Tier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("retrieve user tier: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetUserRole(ctx context.Context, userUUID string) (string, error) {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `SELECT role FROM users WHERE user_id = $1`
	var role string
	err := db.QueryRow(ctx, query, userUUID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrUserNotFound
		}
		return "", fmt.Errorf("retrieve user role: %w", err)
	}

	return role, nil
}

func (r *UserRepository) SetUserAsAdmin(ctx context.Context, userID string) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `
		INSERT INTO users (user_id, role, tier) 
		VALUES ($1, 'admin', 'premium')
		ON CONFLICT (user_id) 
		DO UPDATE SET role = 'admin', tier = 'premium', updated_at = NOW()
	`
	_, err := db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("set user as admin: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateUserTier(ctx context.Context, userID string, tier string) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `UPDATE users SET tier = $1::user_tier, updated_at = NOW() WHERE user_id = $2`
	result, err := db.Exec(ctx, query, tier, userID)
	if err != nil {
		return fmt.Errorf("update user tier: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) IsUserAdmin(ctx context.Context, userID string) error {
	db := postgres.ExtractDB(ctx, r.Db)
	query := `SELECT role FROM users WHERE user_id = $1`
	var role string
	err := db.QueryRow(ctx, query, userID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("check user admin role: %w", err)
	}
	if role != string(domain.RoleAdmin) {
		return domain.ErrUserNotAdmin
	}
	return nil
}
