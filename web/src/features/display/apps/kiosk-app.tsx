"use client";

import type { ComponentType } from "react";
import { KioskScratchpad } from "@/components/kiosk-scratchpad";
import type { KioskAppName, MealEditTarget } from "@/features/display/model/display-types";
import { CalendarApp } from "@/features/display/apps/calendar-app";
import { InventoryApp } from "@/features/display/apps/inventory-app";
import { MealPlanApp } from "@/features/display/apps/meal-plan-app";
import { ShoppingApp } from "@/features/display/apps/shopping-app";
import { RecipesApp } from "@/features/display/apps/recipes-app";
import { WeatherApp } from "@/features/display/apps/weather-app";

export type KioskAppProps = {
  token: string;
  timezone: string;
  clock: Date;
  initialMeal: MealEditTarget | null;
  onDialogTitleChange?: (title: string | null) => void;
};
type DataKioskAppName = Exclude<KioskAppName, "Notes">;

const dataApps: Record<DataKioskAppName, ComponentType<KioskAppProps>> = {
  Shopping: ShoppingApp,
  Calendar: CalendarApp,
  Recipes: RecipesApp,
  "Meal plan": MealPlanApp,
  Inventory: InventoryApp,
  Weather: WeatherApp,
};
const dataAppNames = Object.keys(dataApps) as DataKioskAppName[];

function isDataKioskAppName(app: string): app is DataKioskAppName {
  return dataAppNames.includes(app as DataKioskAppName);
}

export function KioskApp({ app, token, initialMeal, timezone, clock, onDialogTitleChange, onScratchpadSaveReady }: KioskAppProps & {
  app: string;
  onScratchpadSaveReady: (save: (() => Promise<void>) | null) => void;
}) {
  if (app === "Notes") return <KioskScratchpad token={token} onSaveReady={onScratchpadSaveReady} />;
  if (!isDataKioskAppName(app)) return null;
  const App = dataApps[app];
  return <App token={token} initialMeal={initialMeal} timezone={timezone} clock={clock} onDialogTitleChange={onDialogTitleChange} />;
}
