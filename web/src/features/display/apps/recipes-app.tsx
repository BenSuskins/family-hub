"use client";

import { CookingPot, ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";
import { displayRequest } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayRecipe } from "@/features/display/model/display-types";

type RecipeSummary = { ID: string; Title: string };

export function RecipesApp({ token, onDialogTitleChange }: { token: string; onDialogTitleChange?: (title: string | null) => void }) {
  const { data: recipes, error } = useKioskResource<RecipeSummary[]>("/api/display/recipes", token);
  const [details, setDetails] = useState<Record<string, DisplayRecipe>>({});
  const [recipeErrors, setRecipeErrors] = useState<Record<string, string>>({});
  const [selectedRecipeID, setSelectedRecipeID] = useState("");
  const selectedRecipe = recipes?.find(recipe => recipe.ID === selectedRecipeID);
  const detail = selectedRecipeID ? details[selectedRecipeID] : null;

  useEffect(() => {
    onDialogTitleChange?.(selectedRecipe?.Title ?? null);
    return () => onDialogTitleChange?.(null);
  }, [onDialogTitleChange, selectedRecipe?.Title]);

  async function loadRecipe(id: string) {
    try {
      const recipe = await displayRequest<DisplayRecipe>(`/api/display/recipes/${encodeURIComponent(id)}`, token);
      setDetails(current => ({ ...current, [id]: recipe }));
      setRecipeErrors(current => ({ ...current, [id]: "" }));
    } catch {
      setRecipeErrors(current => ({ ...current, [id]: "This recipe is unavailable right now." }));
    }
  }

  function openRecipe(id: string) {
    setSelectedRecipeID(id);
    if (!details[id] && !recipeErrors[id]) void loadRecipe(id);
  }

  if (error && !recipes) return <p role="alert">{error}</p>;
  if (!recipes) return <p role="status">Loading…</p>;

  if (selectedRecipe) {
    return <div className="recipe-detail">
      <button className="kiosk-back recipe-back" onClick={() => setSelectedRecipeID("")}><ChevronLeft size={18} aria-hidden="true" /> Recipe book</button>
      <p className="dialog-label">Family recipe</p>
      <RecipeDetails
        recipe={detail}
        error={recipeErrors[selectedRecipe.ID]}
        onRetry={() => void loadRecipe(selectedRecipe.ID)}
      />
    </div>;
  }

  return <div className="kiosk-list recipe-list">
    <p className="dialog-label">Family favourites · {recipes.length} recipes</p>
    {recipes.map(recipe => <button className="recipe-row" key={recipe.ID} onClick={() => openRecipe(recipe.ID)}>
      <span className="recipe-row-icon"><CookingPot size={20} strokeWidth={1.7} aria-hidden="true" /></span>
      <span className="recipe-row-title">{recipe.Title}</span>
      <ChevronRight className="recipe-row-chevron" size={18} aria-hidden="true" />
    </button>)}
  </div>;
}

function RecipeDetails({ recipe, error, onRetry }: { recipe: DisplayRecipe | null; error?: string; onRetry: () => void }) {
  if (error) return <><p role="alert">{error}</p><button className="action-button" onClick={onRetry}>Try again</button></>;
  if (!recipe) return <p role="status">Loading recipe…</p>;
  const steps = recipe.Steps ?? (recipe.Instructions ? [recipe.Instructions] : []);
  return <>
    {(recipe.Servings || recipe.PrepTime || recipe.CookTime) && <p className="detail-body recipe-meta">{[
      recipe.Servings ? `Serves ${recipe.Servings}` : "",
      recipe.PrepTime ? `${recipe.PrepTime} prep` : "",
      recipe.CookTime ? `${recipe.CookTime} cook` : "",
    ].filter(Boolean).join(" · ")}</p>}
    <h3>Ingredients</h3>
    {(recipe.Ingredients ?? []).flatMap(group => group.items).map((ingredient, index) => <p className="recipe-detail-line" key={index}>{ingredient}</p>)}
    <h3>Method</h3>
    {steps.map((step, index) => <p className="recipe-detail-line" key={index}>{index + 1}. {step}</p>)}
  </>;
}
