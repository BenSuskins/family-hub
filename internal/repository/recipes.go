package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type RecipeRepository interface {
	FindByID(ctx context.Context, id string) (models.Recipe, error)
	FindAll(ctx context.Context) ([]models.Recipe, error)
	Create(ctx context.Context, recipe models.Recipe) (models.Recipe, error)
	Update(ctx context.Context, recipe models.Recipe) error
	Delete(ctx context.Context, id string) error
	FindImageData(ctx context.Context, id string) (string, error)
	UpdateImage(ctx context.Context, id string, imageData string) error
	ClearImage(ctx context.Context, id string) error
}

type SQLiteRecipeRepository struct {
	database *sql.DB
}

func NewRecipeRepository(database *sql.DB) *SQLiteRecipeRepository {
	return &SQLiteRecipeRepository{database: database}
}

func (repository *SQLiteRecipeRepository) FindByID(ctx context.Context, id string) (models.Recipe, error) {
	var recipe models.Recipe
	var ingredientsJSON string
	var stepsJSON string
	var mealTypeRaw *string
	var hasImageInt int
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, title, ingredients, instructions, steps, servings, prep_time, cook_time,
			source_url, category_id, meal_type,
			CASE WHEN image_data != '' THEN 1 ELSE 0 END,
			created_by_user_id, created_at, updated_at
		FROM recipes WHERE id = ?`, id,
	).Scan(
		&recipe.ID, &recipe.Title, &ingredientsJSON, &recipe.Instructions, &stepsJSON,
		&recipe.Servings, &recipe.PrepTime, &recipe.CookTime,
		&recipe.SourceURL, &recipe.CategoryID, &mealTypeRaw, &hasImageInt,
		&recipe.CreatedByUserID, &recipe.CreatedAt, &recipe.UpdatedAt,
	)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("finding recipe by id: %w", err)
	}
	if err := json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients); err != nil {
		return models.Recipe{}, fmt.Errorf("unmarshalling ingredients: %w", err)
	}
	if err := json.Unmarshal([]byte(stepsJSON), &recipe.Steps); err != nil {
		return models.Recipe{}, fmt.Errorf("unmarshalling steps: %w", err)
	}
	if mealTypeRaw != nil {
		mt := models.RecipeMealType(*mealTypeRaw)
		recipe.MealType = &mt
	}
	recipe.HasImage = hasImageInt != 0
	return recipe, nil
}

func (repository *SQLiteRecipeRepository) FindAll(ctx context.Context) ([]models.Recipe, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, title, ingredients, servings, prep_time, cook_time,
			source_url, category_id, meal_type,
			CASE WHEN image_data != '' THEN 1 ELSE 0 END,
			created_by_user_id, created_at, updated_at
		FROM recipes ORDER BY title ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("finding recipes: %w", err)
	}
	defer rows.Close()

	var recipes []models.Recipe
	for rows.Next() {
		var recipe models.Recipe
		var ingredientsJSON string
		var mealTypeRaw *string
		var hasImageInt int
		if err := rows.Scan(
			&recipe.ID, &recipe.Title, &ingredientsJSON,
			&recipe.Servings, &recipe.PrepTime, &recipe.CookTime,
			&recipe.SourceURL, &recipe.CategoryID, &mealTypeRaw, &hasImageInt,
			&recipe.CreatedByUserID, &recipe.CreatedAt, &recipe.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning recipe: %w", err)
		}
		if err := json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients); err != nil {
			return nil, fmt.Errorf("unmarshalling ingredients: %w", err)
		}
		if mealTypeRaw != nil {
			mt := models.RecipeMealType(*mealTypeRaw)
			recipe.MealType = &mt
		}
		recipe.HasImage = hasImageInt != 0
		recipes = append(recipes, recipe)
	}
	return recipes, rows.Err()
}

func (repository *SQLiteRecipeRepository) Create(ctx context.Context, recipe models.Recipe) (models.Recipe, error) {
	if recipe.ID == "" {
		recipe.ID = uuid.New().String()
	}
	now := time.Now()
	recipe.CreatedAt = now
	recipe.UpdatedAt = now

	if recipe.Ingredients == nil {
		recipe.Ingredients = []models.IngredientGroup{}
	}
	if recipe.Steps == nil {
		recipe.Steps = []string{}
	}

	ingredientsJSON, err := json.Marshal(recipe.Ingredients)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("marshalling ingredients: %w", err)
	}
	stepsJSON, err := json.Marshal(recipe.Steps)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("marshalling steps: %w", err)
	}

	var mealTypeStr *string
	if recipe.MealType != nil {
		s := string(*recipe.MealType)
		mealTypeStr = &s
	}

	_, err = repository.database.ExecContext(ctx,
		`INSERT INTO recipes (id, title, ingredients, instructions, steps, servings, prep_time, cook_time,
			source_url, category_id, meal_type, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		recipe.ID, recipe.Title, string(ingredientsJSON), recipe.Instructions, string(stepsJSON),
		recipe.Servings, recipe.PrepTime, recipe.CookTime,
		recipe.SourceURL, recipe.CategoryID, mealTypeStr,
		recipe.CreatedByUserID, recipe.CreatedAt, recipe.UpdatedAt,
	)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("creating recipe: %w", err)
	}
	return recipe, nil
}

func (repository *SQLiteRecipeRepository) Update(ctx context.Context, recipe models.Recipe) error {
	recipe.UpdatedAt = time.Now()

	if recipe.Ingredients == nil {
		recipe.Ingredients = []models.IngredientGroup{}
	}
	if recipe.Steps == nil {
		recipe.Steps = []string{}
	}

	ingredientsJSON, err := json.Marshal(recipe.Ingredients)
	if err != nil {
		return fmt.Errorf("marshalling ingredients: %w", err)
	}
	stepsJSON, err := json.Marshal(recipe.Steps)
	if err != nil {
		return fmt.Errorf("marshalling steps: %w", err)
	}

	var mealTypeStr *string
	if recipe.MealType != nil {
		s := string(*recipe.MealType)
		mealTypeStr = &s
	}

	_, err = repository.database.ExecContext(ctx,
		`UPDATE recipes SET title = ?, ingredients = ?, steps = ?, servings = ?,
			prep_time = ?, cook_time = ?, source_url = ?, category_id = ?, meal_type = ?, updated_at = ?
		WHERE id = ?`,
		recipe.Title, string(ingredientsJSON), string(stepsJSON), recipe.Servings,
		recipe.PrepTime, recipe.CookTime, recipe.SourceURL, recipe.CategoryID,
		mealTypeStr, recipe.UpdatedAt, recipe.ID,
	)
	if err != nil {
		return fmt.Errorf("updating recipe: %w", err)
	}
	return nil
}

func (repository *SQLiteRecipeRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM recipes WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting recipe: %w", err)
	}
	return nil
}

func (repository *SQLiteRecipeRepository) FindImageData(ctx context.Context, id string) (string, error) {
	var imageData string
	err := repository.database.QueryRowContext(ctx,
		`SELECT image_data FROM recipes WHERE id = ?`, id,
	).Scan(&imageData)
	if err != nil {
		return "", fmt.Errorf("finding image data: %w", err)
	}
	return imageData, nil
}

func (repository *SQLiteRecipeRepository) UpdateImage(ctx context.Context, id string, imageData string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE recipes SET image_data = ?, updated_at = ? WHERE id = ?`,
		imageData, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating recipe image: %w", err)
	}
	return nil
}

func (repository *SQLiteRecipeRepository) ClearImage(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE recipes SET image_data = '', updated_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("clearing recipe image: %w", err)
	}
	return nil
}
