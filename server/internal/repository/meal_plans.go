package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mealapp "github.com/bensuskins/family-hub/internal/application/meals"
)

type SQLiteMealPlanRepository struct {
	database *sql.DB
}

func NewMealPlanRepository(database *sql.DB) *SQLiteMealPlanRepository {
	return &SQLiteMealPlanRepository{database: database}
}

func (repository *SQLiteMealPlanRepository) FindByDateAndType(ctx context.Context, date string, mealType mealapp.Type) (mealapp.Meal, error) {
	var meal mealRecord
	err := repository.database.QueryRowContext(ctx,
		`SELECT date, meal_type, recipe_id, name, notes, COALESCE(created_by_user_id,''), created_at, updated_at
		FROM meal_plans WHERE date = ? AND meal_type = ?`, date, mealType,
	).Scan(
		&meal.Date, &meal.MealType, &meal.RecipeID, &meal.Name,
		&meal.Notes, &meal.CreatedByUserID, &meal.CreatedAt, &meal.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mealapp.Meal{}, fmt.Errorf("finding meal plan: %w", mealapp.ErrNotFound)
		}
		return mealapp.Meal{}, fmt.Errorf("finding meal plan: %w", err)
	}
	return mealFromRecord(meal), nil
}

func (repository *SQLiteMealPlanRepository) FindAll(ctx context.Context, filter mealapp.Filter) ([]mealapp.Meal, error) {
	query := `SELECT date, meal_type, recipe_id, name, notes, COALESCE(created_by_user_id,''), created_at, updated_at
	FROM meal_plans WHERE 1=1`

	var args []any

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

	return scanMealPlans(rows)
}

func (repository *SQLiteMealPlanRepository) FindByDate(ctx context.Context, date string) ([]mealapp.Meal, error) {
	return repository.FindAll(ctx, mealapp.Filter{DateFrom: date, DateTo: date})
}

func (repository *SQLiteMealPlanRepository) Upsert(ctx context.Context, input mealapp.Meal) error {
	meal := mealToRecord(input)
	now := time.Now()
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO meal_plans (date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?)
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

func (repository *SQLiteMealPlanRepository) Delete(ctx context.Context, date string, mealType mealapp.Type) error {
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

func scanMealPlans(rows *sql.Rows) ([]mealapp.Meal, error) {
	var meals []mealapp.Meal
	for rows.Next() {
		var meal mealRecord
		if err := rows.Scan(
			&meal.Date, &meal.MealType, &meal.RecipeID, &meal.Name,
			&meal.Notes, &meal.CreatedByUserID, &meal.CreatedAt, &meal.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning meal plan: %w", err)
		}
		meals = append(meals, mealFromRecord(meal))
	}
	return meals, rows.Err()
}

type mealRecord struct {
	Date                         string
	MealType                     string
	RecipeID                     *string
	Name, Notes, CreatedByUserID string
	CreatedAt, UpdatedAt         time.Time
}

func mealFromRecord(record mealRecord) mealapp.Meal {
	return mealapp.Meal{Date: record.Date, MealType: mealapp.Type(record.MealType), RecipeID: copyMealID(record.RecipeID), Name: record.Name, Notes: record.Notes, CreatedByUserID: record.CreatedByUserID, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func mealToRecord(meal mealapp.Meal) mealRecord {
	return mealRecord{Date: meal.Date, MealType: string(meal.MealType), RecipeID: copyMealID(meal.RecipeID), Name: meal.Name, Notes: meal.Notes, CreatedByUserID: meal.CreatedByUserID, CreatedAt: meal.CreatedAt, UpdatedAt: meal.UpdatedAt}
}
func copyMealID(value *string) *string {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
