"use client";

import { useState } from "react";
import { apiBase } from "@/features/display/api/display-api";
import type { DisplayChore, DisplayPerson } from "@/features/display/model/display-types";

export function ChoreCompletionDialog({ chore, people, token, onComplete }: { chore: DisplayChore; people: DisplayPerson[]; token: string; onComplete: () => void }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  async function complete(userID: string) {
    setPending(true);
    setError("");
    try {
      const response = await fetch(`${apiBase}/api/display/chores/${encodeURIComponent(chore.ID)}/complete`, { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" }, body: JSON.stringify({ userID }) });
      if (!response.ok) throw new Error(response.status === 409 ? "This chore was already completed." : "The chore could not be completed. Try again.");
      onComplete();
    } catch (reason) { setError(reason instanceof Error ? reason.message : "The chore could not be completed. Try again."); }
    finally { setPending(false); }
  }
  return <section className="person-picker"><h3>Who completed {chore.Name}?</h3>{people.map(person => <button className="person-button" key={person.id} disabled={pending} onClick={() => void complete(person.id)}><span className="person-avatar">{person.name.slice(0, 1).toUpperCase()}</span>{person.name}</button>)}{error && <p role="alert">{error}</p>}</section>;
}


