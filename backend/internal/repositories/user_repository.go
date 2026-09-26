package repositories

import (
	"context"

	"github.com/ErenKarakus1/Real-Time-Chat-App/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	query := `
		INSERT INTO users (id, username, email, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, password_hash, created_at, updated_at
	`

	return r.scanUser(r.db.QueryRow(ctx, query, user.ID, user.Username, user.Email, user.PasswordHash))
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	return r.scanUser(r.db.QueryRow(ctx, query, id))
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	return r.scanUser(r.db.QueryRow(ctx, query, email))
}

func (r *UserRepository) Search(ctx context.Context, query string, limit int) ([]models.User, error) {
	sql := `
		SELECT id, username, email, password_hash, created_at, updated_at
		FROM users
		WHERE username ILIKE $1
			OR email ILIKE $1
		ORDER BY username ASC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, sql, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		user, err := r.scanUser(rows)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) scanUser(row pgx.Row) (models.User, error) {
	var user models.User

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}
