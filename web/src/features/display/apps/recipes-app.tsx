"use client";

import { CookingPot, ChevronLeft, ChevronRight } from "lucide-react";
import { useState } from "react";
import { Typography } from "@/components/typography";
import { displayRequest } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayRecipe } from "@/features/display/model/display-types";

type RecipeSummary = {
  ID: string;
  Title: string;
  MealType?: string | null;
  PrepTime?: string | null;
  CookTime?: string | null;
};

export function RecipesApp({ token }: { token: string }) {
  const { data: recipes, error } = useKioskResource<RecipeSummary[]>(
    "/api/display/recipes",
    token,
  );
  const [details, setDetails] = useState<Record<string, DisplayRecipe>>({});
  const [recipeErrors, setRecipeErrors] = useState<Record<string, string>>({});
  const [selectedRecipeID, setSelectedRecipeID] = useState("");
  const selectedRecipe = recipes?.find(
    (recipe) => recipe.ID === selectedRecipeID,
  );
  const detail = selectedRecipeID ? details[selectedRecipeID] : null;

  async function loadRecipe(id: string) {
    try {
      const recipe = await displayRequest<DisplayRecipe>(
        `/api/display/recipes/${encodeURIComponent(id)}`,
        token,
      );
      setDetails((current) => ({ ...current, [id]: recipe }));
      setRecipeErrors((current) => ({ ...current, [id]: "" }));
    } catch {
      setRecipeErrors((current) => ({
        ...current,
        [id]: "This recipe is unavailable right now.",
      }));
    }
  }

  function openRecipe(id: string) {
    setSelectedRecipeID(id);
    if (!details[id] && !recipeErrors[id]) void loadRecipe(id);
  }

  if (error && !recipes)
    return (
      <Typography as="p" variant="body" role="alert">
        {error}
      </Typography>
    );
  if (!recipes)
    return (
      <Typography as="p" variant="body" role="status">
        Loading…
      </Typography>
    );

  if (selectedRecipe) {
    return (
      <div className="recipe-detail">
        <button
          className="kiosk-back recipe-back"
          onClick={() => setSelectedRecipeID("")}
        >
          <ChevronLeft size={18} aria-hidden="true" />{" "}
          <Typography as="span" variant="caption">
            Recipe book
          </Typography>
        </button>
        <Typography as="p" variant="dialogLabel" className="dialog-label">
          {selectedRecipe.MealType
            ? `${formatMealType(selectedRecipe.MealType)} recipe`
            : "Family recipe"}
        </Typography>
        <Typography
          as="h2"
          variant="dialogTitle"
          className="recipe-detail-title"
        >
          {selectedRecipe.Title}
        </Typography>
        <RecipeDetails
          recipe={detail}
          error={recipeErrors[selectedRecipe.ID]}
          onRetry={() => void loadRecipe(selectedRecipe.ID)}
        />
      </div>
    );
  }

  return (
    <div className="kiosk-list recipe-list">
      <Typography as="p" variant="dialogLabel" className="dialog-label">
        Family favourites · {recipes.length} recipes
      </Typography>
      {recipes.map((recipe) => (
        <button
          className="recipe-row"
          key={recipe.ID}
          onClick={() => openRecipe(recipe.ID)}
        >
          <span className="recipe-row-icon">
            <CookingPot size={20} strokeWidth={1.7} aria-hidden="true" />
          </span>
          <span className="recipe-row-copy">
            <Typography as="span" variant="body" className="recipe-row-title">
              {recipe.Title}
            </Typography>
            <Typography as="span" variant="caption" className="recipe-row-meta">
              {recipeSummary(recipe)}
            </Typography>
          </span>
          <ChevronRight
            className="recipe-row-chevron"
            size={18}
            aria-hidden="true"
          />
        </button>
      ))}
    </div>
  );
}

function RecipeDetails({
  recipe,
  error,
  onRetry,
}: {
  recipe: DisplayRecipe | null;
  error?: string;
  onRetry: () => void;
}) {
  if (error)
    return (
      <>
        <Typography as="p" variant="body" role="alert">
          {error}
        </Typography>
        <button className="action-button" onClick={onRetry}>
          <Typography as="span" variant="caption">
            Try again
          </Typography>
        </button>
      </>
    );
  if (!recipe)
    return (
      <Typography as="p" variant="body" role="status">
        Loading recipe…
      </Typography>
    );
  const steps =
    recipe.Steps ?? (recipe.Instructions ? [recipe.Instructions] : []);
  return (
    <>
      {(recipe.Servings || recipe.PrepTime || recipe.CookTime) && (
        <Typography
          as="p"
          variant="detailBody"
          className="detail-body recipe-meta"
        >
          {[
            recipe.Servings ? `Serves ${recipe.Servings}` : "",
            recipe.PrepTime ? `${recipe.PrepTime} prep` : "",
            recipe.CookTime ? `${recipe.CookTime} cook` : "",
          ]
            .filter(Boolean)
            .join(" · ")}
        </Typography>
      )}
      <Typography as="h3" variant="cardTitle">
        Ingredients
      </Typography>
      <ul className="recipe-ingredients">
        {(recipe.Ingredients ?? [])
          .flatMap((group) => group.items)
          .map((ingredient, index) => (
            <Typography
              as="li"
              variant="detailBody"
              className="recipe-detail-line"
              key={index}
            >
              {ingredient}
            </Typography>
          ))}
      </ul>
      <Typography as="h3" variant="cardTitle">
        Method
      </Typography>
      <ol className="recipe-method">
        {steps.map((step, index) => (
          <Typography
            as="li"
            variant="detailBody"
            className="recipe-detail-line"
            key={index}
          >
            {step}
          </Typography>
        ))}
      </ol>
    </>
  );
}

function recipeSummary(recipe: RecipeSummary) {
  const minutes = [recipe.PrepTime, recipe.CookTime]
    .map(parseDurationMinutes)
    .filter(Number.isFinite);
  const totalMinutes = minutes.length
    ? minutes.reduce((total, value) => total + value, 0)
    : null;
  const mealType = recipe.MealType ? formatMealType(recipe.MealType) : "";
  return (
    [totalMinutes ? `${totalMinutes} minutes` : "", mealType]
      .filter(Boolean)
      .join(" · ") || "Family favourite"
  );
}

function parseDurationMinutes(value: string | null | undefined) {
  if (!value) return Number.NaN;
  const hours = Number(value.match(/(\d+)\s*hrs?/i)?.[1] ?? 0);
  const minutes = Number(value.match(/(\d+)\s*mins?/i)?.[1] ?? 0);
  if (hours || minutes) return hours * 60 + minutes;
  return Number.parseInt(value, 10);
}

function formatMealType(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1).toLowerCase();
}
