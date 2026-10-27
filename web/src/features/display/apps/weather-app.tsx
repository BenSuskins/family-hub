"use client";

import { CloudRain } from "lucide-react";
import { Typography } from "@/components/typography";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { Weather } from "@/features/display/model/display-types";
import { WeatherIcon } from "@/features/display/components/display-icons";
import { dayKeyAt } from "@/lib/display-logic";

export function WeatherApp({
  token,
  timezone,
  clock,
}: {
  token: string;
  timezone: string;
  clock: Date;
}) {
  const { data, error } = useKioskResource<Weather>(
    "/api/display/weather",
    token,
  );
  if (error && !data)
    return (
      <Typography as="p" variant="body" role="alert">
        {error}
      </Typography>
    );
  if (!data)
    return (
      <Typography as="p" variant="body" role="status">
        Loading…
      </Typography>
    );
  return (
    <div className="weather-forecast">
      <Typography as="p" variant="dialogLabel" className="dialog-label">
        Today · {data.location ?? "At home"}
      </Typography>
      <div className="hourly-forecast">
        {data.hours?.slice(0, 6).map((hour, index) => (
          <div className="hour-card" key={hour.time}>
            <Typography as="span" variant="weatherHour">
              {index === 0
                ? "Now"
                : `${new Date(hour.time).toLocaleTimeString("en-GB", { hour: "2-digit", timeZone: timezone })}:00`}
            </Typography>
            <WeatherIcon code={hour.code} />
            <Typography as="strong" variant="weatherTemperature">
              {Math.round(hour.temperature)}°
            </Typography>
            <Typography as="small" variant="weatherHour">
              {hour.rainProbability === 0
                ? "Dry"
                : `${Math.round(hour.rainProbability)}%`}
            </Typography>
          </div>
        ))}
      </div>
      <Typography
        as="p"
        variant="dialogLabel"
        className="dialog-label forecast-days-label"
      >
        Next seven days
      </Typography>
      <div className="weather-days">
        {data.days?.map((day) => (
          <div className="forecast-row" key={day.date}>
            <Typography as="span" variant="forecastLabel">
              {day.date === dayKeyAt(clock, timezone)
                ? "Today"
                : new Date(`${day.date}T12:00:00`).toLocaleDateString("en-GB", {
                    weekday: "short",
                    day: "numeric",
                    timeZone: timezone,
                  })}
            </Typography>
            <WeatherIcon code={day.code} />
            <Typography as="strong" variant="forecastTemperature">
              {Math.round(day.high)}°{" "}
              <Typography
                as="span"
                variant="forecastTemperature"
                className="muted"
              >
                {Math.round(day.low)}°
              </Typography>
            </Typography>
            <Typography
              as="span"
              variant="forecastRain"
              className="forecast-rain"
            >
              <CloudRain size={15} strokeWidth={1.7} aria-hidden="true" />{" "}
              {Math.round(day.rainProbability ?? 0)}%
            </Typography>
          </div>
        ))}
      </div>
      {data.advice && (
        <Typography as="p" variant="body" className="forecast-advice">
          {data.advice}
        </Typography>
      )}
      <Typography
        as="p"
        variant="dialogLabel"
        className="dialog-label forecast-legend"
      >
        High and low · Rain chance
      </Typography>
    </div>
  );
}
