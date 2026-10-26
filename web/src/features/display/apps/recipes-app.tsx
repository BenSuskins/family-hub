"use client";

import { CookingPot, ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";
import { displayRequest } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayRecipe } from "@/features/display/model/display-types";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

type RecipeSummary = { ID: string; Title: string };

export function RecipesApp({ token, onDialogTitleChange }: KioskAppProps) {
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
      {recipeErrors[selectedRecipe.ID]
        ? <><p role="alert">{recipeErrors[selectedRecipe.ID]}</p><button className="action-button" onClick={() => void loadRecipe(selectedRecipe.ID)}>Try again</button></>
        : !detail
          ? <p role="status">Loading recipe…</p>
          : <>
              {(detail.Servings || detail.PrepTime || detail.CookTime) && <p className="detail-body recipe-meta">{[
                detail.Servings ? `Serves ${detail.Servings}` : "",
                detail.PrepTime ? `${detail.PrepTime} prep` : "",
                detail.CookTime ? `${detail.CookTime} cook` : "",
              ].filter(Boolean).join(" · ")}</p>}
              <h3>Ingredients</h3>
              {(detail.Ingredients ?? []).flatMap(group => group.items).map((ingredient, index) => <p className="recipe-detail-line" key={index}>{ingredient}</p>)}
              <h3>Method</h3>
              {(detail.Steps ?? (detail.Instructions ? [detail.Instructions] : [])).map((step, index) => <p className="recipe-detail-line" key={index}>{index + 1}. {step}</p>)}
            </>}
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
