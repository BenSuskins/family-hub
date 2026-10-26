"use client";

import { CookingPot, ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";
import { Typography } from "@/components/typography";
import { displayRequest } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayRecipe } from "@/features/display/model/display-types";

type RecipeSummary = { ID: string; Title: string };

export function RecipesApp({
  token,
  onDialogTitleChange,
}: {
  token: string;
  onDialogTitleChange?: (title: string | null) => void;
}) {
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

  useEffect(() => {
    onDialogTitleChange?.(selectedRecipe?.Title ?? null);
    return () => onDialogTitleChange?.(null);
  }, [onDialogTitleChange, selectedRecipe?.Title]);

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
          Family recipe
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
          <Typography as="span" variant="body" className="recipe-row-title">
            {recipe.Title}
          </Typography>
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
      {(recipe.Ingredients ?? [])
        .flatMap((group) => group.items)
        .map((ingredient, index) => (
          <Typography
            as="p"
            variant="detailBody"
            className="recipe-detail-line"
            key={index}
          >
            {ingredient}
          </Typography>
        ))}
      <Typography as="h3" variant="cardTitle">
        Method
      </Typography>
      {steps.map((step, index) => (
        <Typography
          as="p"
          variant="detailBody"
          className="recipe-detail-line"
          key={index}
        >
          {index + 1}. {step}
        </Typography>
      ))}
    </>
  );
}
