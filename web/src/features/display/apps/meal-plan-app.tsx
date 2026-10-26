"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { apiBase } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayMeal, MealType } from "@/features/display/model/display-types";
import { dayKeyAt } from "@/lib/display-logic";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

export function MealPlanApp({ token, timezone, initialMeal }: KioskAppProps) {
  const path = `/api/display/meals${initialMeal ? `?week=${initialMeal.date}` : ""}`;
  const { data: meals, error, setError, reload } = useKioskResource<DisplayMeal[]>(path, token);
  const [busy, setBusy] = useState("");
  const [selected, setSelected] = useState("");
  const [mealName, setMealName] = useState("");
  const [newMealDate, setNewMealDate] = useState(() => dayKeyAt(new Date(), timezone));
  const [newMealType, setNewMealType] = useState<MealType>("dinner");
  const [newMealName, setNewMealName] = useState("");
  const initialMealApplied = useRef(false);

  useEffect(() => {
    if (!meals || !initialMeal || initialMealApplied.current) return;
    initialMealApplied.current = true;
    setNewMealDate(initialMeal.date);
    setNewMealType(initialMeal.mealType);
    const meal = meals.find(item => item.Date === initialMeal.date && item.MealType === initialMeal.mealType);
    if (meal) { setSelected(meal.Date + meal.MealType); setMealName(meal.Name); }
  }, [initialMeal, meals]);

  async function save(event: FormEvent<HTMLFormElement>, date: string, mealType: MealType, name: string, closeEditor: boolean) {
    event.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) return;
    const key = closeEditor ? date + mealType : "new-meal";
    setBusy(key);
    setError("");
    try {
      const response = await fetch(`${apiBase}/api/display/meals`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ date, mealType, name: trimmedName }),
      });
      if (!response.ok) throw new Error();
      if (closeEditor) { setSelected(""); setMealName(""); }
      else setNewMealName("");
      await reload();
    } catch {
      setError("We couldn’t save that meal. Please try again.");
    } finally { setBusy(""); }
  }

  if (error && !meals) return <p role="alert">{error}</p>;
  if (!meals) return <p role="status">Loading…</p>;
  return <div className="kiosk-list">
    <form className="inline-form" onSubmit={event => void save(event, newMealDate, newMealType, newMealName, false)}>
      <label>Date<input aria-label="Meal date" type="date" value={newMealDate} onChange={event => setNewMealDate(event.target.value)} /></label>
      <label>Meal<select aria-label="Meal type" value={newMealType} onChange={event => setNewMealType(event.target.value as MealType)}>
        <option value="breakfast">Breakfast</option><option value="lunch">Lunch</option><option value="dinner">Dinner</option>
      </select></label>
      <label>Meal idea<input aria-label="Meal idea" value={newMealName} onChange={event => setNewMealName(event.target.value)} placeholder="Add a meal idea" /></label>
      <button className="action-button" disabled={!newMealName.trim() || busy === "new-meal"}>{busy === "new-meal" ? "Saving…" : "Add meal"}</button>
    </form>
    {error && <p role="alert">{error}</p>}
    {meals.map(meal => <section key={meal.Date + meal.MealType}>
      <p>{meal.Date} · {meal.MealType}</p>
      {selected === meal.Date + meal.MealType
        ? <form className="inline-form" onSubmit={event => void save(event, meal.Date, meal.MealType, mealName, true)}>
            <input aria-label="Meal name" value={mealName} onChange={event => setMealName(event.target.value)} />
            <button className="action-button" disabled={busy === selected}>Save</button>
            <button type="button" className="text-action" onClick={() => setSelected("")}>Cancel</button>
          </form>
        : <><strong>{meal.Name}</strong><button className="text-action" onClick={() => { setSelected(meal.Date + meal.MealType); setMealName(meal.Name); }}>Edit</button></>}
    </section>)}
  </div>;
}
