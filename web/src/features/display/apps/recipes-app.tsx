"use client";

import { useState } from "react";
import { displayRequest } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayRecipe } from "@/features/display/model/display-types";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

type RecipeSummary = { ID: string; Title: string };

export function RecipesApp({ token }: KioskAppProps) {
  const { data: recipes, error } = useKioskResource<RecipeSummary[]>("/api/display/recipes", token);
  const [details, setDetails] = useState<Record<string, DisplayRecipe>>({});
  const [recipeErrors, setRecipeErrors] = useState<Record<string, string>>({});

  async function loadRecipe(id: string) {
    try {
      const recipe = await displayRequest<DisplayRecipe>(`/api/display/recipes/${id}`, token);
      setDetails(current => ({ ...current, [id]: recipe }));
    } catch {
      setRecipeErrors(current => ({ ...current, [id]: "This recipe is unavailable right now." }));
    }
  }

  if (error && !recipes) return <p role="alert">{error}</p>;
  if (!recipes) return <p role="status">Loading…</p>;
  return <div className="kiosk-list">
    <p className="dialog-label">Family favourites · {recipes.length} recipes</p>
    {recipes.map(recipe => {
      const detail = details[recipe.ID];
      return <details key={recipe.ID} onToggle={event => {
        if (event.currentTarget.open && !detail && !recipeErrors[recipe.ID]) void loadRecipe(recipe.ID);
      }}>
        <summary>{recipe.Title}</summary>
        {recipeErrors[recipe.ID]
          ? <><p role="alert">{recipeErrors[recipe.ID]}</p><button className="kiosk-back" onClick={() => {
            setRecipeErrors(current => ({ ...current, [recipe.ID]: "" }));
            void loadRecipe(recipe.ID);
          }}>Try again</button></>
          : !detail
            ? <p role="status">Loading recipe…</p>
            : <><h3>Ingredients</h3>
              {(detail.Ingredients ?? []).flatMap(group => group.items).map((ingredient, index) => <p key={index}>{ingredient}</p>)}
              <h3>Method</h3>
              {(detail.Steps ?? (detail.Instructions ? [detail.Instructions] : [])).map((step, index) => <p key={index}>{index + 1}. {step}</p>)}
            </>}
      </details>;
    })}
  </div>;
}
