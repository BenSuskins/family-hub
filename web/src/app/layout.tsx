import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Family household",
  description: "A shared Family Hub display for your household.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
