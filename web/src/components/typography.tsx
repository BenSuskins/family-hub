import { createElement, type ElementType, type HTMLAttributes } from "react";

export const typographyVariants = [
  "pageTitle",
  "date",
  "greeting",
  "clock",
  "cardTitle",
  "cardHint",
  "departureTime",
  "departureTitle",
  "subtitle",
  "mealTitle",
  "compactMealTitle",
  "mealMetadata",
  "body",
  "caption",
  "dialogTitle",
  "dialogLabel",
  "detailTitle",
  "detailBody",
  "shoppingItem",
  "mealChoice",
  "mealChoiceStatus",
  "mealPlanValue",
  "mealPlanLabel",
  "mealChoiceHint",
  "customMealChoice",
  "inventoryItem",
  "inventoryQuantity",
  "inventoryUnit",
  "areaHeading",
  "weatherHour",
  "weatherTemperature",
  "forecastLabel",
  "forecastTemperature",
  "forecastRain",
  "calendarWeekday",
  "calendarDate",
  "calendarHour",
  "calendarEvent",
  "calendarEventTime",
  "displayEventTitle",
  "displayEventTime",
  "displayEventDate",
  "displayEventMeta",
] as const;

export type TypographyVariant = (typeof typographyVariants)[number];

export type TypographyProps = HTMLAttributes<HTMLElement> & {
  as?: ElementType;
  variant: TypographyVariant;
};

export function Typography({
  as = "span",
  className,
  variant,
  ...props
}: TypographyProps) {
  const variantClass = variant.replace(
    /[A-Z]/g,
    (letter) => `-${letter.toLowerCase()}`,
  );
  const classes = ["typography", `typography--${variantClass}`, className]
    .filter(Boolean)
    .join(" ");

  return createElement(as, { ...props, className: classes });
}
