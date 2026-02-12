package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type MealPlanFilter struct {
	DateFrom string
	DateTo   string
}

type MealPlanRepository interface {
	FindByDateAndType(ctx context.Context, date string, mealType models.MealType) (models.MealPlan, error)
	FindAll(ctx context.Context, filter MealPlanFilter) ([]models.MealPlan, error)
	FindByDate(ctx context.Context, date string) ([]models.MealPlan, error)
	Upsert(ctx context.Context, meal models.MealPlan) error
	Delete(ctx context.Context, date string, mealType models.MealType) error
	ClearRecipeID(ctx context.Context, recipeID string) error
}

type SQLiteMealPlanRepository struct {
	database *sql.DB
}

func NewMealPlanRepository(database *sql.DB) *SQLiteMealPlanRepository {
	return &SQLiteMealPlanRepository{database: database}
}

func (repository *SQLiteMealPlanRepository) FindByDateAndType(ctx context.Context, date string, mealType models.MealType) (models.MealPlan, error) {
	var meal models.MealPlan
	err := repository.database.QueryRowContext(ctx,
		`SELECT date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at
		FROM meal_plans WHERE date = ? AND meal_type = ?`, date, mealType,
	).Scan(
		&meal.Date, &meal.MealType, &meal.RecipeID, &meal.Name,
		&meal.Notes, &meal.CreatedByUserID, &meal.CreatedAt, &meal.UpdatedAt,
	)
	if err != nil {
		return models.MealPlan{}, fmt.Errorf("finding meal plan: %w", err)
	}
	return meal, nil
}

func (repository *SQLiteMealPlanRepository) FindAll(ctx context.Context, filter MealPlanFilter) ([]models.MealPlan, error) {
	query := `SELECT date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at
	FROM meal_plans WHERE 1=1`

	var args []interface{}

	if filter.DateFrom != "" {
		query += " AND date >= ?"
		args = append(args, filter.DateFrom)
	}
	if filter.DateTo != "" {
		query += " AND date <= ?"
		args = append(args, filter.DateTo)
	}

	query += " ORDER BY date ASC, CASE meal_type WHEN 'breakfast' THEN 1 WHEN 'lunch' THEN 2 WHEN 'dinner' THEN 3 END"

	rows, err := repository.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finding meal plans: %w", err)
	}
	defer rows.Close()

	var meals []models.MealPlan
	for rows.Next() {
		var meal models.MealPlan
		if err := rows.Scan(
			&meal.Date, &meal.MealType, &meal.RecipeID, &meal.Name,
			&meal.Notes, &meal.CreatedByUserID, &meal.CreatedAt, &meal.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning meal plan: %w", err)
		}
		meals = append(meals, meal)
	}
	return meals, rows.Err()
}

func (repository *SQLiteMealPlanRepository) FindByDate(ctx context.Context, date string) ([]models.MealPlan, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at
		FROM meal_plans WHERE date = ?
		ORDER BY CASE meal_type WHEN 'breakfast' THEN 1 WHEN 'lunch' THEN 2 WHEN 'dinner' THEN 3 END`,
		date,
	)
	if err != nil {
		return nil, fmt.Errorf("finding meal plans by date: %w", err)
	}
	defer rows.Close()

	var meals []models.MealPlan
	for rows.Next() {
		var meal models.MealPlan
		if err := rows.Scan(
			&meal.Date, &meal.MealType, &meal.RecipeID, &meal.Name,
			&meal.Notes, &meal.CreatedByUserID, &meal.CreatedAt, &meal.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning meal plan: %w", err)
		}
		meals = append(meals, meal)
	}
	return meals, rows.Err()
}

func (repository *SQLiteMealPlanRepository) Upsert(ctx context.Context, meal models.MealPlan) error {
	now := time.Now()
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO meal_plans (date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (date, meal_type) DO UPDATE SET
			recipe_id = excluded.recipe_id,
			name = excluded.name,
			notes = excluded.notes,
			updated_at = excluded.updated_at`,
		meal.Date, meal.MealType, meal.RecipeID, meal.Name, meal.Notes,
		meal.CreatedByUserID, now, now,
	)
	if err != nil {
		return fmt.Errorf("upserting meal plan: %w", err)
	}
	return nil
}

func (repository *SQLiteMealPlanRepository) Delete(ctx context.Context, date string, mealType models.MealType) error {
	_, err := repository.database.ExecContext(ctx,
		"DELETE FROM meal_plans WHERE date = ? AND meal_type = ?", date, mealType,
	)
	if err != nil {
		return fmt.Errorf("deleting meal plan: %w", err)
	}
	return nil
}

func (repository *SQLiteMealPlanRepository) ClearRecipeID(ctx context.Context, recipeID string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE meal_plans SET recipe_id = NULL, updated_at = ? WHERE recipe_id = ?",
		time.Now(), recipeID,
	)
	if err != nil {
		return fmt.Errorf("clearing recipe id from meal plans: %w", err)
	}
	return nil
}
