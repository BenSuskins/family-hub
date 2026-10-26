import { CalendarDays, CarFront, Cloud, CloudFog, CloudRain, CloudSnow, CloudSun, ListChecks, Monitor, ShoppingBag, Sun, TrainFront, Trash2, Utensils, type LucideIcon } from "lucide-react";

const sectionIcons: Record<string, LucideIcon> = {
  calendar: CalendarDays,
  meals: Utensils,
  chores: ListChecks,
  weather: Sun,
  departures: CarFront,
  trains: TrainFront,
  bins: Trash2,
  shopping: ShoppingBag,
};

export function SectionIcon({ name, size = 21 }: { name: string; size?: number }) {
  const Icon = sectionIcons[name] ?? Monitor;
  return <Icon size={size} strokeWidth={1.7} aria-hidden="true" className="display-icon" />;
}

export function WeatherIcon({ code, size = 21 }: { code: number; size?: number }) {
  const Icon = code === 0 ? Sun
    : [1, 2].includes(code) ? CloudSun
      : code === 3 ? Cloud
        : [45, 48].includes(code) ? CloudFog
          : [71, 73, 75, 77, 85, 86].includes(code) ? CloudSnow
            : CloudRain;
  return <Icon size={size} strokeWidth={1.7} aria-hidden="true" />;
}
