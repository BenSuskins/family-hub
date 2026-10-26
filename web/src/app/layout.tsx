import type { Metadata } from "next";
import { Suspense } from "react";
import { AppShell } from "@/components/app-shell";
import "./globals.css";

export const metadata: Metadata = {
  title: "Family Hub — Home, together",
  description: "A calm shared space for the everyday life of your home.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return <html lang="en"><body><Suspense fallback={<main className="page-wrap" aria-busy="true"/>}><AppShell>{children}</AppShell></Suspense></body></html>;
}
