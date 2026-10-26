import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Family Hub — Home, together",
  description: "A calm shared space for the everyday life of your home.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return <html lang="en"><body>{children}</body></html>;
}
