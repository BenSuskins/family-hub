import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { Window } from "happy-dom";
import type { DisplayDashboardProps } from "@/features/display/components/display-dashboard";
import { DisplayDashboard } from "@/features/display/components/display-dashboard";
import { useDisplayShoppingActions } from "@/features/display/hooks/use-display-shopping-actions";
import type {
  DisplayShoppingItem,
  Snapshot,
} from "@/features/display/model/display-types";

const browser = new Window({ url: "http://localhost:3000" });
Object.assign(globalThis, {
  window: browser,
  document: browser.document,
  navigator: browser.navigator,
  HTMLElement: browser.HTMLElement,
  Element: browser.Element,
  IS_REACT_ACT_ENVIRONMENT: true,
});

const { cleanup, fireEvent, render, screen } =
  await import("@testing-library/react");
const originalFetch = globalThis.fetch;

afterEach(() => {
  cleanup();
  globalThis.fetch = originalFetch;
});

const clock = new Date("2026-10-27T17:30:00Z");

function makeSnapshot(overrides: Partial<Snapshot> = {}): Snapshot {
  return {
    schemaVersion: 2,
    displayId: "display-1",
    events: [],
    chores: [],
    meals: [],
    people: [],
    enabledSections: ["calendar", "meals", "shopping", "inventory"],
    timezone: "Europe/London",
    resourceUpdatedAt: {},
    availability: {},
    ...overrides,
  };
}

function renderDashboard(overrides: Partial<DisplayDashboardProps> = {}) {
  return render(
    <DisplayDashboard
      snapshot={makeSnapshot()}
      shoppingItems={[]}
      inventoryAreas={[]}
      clock={clock}
      loading={false}
      failures={[]}
      pendingChoreId=""
      onOpenApp={() => {}}
      onOpenEvent={() => {}}
      onSelectChore={() => {}}
      onCompleteAssignedChore={() => {}}
      onToggleShopping={() => {}}
      pendingShoppingId=""
      onRefresh={() => {}}
      {...overrides}
    />,
  );
}

test("shows rolling calendar and meal previews and hides cards without entries", () => {
  renderDashboard({
    snapshot: makeSnapshot({
      enabledSections: [
        "calendar",
        "meals",
        "chores",
        "departures",
        "bins",
        "trains",
        "shopping",
        "inventory",
      ],
      events: [
        { ID: "today", Title: "Library", StartTime: "2026-10-27T19:00:00Z" },
        { ID: "later", Title: "Swimming", StartTime: "2026-11-02T16:00:00Z" },
        {
          ID: "outside",
          Title: "Next week",
          StartTime: "2026-11-03T16:00:00Z",
        },
      ],
      meals: [
        { Date: "2026-10-27", MealType: "dinner", Name: "Tacos" },
        { Date: "2026-11-02", MealType: "dinner", Name: "Pasta" },
        { Date: "2026-11-03", MealType: "dinner", Name: "Curry" },
      ],
    }),
  });

  assert.ok(
    screen.getByRole("region", { name: "Calendar for the next seven days" }),
  );
  assert.ok(
    screen.getByRole("region", { name: "Meals for the next seven days" }),
  );
  assert.ok(screen.getByText("Swimming"));
  assert.ok(screen.getByText("Pasta"));
  assert.equal(screen.queryByText("Next week"), null);
  assert.equal(screen.queryByText("Curry"), null);
  assert.equal(screen.queryByRole("region", { name: "Shopping list" }), null);
  assert.equal(screen.queryByRole("region", { name: "Low inventory" }), null);
});

test("ranks available cards and displays no more than five", () => {
  renderDashboard({
    snapshot: makeSnapshot({
      enabledSections: [
        "calendar",
        "meals",
        "chores",
        "departures",
        "bins",
        "trains",
        "shopping",
        "inventory",
      ],
      events: [
        { ID: "today", Title: "Library", StartTime: "2026-10-27T19:00:00Z" },
        { ID: "later", Title: "Swimming", StartTime: "2026-11-02T16:00:00Z" },
      ],
      chores: [{ ID: "chore-1", Name: "Laundry", Status: "pending" }],
      meals: [{ Date: "2026-10-27", MealType: "dinner", Name: "Tacos" }],
      departures: {
        status: "available",
        items: [
          {
            eventID: "today",
            title: "Library",
            destination: "",
            startTime: "2026-10-27T18:30:00Z",
            leaveAt: "2026-10-27T23:19:00Z",
            drivingSeconds: 0,
          },
        ],
      },
      bins: {
        collections: [
          { date: "2026-10-27", type: "recycling", name: "Recycling" },
        ],
      },
      commute: {
        status: "available",
        boards: [
          {
            origin: { code: "KGX" },
            destination: { code: "CBG" },
            fetchedAt: "2026-10-27T17:30:00Z",
            trains: [
              {
                serviceId: "train-1",
                status: "on_time",
                scheduledDeparture: "18:00",
              },
            ],
          },
        ],
      },
    }),
    shoppingItems: [{ id: "milk", name: "Milk", purchased: false, version: 1 }],
    inventoryAreas: [
      {
        ID: "cupboard",
        Name: "Kitchen",
        Items: [
          {
            ID: "tea",
            Name: "Tea",
            TrackingMode: "count",
            Quantity: 2,
            Level: 0,
            LowAt: 2,
            Unit: "boxes",
          },
        ],
      },
    ],
  });

  assert.equal(
    document.querySelector(".display-grid")?.getAttribute("data-card-count"),
    "5",
  );
});

test("formats long departure countdowns in hours and scales bin collection content", () => {
  renderDashboard({
    snapshot: makeSnapshot({
      enabledSections: ["departures", "bins"],
      departures: {
        status: "available",
        items: [
          {
            eventID: "today",
            title: "Library",
            destination: "",
            startTime: "2026-10-28T00:00:00Z",
            leaveAt: "2026-10-27T23:19:00Z",
            drivingSeconds: 0,
          },
        ],
      },
      bins: {
        collections: [
          { date: "2026-10-28", type: "recycling", name: "Recycling" },
        ],
      },
    }),
  });

  assert.ok(screen.getByText("Leave in 5 hr 49 min"));
  assert.ok(screen.getByRole("region", { name: "Bin collections" }));
  assert.ok(screen.getByText("Recycling"));
});

test("uses configured low-stock thresholds and opens inventory; shopping rows are actionable", () => {
  const toggled: string[] = [];
  const opened: string[] = [];
  renderDashboard({
    snapshot: makeSnapshot(),
    shoppingItems: [{ id: "milk", name: "Milk", purchased: false, version: 4 }],
    inventoryAreas: [
      {
        ID: "cupboard",
        Name: "Kitchen",
        Items: [
          {
            ID: "tea",
            Name: "Tea",
            TrackingMode: "count",
            Quantity: 2,
            Level: 0,
            LowAt: 2,
            Unit: "boxes",
          },
          {
            ID: "rice",
            Name: "Rice",
            TrackingMode: "count",
            Quantity: 3,
            Level: 0,
            LowAt: 2,
            Unit: "bags",
          },
          {
            ID: "soap",
            Name: "Soap",
            TrackingMode: "level",
            Quantity: 0,
            Level: 25,
            LowAt: 25,
            Unit: "",
          },
        ],
      },
    ],
    onOpenApp: (app) => opened.push(app),
    onToggleShopping: (item) => toggled.push(item.id),
  });

  assert.ok(screen.getByText("Tea"));
  assert.ok(screen.getByText("Soap"));
  assert.equal(screen.queryByText("Rice"), null);
  fireEvent.click(
    screen.getByRole("button", { name: "Open Inventory for Tea" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Mark purchased: Milk" }));
  assert.deepEqual(opened, ["Inventory"]);
  assert.deepEqual(toggled, ["milk"]);
});

function ShoppingActionProbe({ reload }: { reload: () => Promise<void> }) {
  const { feedback, toggleShopping, undoShopping } = useDisplayShoppingActions(
    "display-token",
    reload,
  );
  const item: DisplayShoppingItem = {
    id: "milk",
    name: "Milk",
    purchased: false,
    version: 4,
  };
  return (
    <>
      <button onClick={() => toggleShopping(item)}>Mark purchased</button>
      {feedback && <p role="status">{feedback.message}</p>}
      {feedback?.undoItem && <button onClick={undoShopping}>Undo</button>}
    </>
  );
}

test("shopping card action can be undone with the returned item version", async () => {
  const requests: { purchased: boolean; version: number }[] = [];
  globalThis.fetch = async (_input, init) => {
    const body = JSON.parse(String(init?.body)) as {
      purchased: boolean;
      version: number;
    };
    requests.push(body);
    return Response.json({
      id: "milk",
      name: "Milk",
      purchased: body.purchased,
      version: body.version + 1,
    });
  };
  let reloadCount = 0;
  render(
    <ShoppingActionProbe
      reload={async () => {
        reloadCount += 1;
      }}
    />,
  );

  fireEvent.click(screen.getByRole("button", { name: "Mark purchased" }));
  await screen.findByText("Milk purchased");
  fireEvent.click(screen.getByRole("button", { name: "Undo" }));
  await screen.findByText("Milk returned to the list");

  assert.deepEqual(requests, [
    { purchased: true, version: 4 },
    { purchased: false, version: 5 },
  ]);
  assert.equal(reloadCount, 2);
  assert.equal(screen.queryByRole("button", { name: "Undo" }), null);
});
