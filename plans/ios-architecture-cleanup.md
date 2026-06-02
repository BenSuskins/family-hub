# iOS Architecture Cleanup + Contract Tests

Goal: remove accreted duplication and unify load/error UX in the iOS app, and
introduce a contract-test seam so the `DemoAPIClient` fake is verified against
the same behavioural contract as the real client.

## Decisions
- **Contract tests**: parameterised suite run against `DemoAPIClient` now; the
  live `APIClient` (URLProtocol stub server) is left as a stubbed subclass +
  TODO so the seam exists without the extra infra yet.
- **#9 (ShareAPIError/APIError dedup)**: deferred. The share extension cannot
  import the app module; unifying needs a shared framework target. Documented
  in-code instead of risky `.pbxproj` surgery.

## Work items

| # | Change | Files |
|---|--------|-------|
| 1 | Shared cached POSIX date formatters + `APIDate.parse` | new `Utilities/DateUtilities.swift`; APIClient, CalendarVM, MealsView, HomeView, ChoreFormView, Chore, DemoAPIClient |
| 2 | `StateContentView` ViewState renderer | new `Components/StateContentView.swift`; HomeView, MealsView, SearchView, RecipesView |
| 3 | `MutableListViewModel` CRUD helpers; collapse Home `load`/`loadSilently` | new `Features/Shared/MutableListViewModel.swift`; ChoresVM, RecipesVM, HomeVM |
| 4 | `Chore.with(status:)` / `.completed` | Chore.swift; ChoresVM, DemoAPIClient |
| 5 | `Sequence<User>.keyedByID` | User.swift; HomeVM, ChoresVM, CalendarVM |
| 6 | Reusable `SegmentedControl` (optional badge) | new `Components/SegmentedControl.swift`; ChoresListView |
| 7 | `SectionHeaderLabel` gains `color` | AppTheme.swift; ChoresListView |
| 8 | Document deliberate multi-cache strategy | APIClient.swift |
| 9 | Document ShareAPIError duplication + deferral | ShareAPIError.swift |
| — | Contract tests | new `FamilyHubTests/Contract/APIClientContract.swift` |

## Notes / non-goals
- Calendar and Chores keep bespoke layouts (cached stale data / always-visible
  segmented control) — they don't fit `StateContentView`'s all-or-nothing model.
- MealsView mode toggle and Calendar's native segmented `Picker` are distinct
  visual styles; not folded into the new `SegmentedControl`.
- `SearchView` appears unused by `ContentView`; left as-is (out of scope).

## Verification
`xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 17'`
