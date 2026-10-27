"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { displayRequest } from "@/features/display/api/display-api";

type KioskResource<T> = {
  data: T | null;
  error: string;
  setError: (message: string) => void;
  reload: () => Promise<void>;
};

type ResourceResult<T> = { data: T; error: "" } | { data: null; error: string };

async function requestResource<T>(path: string, token: string): Promise<ResourceResult<T>> {
  try {
    return { data: await displayRequest<T>(path, token), error: "" };
  } catch {
    return { data: null, error: "This family information is unavailable right now." };
  }
}

export function useKioskResource<T>(path: string, token: string): KioskResource<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState("");
  const requestGeneration = useRef(0);
  const reload = useCallback(async () => {
    const generation = ++requestGeneration.current;
    const result = await requestResource<T>(path, token);
    if (generation !== requestGeneration.current) return;
    if (!result.error) setData(result.data);
    setError(result.error);
  }, [path, token]);

  useEffect(() => {
    const generation = ++requestGeneration.current;
    void requestResource<T>(path, token).then(result => {
      if (generation !== requestGeneration.current) return;
      if (!result.error) setData(result.data);
      setError(result.error);
    });
    return () => { requestGeneration.current += 1; };
  }, [path, token]);

  return { data, error, setError, reload };
}
