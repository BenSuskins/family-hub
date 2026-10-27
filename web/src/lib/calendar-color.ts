const subscriptionColors: Record<string, string> = {
  indigo: "#6366F1",
  violet: "#8B5CF6",
  rose: "#F43F5E",
  red: "#EF4444",
  orange: "#F97316",
  amber: "#F59E0B",
  emerald: "#10B981",
  teal: "#14B8A6",
  sky: "#0EA5E9",
  blue: "#3B82F6",
};

export function calendarEventColor(value?: string | null) {
  if (!value) return subscriptionColors.indigo;
  const namedColor = subscriptionColors[value.toLowerCase()];
  if (namedColor) return namedColor;
  if (/^#(?:[\da-f]{3}|[\da-f]{6})$/i.test(value)) return value;
  return subscriptionColors.indigo;
}
