"use client";

import { useCallback, useEffect, useState } from "react";
import { displayRequest } from "@/features/display/api/display-api";

type KioskResource<T> = {
  data: T | null;
  error: string;
  setError: (message: string) => void;
  reload: () => Promise<void>;
};

export function useKioskResource<T>(path: string, token: string): KioskResource<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState("");
  const reload = useCallback(async () => {
    setError("");
    try {
      setData(await displayRequest<T>(path, token));
    } catch {
      setError("This family information is unavailable right now.");
    }
  }, [path, token]);

  useEffect(() => { void reload(); }, [reload]);

  return { data, error, setError, reload };
}
