package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"efficient-request-queueing-for-llm-inference/internal/auth/domain"
	"efficient-request-queueing-for-llm-inference/internal/shared/tx/postgres"
)

type oauthRepository struct {
	db *pgxpool.Pool
}

func NewOauthRepository(db *pgxpool.Pool) *oauthRepository {
	return &oauthRepository{db: db}
}

func (r *oauthRepository) FindByProviderUserID(ctx context.Context, provider, providerUserID string) (*domain.AuthenticatedUser, error) {
	db := postgres.ExtractDB(ctx, r.db)

	query := `SELECT user_id FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`
	var userUUID string
	err := db.QueryRow(ctx, query, provider, providerUserID).Scan(&userUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOauthAccountNotFound
		}
		return nil, fmt.Errorf("oauth account lookup: %w", err)
	}

	return &domain.AuthenticatedUser{UserID: userUUID}, nil
}

func (r *oauthRepository) Create(ctx context.Context, providerUserID, provider, userUUID string) (*domain.AuthenticatedUser, error) {
	db := postgres.ExtractDB(ctx, r.db)

	query := `INSERT INTO oauth_accounts (user_id, provider, provider_user_id)
               VALUES ($1, $2, $3)`
	_, err := db.Exec(ctx, query, userUUID, provider, providerUserID)
	if err != nil {
		return nil, fmt.Errorf("create OAuth account: %w", err)
	}

	return &domain.AuthenticatedUser{UserID: userUUID}, nil
}
