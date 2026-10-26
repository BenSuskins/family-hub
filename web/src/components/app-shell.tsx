"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { getToken } from "@/lib/api";

const navigation = [
  ["Today", "/", "◷"],
  ["Calendar", "/calendar", "▦"],
  ["Chores", "/chores", "✓"],
  ["Meals", "/meals", "♧"],
  ["Recipes", "/recipes", "⌁"],
  ["Shopping", "/shopping", "＋"],
  ["Inventory", "/inventory", "▤"],
  ["Scratchpad", "/scratchpad", "✎"],
  ["Household", "/household", "⌂"],
] as const;
const mobileNavigation = [...navigation.filter(([, href]) => ["/", "/calendar", "/chores"].includes(href)), ["More", "/more", "···"] as const];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [signedIn, setSignedIn] = useState(false);
  const [today, setToday] = useState("");
  useEffect(() => {
    setSignedIn(Boolean(getToken()));
    setToday(new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric" }).format(new Date()).toUpperCase());
  }, []);

  if (pathname === "/login" || pathname === "/auth/callback" || pathname === "/display") return <>{children}</>;

  return (
    <div className="app-frame">
      <aside className="sidebar">
        <Link className="brand" href="/"><span className="brand-mark">f</span><span>family hub<small>together, at home</small></span></Link>
        <p className="nav-label">YOUR HOME</p>
        <nav aria-label="Main navigation">
          {navigation.map(([label, href, icon]) => <Link key={href} href={href} className={`nav-link ${pathname === href ? "active" : ""}`}><span aria-hidden="true">{icon}</span>{label}</Link>)}
        </nav>
        <div className="sidebar-bottom"><div className="home-chip">Shared family home</div><Link className="display-link" href="/display">▣ &nbsp; Wall display</Link></div>
      </aside>
      <main className="main-content">
        <header className="topbar"><span className="eyebrow">{today || "FAMILY HUB"}</span><div className="topbar-actions">{!signedIn && <Link className="sign-in-top" href="/login">Sign in</Link>}<span className="connection">{signedIn ? "Signed in" : "Not connected"}</span>{signedIn && <button className="sign-out" onClick={() => { window.localStorage.removeItem("family-hub-token"); window.location.assign("/login"); }}>Sign out</button>}</div></header>
        {children}
      </main>
      <nav className="mobile-nav" aria-label="Main navigation">{mobileNavigation.map(([label, href, icon]) => <Link key={href} href={href} className={(pathname === href || (href === "/more" && !["/", "/calendar", "/chores"].includes(pathname))) ? "active" : ""}><span>{icon}</span>{label}</Link>)}</nav>
    </div>
  );
}
