import { notFound, redirect } from "next/navigation";
import { CalendarPage, ChoresPage, HouseholdPage, InventoryPage, MealsPage, MorePage, RecipesPage, ScratchpadPage, ShoppingPage, WeatherPage } from "@/components/member-pages";

export default async function SectionPage({ params }: PageProps<"/[section]">) {
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
