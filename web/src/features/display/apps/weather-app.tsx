"use client";

import { CloudRain } from "lucide-react";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { Weather } from "@/features/display/model/display-types";
import { WeatherIcon } from "@/features/display/components/display-icons";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

export function WeatherApp({ token, timezone }: KioskAppProps) {
  const { data, error } = useKioskResource<Weather>("/api/display/weather", token);
  if (error && !data) return <p role="alert">{error}</p>;
  if (!data) return <p role="status">Loading…</p>;
  return <div className="weather-forecast">
    <p className="dialog-label">Today · {data.location ?? "At home"}</p>
    <div className="hourly-forecast">{data.hours?.map((hour, index) => <div className="hour-card" key={hour.time}>
      <span>{index === 0 ? "Now" : new Date(hour.time).toLocaleTimeString("en-GB", { hour: "2-digit", timeZone: timezone })}</span>
      <WeatherIcon code={hour.code} /><strong>{Math.round(hour.temperature)}°</strong><small>{hour.rainProbability === 0 ? "Dry" : `${Math.round(hour.rainProbability)}%`}</small>
    </div>)}</div>
    <p className="dialog-label forecast-days-label">Next seven days</p>
    <div className="weather-days">{data.days?.map(day => <div className="forecast-row" key={day.date}>
      <span>{new Date(`${day.date}T12:00:00`).toLocaleDateString("en-GB", { weekday: "short", day: "numeric", timeZone: timezone })}</span>
      <WeatherIcon code={day.code} />
      <strong>{Math.round(day.high)}° <span className="muted">{Math.round(day.low)}°</span></strong>
      <span className="forecast-rain"><CloudRain size={15} strokeWidth={1.7} aria-hidden="true" /> {Math.round(day.rainProbability ?? 0)}%</span>
    </div>)}</div>
    {data.advice && <p className="forecast-advice">{data.advice}</p>}
    <p className="dialog-label forecast-legend">High and low · Rain chance</p>
  </div>;
}
