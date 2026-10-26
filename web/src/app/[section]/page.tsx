import { Suspense } from "react";
import { notFound, redirect } from "next/navigation";
import { CalendarPage, ChoresPage, HouseholdPage, InventoryPage, MealsPage, MorePage, RecipesPage, ScratchpadPage, ShoppingPage, WeatherPage } from "@/components/member-pages";

async function SectionContent({ params }: { params: Promise<{ section: string }> }) {
  const { section } = await params;
  if (section === "calendar") return <CalendarPage />;
  if (section === "chores") return <ChoresPage />;
  if (section === "meals" || section === "food") return section === "food" ? redirect("/meals") : <MealsPage />;
  if (section === "recipes") return <RecipesPage />;
  if (section === "shopping") return <ShoppingPage />;
  if (section === "inventory") return <InventoryPage />;
  if (section === "scratchpad") return <ScratchpadPage />;
  if (section === "weather") return <WeatherPage />;
  if (section === "more") return <MorePage />;
  if (section === "household") return <HouseholdPage />;
  notFound();
}

export default function SectionPage({ params }: PageProps<"/[section]">) {
  return <Suspense fallback={<main className="page-wrap"><p role="status">Loading your shared home…</p></main>}><SectionContent params={params} /></Suspense>;
}
