package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type CategoryRepository interface {
	FindByID(ctx context.Context, id string) (models.Category, error)
	FindAll(ctx context.Context) ([]models.Category, error)
	Create(ctx context.Context, category models.Category) (models.Category, error)
	Update(ctx context.Context, id string, name string) error
	Delete(ctx context.Context, id string) error
}

type SQLiteCategoryRepository struct {
	database *sql.DB
}

func NewCategoryRepository(database *sql.DB) *SQLiteCategoryRepository {
	return &SQLiteCategoryRepository{database: database}
}

func (repository *SQLiteCategoryRepository) FindByID(ctx context.Context, id string) (models.Category, error) {
	var category models.Category
	err := repository.database.QueryRowContext(ctx,
		"SELECT id, name, created_by_user_id, created_at FROM categories WHERE id = ?", id,
	).Scan(&category.ID, &category.Name, &category.CreatedByUserID, &category.CreatedAt)
	if err != nil {
		return models.Category{}, fmt.Errorf("finding category by id: %w", err)
	}
	return category, nil
}

func (repository *SQLiteCategoryRepository) FindAll(ctx context.Context) ([]models.Category, error) {
	rows, err := repository.database.QueryContext(ctx,
		"SELECT id, name, created_by_user_id, created_at FROM categories ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("finding all categories: %w", err)
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.CreatedByUserID, &category.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning category: %w", err)
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

func (repository *SQLiteCategoryRepository) Create(ctx context.Context, category models.Category) (models.Category, error) {
	if category.ID == "" {
		category.ID = uuid.New().String()
	}
	category.CreatedAt = time.Now()

	_, err := repository.database.ExecContext(ctx,
		"INSERT INTO categories (id, name, created_by_user_id, created_at) VALUES (?, ?, ?, ?)",
		category.ID, category.Name, category.CreatedByUserID, category.CreatedAt,
	)
	if err != nil {
		return models.Category{}, fmt.Errorf("creating category: %w", err)
	}
	return category, nil
}

func (repository *SQLiteCategoryRepository) Update(ctx context.Context, id string, name string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE categories SET name = ? WHERE id = ?", name, id,
	)
	if err != nil {
		return fmt.Errorf("updating category: %w", err)
	}
	return nil
}

func (repository *SQLiteCategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting category: %w", err)
	}
	return nil
}
