"use client";

import dynamic from "next/dynamic";
import { Typography } from "@/components/typography";
import { KioskScratchpad } from "@/components/kiosk-scratchpad";
import type { MealEditTarget } from "@/features/display/model/display-types";

type SharedAppProps = { token: string };
type ShoppingItem = {
  id: string;
  name: string;
  purchased: boolean;
  version: number;
};
type ShoppingAppProps = SharedAppProps & {
  items: ShoppingItem[] | null;
  error: string;
  reload: () => Promise<void>;
};
type CalendarAppProps = SharedAppProps & { timezone: string; clock: Date };
type WeatherAppProps = CalendarAppProps;
type MealPlanAppProps = WeatherAppProps & {
  initialMeal: MealEditTarget | null;
  onDialogTitleChange?: (title: string | null) => void;
};

const Shopping = dynamic<ShoppingAppProps>(
  () =>
    import("@/features/display/apps/shopping-app").then(
      (module) => module.ShoppingApp,
    ),
  { loading: LoadingApp },
);
const Calendar = dynamic<CalendarAppProps>(
  () =>
    import("@/features/display/apps/calendar-app").then(
      (module) => module.CalendarApp,
    ),
  { loading: LoadingApp },
);
const Recipes = dynamic<SharedAppProps>(
  () =>
    import("@/features/display/apps/recipes-app").then(
      (module) => module.RecipesApp,
    ),
  { loading: LoadingApp },
);
const MealPlan = dynamic<MealPlanAppProps>(
  () =>
    import("@/features/display/apps/meal-plan-app").then(
      (module) => module.MealPlanApp,
    ),
  { loading: LoadingApp },
);
const Inventory = dynamic<SharedAppProps>(
  () =>
    import("@/features/display/apps/inventory-app").then(
      (module) => module.InventoryApp,
    ),
  { loading: LoadingApp },
);
const Weather = dynamic<WeatherAppProps>(
  () =>
    import("@/features/display/apps/weather-app").then(
      (module) => module.WeatherApp,
    ),
  { loading: LoadingApp },
);

type KioskAppProps =
  | ({
      app: "Notes";
      onScratchpadSaveReady: (save: (() => Promise<void>) | null) => void;
    } & SharedAppProps)
  | ({
      app: "Shopping";
      items: ShoppingItem[] | null;
      error: string;
      reload: () => Promise<void>;
    } & SharedAppProps)
  | ({ app: "Inventory" } & SharedAppProps)
  | ({ app: "Calendar" } & CalendarAppProps)
  | ({ app: "Recipes" } & SharedAppProps)
  | ({ app: "Meal plan" } & MealPlanAppProps)
  | ({ app: "Weather" } & WeatherAppProps);

export function KioskApp(props: KioskAppProps) {
  if (props.app === "Notes")
    return (
      <KioskScratchpad
        token={props.token}
        onSaveReady={props.onScratchpadSaveReady}
      />
    );
  if (props.app === "Shopping")
    return (
      <Shopping
        token={props.token}
        items={props.items}
        error={props.error}
        reload={props.reload}
      />
    );
  if (props.app === "Calendar")
    return (
      <Calendar
        token={props.token}
        timezone={props.timezone}
        clock={props.clock}
      />
    );
  if (props.app === "Recipes") return <Recipes token={props.token} />;
  if (props.app === "Meal plan")
    return (
      <MealPlan
        token={props.token}
        timezone={props.timezone}
        clock={props.clock}
        initialMeal={props.initialMeal}
        onDialogTitleChange={props.onDialogTitleChange}
      />
    );
  if (props.app === "Inventory") return <Inventory token={props.token} />;
  if (props.app === "Weather")
    return (
      <Weather
        token={props.token}
        timezone={props.timezone}
        clock={props.clock}
      />
    );
  return null;
}

function LoadingApp() {
  return (
    <Typography as="p" variant="body" className="display-loading" role="status">
      Loading app…
    </Typography>
  );
}
