"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { apiBase } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayMeal, MealType } from "@/features/display/model/display-types";
import { dayKeyAt, mealWindowAt } from "@/lib/display-logic";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

type RecipeChoice = { ID: string; Title: string; MealType?: string | null };

const mealTypes: MealType[] = ["breakfast", "lunch", "dinner"];
const mealLabels: Record<MealType, string> = { breakfast: "Breakfast", lunch: "Lunch", dinner: "Dinner" };

export function MealPlanApp({ token, timezone, initialMeal, clock, onDialogTitleChange }: KioskAppProps) {
  const today = dayKeyAt(clock, timezone);
  const selectedDate = initialMeal?.date ?? mealWindowAt(clock, timezone).date;
  const [selected, setSelected] = useState<MealType | "">(initialMeal?.mealType ?? "");
  const [customMeal, setCustomMeal] = useState(false);
  const [mealName, setMealName] = useState("");
  const [busy, setBusy] = useState("");
  const [editingDate, setEditingDate] = useState<string | null>(initialMeal?.date ?? null);
  const initialMealApplied = useRef(false);
  const activeDate = selected ? editingDate ?? initialMeal?.date ?? selectedDate : selectedDate;
  const { data: meals, error, setError, reload } = useKioskResource<DisplayMeal[]>(`/api/display/meals?week=${activeDate}`, token);
  const { data: recipes, error: recipesError, reload: reloadRecipes } = useKioskResource<RecipeChoice[]>("/api/display/recipes", token);

  useEffect(() => {
    if (!meals || !initialMeal || initialMealApplied.current) return;
    initialMealApplied.current = true;
    const meal = meals.find(item => item.Date === initialMeal.date && item.MealType === initialMeal.mealType);
    setMealName(meal?.Name ?? "");
  }, [initialMeal, meals]);

  useEffect(() => {
    onDialogTitleChange?.(selected ? `Edit ${selected}` : null);
    return () => onDialogTitleChange?.(null);
  }, [onDialogTitleChange, selected]);

  const dateLabel = new Intl.DateTimeFormat("en-GB", {
    weekday: "long", day: "numeric", month: "long", timeZone: timezone,
  }).format(new Date(`${selectedDate}T12:00:00Z`));
  const dayLabel = activeDate === today ? "today" : "tomorrow";

  function editMeal(mealType: MealType) {
    const meal = meals?.find(item => item.Date === selectedDate && item.MealType === mealType);
    setEditingDate(selectedDate);
    setSelected(mealType);
    setMealName(meal?.Name ?? "");
    setCustomMeal(false);
    setError("");
  }

  async function save(name: string, recipeID?: string) {
    if (!selected || !name.trim()) return;
    setBusy(recipeID ?? "custom");
    setError("");
    try {
      const response = await fetch(`${apiBase}/api/display/meals`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ date: activeDate, mealType: selected, name: name.trim(), ...(recipeID ? { recipeID } : {}) }),
      });
      if (!response.ok) throw new Error();
      setSelected("");
      setEditingDate(null);
      setCustomMeal(false);
      setMealName("");
      await reload();
    } catch {
      setError("We couldn’t save that meal. Please try again.");
    } finally {
      setBusy("");
    }
  }

  async function saveCustomMeal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await save(mealName);
  }

  if (error && !meals) return <p role="alert">{error}</p>;
  if (!meals) return <p role="status">Loading…</p>;

  if (selected) {
    const currentMeal = meals.find(item => item.Date === activeDate && item.MealType === selected);
    const typedRecipes = recipes?.filter(recipe => recipe.MealType?.toLowerCase() === selected) ?? [];
    const untypedRecipes = recipes?.filter(recipe => !recipe.MealType) ?? [];
    const choices = typedRecipes.length ? typedRecipes : untypedRecipes;
    const includesCurrent = currentMeal && !choices.some(recipe => recipe.Title === currentMeal.Name);

    return <div className="meal-edit-view">
      <p className="dialog-label">Choose a meal for {dayLabel}.</p>
      {recipesError && <p role="alert">{recipesError} <button className="text-action" onClick={() => void reloadRecipes()}>Try again</button></p>}
      {error && <p role="alert">{error}</p>}
      <div className="meal-choice-list">
        {includesCurrent && <button className="meal-choice" aria-pressed="true" disabled={Boolean(busy)} onClick={() => { setSelected(""); setEditingDate(null); setCustomMeal(false); }}>
          <span>{currentMeal.Name}</span><strong>Planned</strong>
        </button>}
        {choices.map(recipe => <button className="meal-choice" key={recipe.ID} aria-pressed={recipe.Title === currentMeal?.Name} disabled={Boolean(busy)} onClick={() => void save(recipe.Title, recipe.ID)}>
          <span>{recipe.Title}</span>{recipe.Title === currentMeal?.Name ? <strong>Planned</strong> : <span className="muted">{busy === recipe.ID ? "Saving…" : "Choose"}</span>}
        </button>)}
        <button className="meal-choice custom-meal-choice" onClick={() => { setCustomMeal(true); setMealName(""); }} disabled={Boolean(busy)}>
          <span>Something else</span><span className="muted">Type a meal</span>
        </button>
      </div>
      {customMeal && <form className="meal-custom-form" onSubmit={event => void saveCustomMeal(event)}>
        <label>Meal idea<input aria-label="Meal idea" autoFocus value={mealName} onChange={event => setMealName(event.target.value)} placeholder="Enter a meal" /></label>
        <button className="action-button" disabled={!mealName.trim() || Boolean(busy)}>{busy ? "Saving…" : "Save meal"}</button>
      </form>}
    </div>;
  }

  return <div className="meal-plan-app">
    <p className="dialog-label meal-day-label">{dateLabel} · For everyone</p>
    {error && <p role="alert">{error}</p>}
    {mealTypes.map(mealType => {
      const meal = meals.find(item => item.Date === selectedDate && item.MealType === mealType);
      return <div className="meal-plan-row" key={mealType}>
        <span>{mealLabels[mealType]}</span>
        <button className="meal-choice-trigger" onClick={() => editMeal(mealType)}>{meal?.Name ?? "Choose a meal"} <span className="muted">· Edit</span></button>
      </div>;
    })}
    <p className="dialog-label meal-edit-hint">Choose a meal to edit.</p>
  </div>;
}
