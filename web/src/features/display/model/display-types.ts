import type { DisplayDeparture, DisplayEvent, MealType } from "@/lib/display-logic";

export type DisplayChore = { ID: string; Name: string; Status: string; DueDate?: string | null; AssignedToUserID?: string | null };
export type DisplayMeal = { Date: string; MealType: MealType; Name: string; RecipeID?: string | null };
export type DisplayRecipe = { ID: string; Title: string; Steps?: string[]; Instructions?: string; Ingredients?: { name: string; items: string[] }[]; Servings?: number | null; PrepTime?: string | null; CookTime?: string | null; MealType?: string | null };
export type Weather = { status: string; fetchedAt?: string; location?: string; advice?: string; current?: { temperature: number; code: number; summary?: string }; high?: number; low?: number; hours?: { time: string; temperature: number; code: number; rainProbability: number }[]; days?: { date: string; high: number; low: number; code: number; rainProbability?: number }[] };
export type BinSchedule = { collections?: { date: string; type: string; name: string }[]; enabled?: boolean };
export type CommuteBoard = { origin: { code: string }; destination: { code: string }; fetchedAt: string; trains: { serviceId: string; status: string; expectedDeparture?: string | null; scheduledDeparture?: string | null; platform?: string | null; operator?: string }[] };
export type CommuteResult = { status: string; timezone?: string; boards: CommuteBoard[] };
export type DisplayPerson = { id: string; name: string };
export type DisplaySettings = { id: string; name: string; enabledSections: string[]; commutePreferences?: unknown };
export type Snapshot = {
  schemaVersion: 2;
  displayId: string;
  householdName?: string;
  events: DisplayEvent[];
  chores: DisplayChore[];
  meals: DisplayMeal[];
  weather?: Weather;
  departures?: { status: string; items: DisplayDeparture[] };
  bins?: BinSchedule;
  commute?: CommuteResult;
  people: DisplayPerson[];
  enabledSections: string[];
  timezone: string;
  resourceUpdatedAt: Record<string, string>;
  availability: Record<string, string>;
};
export type Pairing = { challengeId: string; pairingCode: string; expiresAt: string };
export type MealEditTarget = { date: string; mealType: MealType };
export type EventDetails = { title: string; startTime: string; endTime?: string | null; location?: string; leaveAt?: string; drivingSeconds?: number };
export type KioskAppName = "Shopping" | "Notes" | "Calendar" | "Recipes" | "Meal plan" | "Inventory" | "Weather";
export type DisplaySelection =
  | { kind: "app"; app: KioskAppName | "launcher" }
  | { kind: "event"; event: EventDetails }
  | { kind: "chore"; chore: DisplayChore }
  | null;

export type { DisplayDeparture, DisplayEvent, MealType } from "@/lib/display-logic";
