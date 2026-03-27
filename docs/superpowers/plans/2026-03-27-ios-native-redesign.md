# iOS Native Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the iOS app from a custom-themed web-like design to a fully native iOS 26 app with 3 tabs, system colors, and Liquid Glass chrome.

**Architecture:** Strip custom `Theme.swift` design system entirely. Replace all custom components with stock SwiftUI. Restructure from 5 tabs (Home/Chores/Meals/Recipes/Calendar) to 3 tabs (Home/Meals/Calendar). Chores become a drill-in from Home. Recipes merge into Meals tab.

**Tech Stack:** SwiftUI (iOS 26+), `@Observable`, `NavigationStack`, native `List`/`Section`/`Picker`

**Build order:** Each task leaves the project compilable. Views are rewritten first (removing Theme references). Theme.swift and custom components are deleted last, once nothing references them.

---

### Task 1: Extract ChoreBadge from StatusBadge into Chore.swift

The `Chore.badgeVariant` extension currently returns `StatusBadge.Variant`, coupling the model to a view component we're deleting. Extract a standalone `ChoreBadge` enum into `Chore.swift` so the model is self-contained. Update all call sites from `.badgeVariant` to `.badge`. Keep `StatusBadge.swift` alive for now — it will be deleted in the final cleanup task.

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Models/Chore.swift`
- Modify: `ios/FamilyHub/FamilyHub/Components/StatusBadge.swift`
- Create: `ios/FamilyHub/FamilyHubTests/Models/ChoreBadgeTests.swift`

- [ ] **Step 1: Write ChoreBadge tests**

Create `ios/FamilyHub/FamilyHubTests/Models/ChoreBadgeTests.swift`:

```swift
import XCTest
@testable import FamilyHub

final class ChoreBadgeTests: XCTestCase {
    func testOverdueLabel() {
        XCTAssertEqual(ChoreBadge.overdue.label, "Overdue")
    }

    func testDueTodayLabel() {
        XCTAssertEqual(ChoreBadge.dueToday.label, "Today")
    }

    func testDueSoonLabel() {
        XCTAssertEqual(ChoreBadge.dueSoon.label, "Due Soon")
    }

    func testChoreOverdueStatusMapsToOverdueBadge() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .overdue,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertEqual(chore.badge, .overdue)
    }

    func testChoreCompletedStatusHasNoBadge() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .completed,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertNil(chore.badge)
    }

    func testChorePendingDueTodayMapsToDueToday() {
        let todayString = ISO8601DateFormatter().string(from: Date())
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: todayString, assignedToUserID: nil)
        XCTAssertEqual(chore.badge, .dueToday)
    }

    func testChorePendingFutureDateMapsToDueSoon() {
        let futureDate = Calendar.current.date(byAdding: .day, value: 3, to: Date())!
        let futureString = ISO8601DateFormatter().string(from: futureDate)
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: futureString, assignedToUserID: nil)
        XCTAssertEqual(chore.badge, .dueSoon)
    }
}
```

Add this file to the `FamilyHubTests` target in `project.pbxproj`.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/ChoreBadgeTests 2>&1 | tail -10
```

Expected: FAIL — `ChoreBadge` does not exist, `chore.badge` does not exist.

- [ ] **Step 3: Add ChoreBadge enum and Chore.badge to Chore.swift**

Add `import SwiftUI` at the top of `Chore.swift`, then append after the existing `formattedDueDate` extension:

```swift
enum ChoreBadge: Equatable {
    case overdue
    case dueToday
    case dueSoon

    var label: String {
        switch self {
        case .overdue:  return "Overdue"
        case .dueToday: return "Today"
        case .dueSoon:  return "Due Soon"
        }
    }

    var color: Color {
        switch self {
        case .overdue:            return .red
        case .dueToday, .dueSoon: return .orange
        }
    }
}

extension Chore {
    var badge: ChoreBadge? {
        switch status {
        case .overdue:   return .overdue
        case .completed: return nil
        case .pending:
            guard let dueDate else { return .dueSoon }
            let date = ISO8601DateFormatter().date(from: dueDate)
                ?? parseShortDate(dueDate)
            guard let date else { return .dueSoon }
            return Calendar.current.isDateInToday(date) ? .dueToday : .dueSoon
        }
    }

    private func parseShortDate(_ string: String) -> Date? {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        return formatter.date(from: String(string.prefix(10)))
    }
}
```

Keep the old `badgeVariant` extension in `StatusBadge.swift` for now — existing views still reference it.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/ChoreBadgeTests 2>&1 | tail -10
```

Expected: All 7 tests pass.

- [ ] **Step 5: Run full test suite to verify nothing is broken**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -20
```

Expected: All tests pass. Old `StatusBadgeTests` still pass too (old code is still there).

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor(ios): extract ChoreBadge enum into Chore.swift"
```

---

### Task 2: Rewrite ContentView to 3-tab structure

Replace the 5-tab `TabView` with 3 tabs: Home, Meals, Calendar. Uses iOS 18+ `Tab` API. The `HomeView` doesn't exist yet, so temporarily alias `DashboardView` as the Home tab content — Task 3 will create the real `HomeView`.

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/ContentView.swift`

- [ ] **Step 1: Rewrite ContentView.swift**

```swift
import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol

    var body: some View {
        TabView {
            Tab("Home", systemImage: "house") {
                DashboardView(apiClient: apiClient)
            }
            Tab("Meals", systemImage: "fork.knife") {
                MealsView(apiClient: apiClient)
            }
            Tab("Calendar", systemImage: "calendar") {
                CalendarView(apiClient: apiClient)
            }
        }
    }
}
```

This temporarily uses `DashboardView` for the Home tab. `RecipesView` and `ChoresView` are no longer tabs but still exist in the codebase — they'll be rewritten in later tasks.

- [ ] **Step 2: Build to verify compilation**

```bash
cd ios/FamilyHub && xcodebuild build -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -5
```

Expected: BUILD SUCCEEDED.

- [ ] **Step 3: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/ContentView.swift && git commit -m "refactor(ios): rewrite ContentView to 3-tab structure"
```

---

### Task 3: Create HomeView and HomeViewModel

Replace `DashboardView`/`DashboardViewModel` with `HomeView`/`HomeViewModel`. HomeView uses a native `List` with sections for chores (with swipe-to-complete), today's meals, and weekly stats. Includes a "See All Chores" `NavigationLink`.

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Features/Home/HomeView.swift`
- Create: `ios/FamilyHub/FamilyHub/Features/Home/HomeViewModel.swift`
- Create: `ios/FamilyHub/FamilyHubTests/Home/HomeViewModelTests.swift`
- Modify: `ios/FamilyHub/FamilyHub/Features/ContentView.swift` (swap DashboardView → HomeView)

- [ ] **Step 1: Write HomeViewModel tests**

Create `ios/FamilyHub/FamilyHubTests/Home/HomeViewModelTests.swift`:

```swift
import XCTest
@testable import FamilyHub

@MainActor
final class HomeViewModelTests: XCTestCase {
    func testLoadSuccess() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(DashboardStats(
            choresDueToday: 2,
            choresOverdue: 1,
            choresDueTodayList: [],
            choresOverdueList: []
        ))
        let viewModel = HomeViewModel(apiClient: fake)

        await viewModel.load()

        guard case .loaded(let stats) = viewModel.state else {
            XCTFail("expected loaded state, got \(viewModel.state)")
            return
        }
        XCTAssertEqual(stats.choresDueToday, 2)
        XCTAssertEqual(stats.choresOverdue, 1)
    }

    func testLoadFailure() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .failure(APIError.server(500))
        let viewModel = HomeViewModel(apiClient: fake)

        await viewModel.load()

        guard case .failed(let error) = viewModel.state else {
            XCTFail("expected failed state")
            return
        }
        if case .server(let code) = error {
            XCTAssertEqual(code, 500)
        } else {
            XCTFail("expected server error, got \(error)")
        }
    }

    func testInitialStateIsIdle() {
        let viewModel = HomeViewModel(apiClient: FakeAPIClient())
        guard case .idle = viewModel.state else {
            XCTFail("expected idle initial state")
            return
        }
    }

    func testLoadFetchesUsers() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(
            DashboardStats(choresDueToday: 1, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [])
        )
        fake.usersResult = .success([User(id: "u1", name: "Ben Suskins", email: "", avatarURL: "")])
        let viewModel = HomeViewModel(apiClient: fake)

        await viewModel.load()

        XCTAssertEqual(viewModel.users["u1"]?.name, "Ben Suskins")
    }

    func testCompleteChoreSuccess() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(
            DashboardStats(choresDueToday: 1, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [])
        )
        fake.completeChoreResult = .success(())
        let viewModel = HomeViewModel(apiClient: fake)
        await viewModel.load()

        let result = await viewModel.completeChore(id: "c1")

        XCTAssertTrue(result)
    }

    func testCompleteChoreFailure() async {
        let fake = FakeAPIClient()
        fake.dashboardResult = .success(
            DashboardStats(choresDueToday: 0, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [])
        )
        fake.completeChoreResult = .failure(APIError.server(500))
        let viewModel = HomeViewModel(apiClient: fake)
        await viewModel.load()

        let result = await viewModel.completeChore(id: "c1")

        XCTAssertFalse(result)
    }
}
```

Add this file to the `FamilyHubTests` target in `project.pbxproj`.

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/HomeViewModelTests 2>&1 | tail -10
```

Expected: FAIL — `HomeViewModel` does not exist.

- [ ] **Step 3: Create HomeViewModel**

Create `ios/FamilyHub/FamilyHub/Features/Home/HomeViewModel.swift`:

```swift
import Foundation
import Observation

@Observable
@MainActor
final class HomeViewModel {
    var state: ViewState<DashboardStats> = .idle
    var users: [String: User] = [:]

    let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        state = .loading
        async let statsTask = apiClient.fetchDashboardStats()
        async let usersTask = apiClient.fetchUsers()
        do {
            let (stats, userList) = try await (statsTask, usersTask)
            users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
            state = .loaded(stats)
        } catch let error as APIError {
            state = .failed(error)
        } catch {
            state = .failed(.network(error))
        }
    }

    func completeChore(id: String) async -> Bool {
        do {
            try await apiClient.completeChore(id: id)
            await load()
            return true
        } catch {
            return false
        }
    }
}
```

Add this file to the `FamilyHub` target in `project.pbxproj`.

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/HomeViewModelTests 2>&1 | tail -10
```

Expected: All 6 tests pass.

- [ ] **Step 5: Create HomeView**

Create `ios/FamilyHub/FamilyHub/Features/Home/HomeView.swift`:

```swift
import SwiftUI

struct HomeView: View {
    @State private var viewModel: HomeViewModel
    @State private var showProfile = false
    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: HomeViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            List {
                switch viewModel.state {
                case .idle, .loading:
                    Section {
                        ProgressView()
                            .frame(maxWidth: .infinity)
                    }
                case .failed(let error):
                    Section {
                        Text(error.localizedDescription)
                            .foregroundStyle(.red)
                    }
                case .loaded(let stats):
                    choreSection(stats)
                    mealsSection
                    statsSection(stats)
                }
            }
            .listStyle(.insetGrouped)
            .refreshable { await viewModel.load() }
            .navigationTitle("Home")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showProfile = true
                    } label: {
                        UserAvatar(user: nil, size: 32)
                    }
                }
            }
            .sheet(isPresented: $showProfile) {
                ProfileView()
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Chores Section

    @ViewBuilder
    private func choreSection(_ stats: DashboardStats) -> some View {
        let allDue = stats.choresOverdueList + stats.choresDueTodayList
        Section {
            if allDue.isEmpty {
                Label("All caught up!", systemImage: "checkmark.circle")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(allDue) { chore in
                    choreRow(chore)
                        .swipeActions(edge: .leading) {
                            Button {
                                Task { await viewModel.completeChore(id: chore.id) }
                            } label: {
                                Label("Done", systemImage: "checkmark")
                            }
                            .tint(.green)
                        }
                }
            }
            NavigationLink {
                ChoresListView(apiClient: apiClient)
            } label: {
                Text("See All Chores")
            }
        } header: {
            Text("Chores")
        }
    }

    private func choreRow(_ chore: Chore) -> some View {
        HStack(spacing: 10) {
            UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.body)
                HStack(spacing: 6) {
                    if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                        Text(user.name)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    if let badge = chore.badge {
                        Text(badge.label)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(badge.color)
                    }
                    if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.caption2)
                            .foregroundStyle(chore.status == .overdue ? .red : .orange)
                    }
                }
            }
        }
    }

    // MARK: - Meals Section

    private var mealsSection: some View {
        Section {
            mealRow(label: "Lunch", name: nil)
            mealRow(label: "Dinner", name: nil)
        } header: {
            Text("Today's Meals")
        }
    }

    private func mealRow(label: String, name: String?) -> some View {
        HStack {
            Text(label)
                .foregroundStyle(.secondary)
            Spacer()
            Text(name ?? "—")
                .foregroundStyle(name != nil ? .primary : .tertiary)
        }
    }

    // MARK: - Stats Section

    private func statsSection(_ stats: DashboardStats) -> some View {
        Section {
            HStack(spacing: 12) {
                statItem(value: stats.choresDueToday + stats.choresOverdue, label: "Chores due")
                Divider()
                statItem(value: 0, label: "Meals planned")
                Divider()
                statItem(value: 0, label: "Events")
            }
            .padding(.vertical, 4)
        } header: {
            Text("This Week")
        }
    }

    private func statItem(value: Int, label: String) -> some View {
        VStack(spacing: 2) {
            Text("\(value)")
                .font(.title2.bold())
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }
}
```

Note: This references `ChoresListView` which doesn't exist yet. Create a temporary stub file `ios/FamilyHub/FamilyHub/Features/Chores/ChoresListView.swift` so the project compiles:

```swift
import SwiftUI

struct ChoresListView: View {
    init(apiClient: any APIClientProtocol) {}
    var body: some View {
        Text("Chores list — placeholder")
    }
}
```

This stub will be replaced with the real implementation in Task 4.

Add both new files to the `FamilyHub` target in `project.pbxproj`.

- [ ] **Step 6: Update ContentView to use HomeView**

```swift
import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol

    var body: some View {
        TabView {
            Tab("Home", systemImage: "house") {
                HomeView(apiClient: apiClient)
            }
            Tab("Meals", systemImage: "fork.knife") {
                MealsView(apiClient: apiClient)
            }
            Tab("Calendar", systemImage: "calendar") {
                CalendarView(apiClient: apiClient)
            }
        }
    }
}
```

- [ ] **Step 7: Build and run all tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -20
```

Expected: BUILD SUCCEEDED, all tests pass. Old `DashboardView` and `DashboardViewModel` still exist and compile — they're just not referenced by ContentView anymore.

- [ ] **Step 8: Commit**

```bash
git add -A && git commit -m "feat(ios): create HomeView with native List, swipe-to-complete chores"
```

---

### Task 4: Rewrite ChoresListView and ChoreDetailView

Replace the stub `ChoresListView` with the full implementation. Uses native segmented `Picker`, swipe actions, and system colors. Remove all `Theme.x` references from both files.

**Files:**
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Chores/ChoresListView.swift` (replace stub)
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Chores/ChoreDetailView.swift`

- [ ] **Step 1: Rewrite ChoresListView**

Replace the stub in `ios/FamilyHub/FamilyHub/Features/Chores/ChoresListView.swift`:

```swift
import SwiftUI

struct ChoresListView: View {
    @State private var viewModel: ChoresViewModel
    @State private var selectedTab: Tab = .pending

    enum Tab: String, CaseIterable {
        case pending = "Pending"
        case completed = "Completed"
    }

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            Picker("Filter", selection: $selectedTab) {
                ForEach(Tab.allCases, id: \.self) { tab in
                    Text(tab.rawValue).tag(tab)
                }
            }
            .pickerStyle(.segmented)
            .listRowBackground(Color.clear)
            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))

            if case .failed(let error) = viewModel.state {
                Section {
                    Text(error.localizedDescription)
                        .foregroundStyle(.red)
                }
            }

            if selectedTab == .pending {
                pendingContent
            } else {
                completedContent
            }
        }
        .listStyle(.insetGrouped)
        .refreshable { await viewModel.load() }
        .navigationTitle("Chores")
        .task { await viewModel.load() }
    }

    // MARK: - Pending

    @ViewBuilder
    private var pendingContent: some View {
        if !viewModel.overdueChores.isEmpty {
            Section("Overdue") {
                ForEach(viewModel.overdueChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            }
        }

        if !viewModel.dueSoonChores.isEmpty {
            Section("Due Soon") {
                ForEach(viewModel.dueSoonChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            }
        }

        if viewModel.overdueChores.isEmpty && viewModel.dueSoonChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("All done!", systemImage: "checkmark.circle.fill")
            }
        }
    }

    // MARK: - Completed

    @ViewBuilder
    private var completedContent: some View {
        if viewModel.completedChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("No completed chores", systemImage: "clock")
            }
        } else {
            Section {
                ForEach(viewModel.completedChores) { chore in
                    choreRow(chore, isCompleted: true)
                }
            }
        }
    }

    // MARK: - Row

    private func choreRow(_ chore: Chore, isCompleted: Bool) -> some View {
        NavigationLink {
            ChoreDetailView(chore: chore, viewModel: viewModel)
        } label: {
            HStack(spacing: 10) {
                if isCompleted {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(.green)
                        .font(.title2)
                } else {
                    UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text(chore.name)
                        .font(.body)
                    HStack(spacing: 6) {
                        if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                            Text(user.name)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.caption2)
                                .foregroundStyle(isCompleted ? .tertiary :
                                    (chore.status == .overdue ? .red : .orange))
                        }
                    }
                }
            }
        }
        .swipeActions(edge: .leading) {
            if !isCompleted {
                Button {
                    Task { await viewModel.complete(choreID: chore.id) }
                } label: {
                    Label("Done", systemImage: "checkmark")
                }
                .tint(.green)
            }
        }
    }
}
```

- [ ] **Step 2: Rewrite ChoreDetailView**

Replace `ios/FamilyHub/FamilyHub/Features/Chores/ChoreDetailView.swift`:

```swift
import SwiftUI

struct ChoreDetailView: View {
    let chore: Chore
    let viewModel: ChoresViewModel

    @Environment(\.dismiss) private var dismiss
    @State private var isCompleting = false
    @State private var completionError: String?

    var body: some View {
        List {
            Section {
                HStack(spacing: 12) {
                    UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                    VStack(alignment: .leading, spacing: 2) {
                        Text(viewModel.users[chore.assignedToUserID ?? ""]?.name ?? "Unassigned")
                            .font(.body.weight(.medium))
                        if let badge = chore.badge {
                            Text(badge.label)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(badge.color)
                        }
                    }
                    Spacer()
                    if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.subheadline)
                            .foregroundStyle(chore.status == .overdue ? .red : .secondary)
                    }
                }
            }

            if !chore.description.isEmpty {
                Section("Description") {
                    Text(chore.description)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }

            if chore.status != .completed {
                Section {
                    Button {
                        Task {
                            isCompleting = true
                            completionError = nil
                            let success = await viewModel.complete(choreID: chore.id)
                            isCompleting = false
                            if success { dismiss() } else { completionError = viewModel.errorMessage }
                        }
                    } label: {
                        HStack {
                            Spacer()
                            if isCompleting {
                                ProgressView()
                            } else {
                                Label("Mark Complete", systemImage: "checkmark")
                            }
                            Spacer()
                        }
                    }
                    .tint(.green)
                    .disabled(isCompleting)

                    if let completionError {
                        Text(completionError)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle(chore.name)
        .navigationBarTitleDisplayMode(.inline)
    }
}
```

- [ ] **Step 3: Delete old ChoresView.swift**

```bash
rm ios/FamilyHub/FamilyHub/Features/Chores/ChoresView.swift
```

Remove the `ChoresView.swift` reference from `project.pbxproj`.

- [ ] **Step 4: Run chores tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/ChoresViewModelTests 2>&1 | tail -10
```

Expected: All ChoresViewModel tests pass (view model unchanged).

- [ ] **Step 5: Build to verify compilation**

```bash
cd ios/FamilyHub && xcodebuild build -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -5
```

Expected: BUILD SUCCEEDED.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor(ios): rewrite ChoresListView and ChoreDetailView with native styling"
```

---

### Task 5: Rewrite MealsView with scope bar and merge Recipes

Add a segmented `Picker` to toggle between Plan and Recipes views. Move recipe files into the Meals directory. Rewrite `RecipeDetailView` with system colors.

**Files:**
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Meals/MealsView.swift`
- Move: `ios/FamilyHub/FamilyHub/Features/Recipes/RecipesView.swift` → `ios/FamilyHub/FamilyHub/Features/Meals/RecipesView.swift`
- Move: `ios/FamilyHub/FamilyHub/Features/Recipes/RecipesViewModel.swift` → `ios/FamilyHub/FamilyHub/Features/Meals/RecipesViewModel.swift`
- Move: `ios/FamilyHub/FamilyHub/Features/Recipes/RecipeDetailView.swift` → `ios/FamilyHub/FamilyHub/Features/Meals/RecipeDetailView.swift`
- Delete: `ios/FamilyHub/FamilyHub/Features/Recipes/` (directory after moves)

- [ ] **Step 1: Move recipe files into Meals directory**

```bash
mv ios/FamilyHub/FamilyHub/Features/Recipes/RecipesView.swift ios/FamilyHub/FamilyHub/Features/Meals/RecipesView.swift
mv ios/FamilyHub/FamilyHub/Features/Recipes/RecipesViewModel.swift ios/FamilyHub/FamilyHub/Features/Meals/RecipesViewModel.swift
mv ios/FamilyHub/FamilyHub/Features/Recipes/RecipeDetailView.swift ios/FamilyHub/FamilyHub/Features/Meals/RecipeDetailView.swift
rmdir ios/FamilyHub/FamilyHub/Features/Recipes
```

Update `project.pbxproj` to reflect the new file paths.

- [ ] **Step 2: Rewrite MealsView with scope bar**

Replace `ios/FamilyHub/FamilyHub/Features/Meals/MealsView.swift`:

```swift
import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel
    @State private var recipesViewModel: RecipesViewModel
    @State private var selectedScope: Scope = .plan
    private let apiClient: any APIClientProtocol

    enum Scope: String, CaseIterable {
        case plan = "Plan"
        case recipes = "Recipes"
    }

    private let mealTypes = ["breakfast", "lunch", "dinner"]

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "EEE d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let dateKeyFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "yyyy-MM-dd"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()
    private static let weekTitleFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d MMM"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
        _recipesViewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("View", selection: $selectedScope) {
                    ForEach(Scope.allCases, id: \.self) { scope in
                        Text(scope.rawValue).tag(scope)
                    }
                }
                .pickerStyle(.segmented)
                .padding(.horizontal)
                .padding(.vertical, 8)

                if selectedScope == .plan {
                    planView
                } else {
                    recipesView
                }
            }
            .navigationTitle("Meals")
            .toolbar {
                if selectedScope == .plan {
                    ToolbarItem(placement: .navigationBarLeading) {
                        Button { viewModel.previousWeek() } label: {
                            Image(systemName: "chevron.left")
                        }
                    }
                    ToolbarItem(placement: .principal) {
                        Text(weekTitle)
                            .font(.headline)
                    }
                    ToolbarItem(placement: .navigationBarTrailing) {
                        Button { viewModel.nextWeek() } label: {
                            Image(systemName: "chevron.right")
                        }
                    }
                }
            }
        }
        .task {
            await viewModel.load()
            await recipesViewModel.load()
        }
    }

    private var weekTitle: String {
        let start = viewModel.currentWeek
        let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
        return "\(Self.weekTitleFormatter.string(from: start)) – \(Self.weekTitleFormatter.string(from: end))"
    }

    // MARK: - Plan View

    @ViewBuilder
    private var planView: some View {
        switch viewModel.state {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let error):
            ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
        case .loaded(let meals):
            List {
                ForEach(0..<7, id: \.self) { offset in
                    let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                    let dateKey = Self.dateKeyFormatter.string(from: date)
                    Section(Self.dayFormatter.string(from: date)) {
                        ForEach(mealTypes, id: \.self) { mealType in
                            let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                            HStack {
                                Text(mealType.capitalized)
                                    .foregroundStyle(.secondary)
                                Spacer()
                                Text(plan?.name ?? "—")
                                    .foregroundStyle(plan != nil ? .primary : .tertiary)
                            }
                        }
                    }
                }
            }
            .listStyle(.insetGrouped)
            .refreshable { await viewModel.load() }
        }
    }

    // MARK: - Recipes View

    @ViewBuilder
    private var recipesView: some View {
        switch recipesViewModel.state {
        case .idle, .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .failed(let error):
            ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
        case .loaded:
            ScrollView {
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                    ForEach(recipesViewModel.filteredRecipes) { recipe in
                        NavigationLink {
                            RecipeDetailView(recipe: recipe, apiClient: apiClient)
                        } label: {
                            RecipeCard(recipe: recipe)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal)
            }
            .refreshable { await recipesViewModel.load() }
            .searchable(text: $recipesViewModel.searchQuery, prompt: "Search recipes")
        }
    }
}

private struct RecipeCard: View {
    let recipe: Recipe

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            RoundedRectangle(cornerRadius: 8)
                .fill(Color(.tertiarySystemFill))
                .aspectRatio(4/3, contentMode: .fit)
                .overlay {
                    Image(systemName: "fork.knife")
                        .foregroundStyle(.tertiary)
                        .font(.title2)
                }
            VStack(alignment: .leading, spacing: 4) {
                Text(recipe.title)
                    .font(.subheadline.weight(.semibold))
                    .lineLimit(2)
                HStack(spacing: 8) {
                    if let prep = recipe.prepTime {
                        Label("\(prep) prep", systemImage: "clock")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                    if let servings = recipe.servings {
                        Label("\(servings)", systemImage: "person.2")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .padding(.horizontal, 6)
            .padding(.bottom, 8)
        }
        .background(Color(.secondarySystemGroupedBackground))
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
```

- [ ] **Step 3: Rewrite RecipeDetailView with system colors**

Replace `ios/FamilyHub/FamilyHub/Features/Meals/RecipeDetailView.swift`:

```swift
import SwiftUI
import UIKit

struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol

    @State private var cookModeActive = false
    @State private var fullRecipe: Recipe?
    @State private var isLoading = true
    @State private var fetchError = false

    var body: some View {
        Group {
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if fetchError && fullRecipe == nil {
                ContentUnavailableView("Failed to load", systemImage: "exclamationmark.triangle")
            } else {
                recipeContent(fullRecipe ?? recipe)
            }
        }
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    cookModeActive.toggle()
                } label: {
                    Label(cookModeActive ? "Exit Cook Mode" : "Cook Mode",
                          systemImage: cookModeActive ? "flame.fill" : "flame")
                        .foregroundStyle(cookModeActive ? .orange : .accentColor)
                }
            }
        }
        .onDisappear {
            UIApplication.shared.isIdleTimerDisabled = false
        }
        .onChange(of: cookModeActive) { _, active in
            UIApplication.shared.isIdleTimerDisabled = active
        }
        .task {
            do {
                fullRecipe = try await apiClient.fetchRecipe(id: recipe.id)
            } catch {
                fetchError = true
            }
            isLoading = false
        }
    }

    private func recipeContent(_ r: Recipe) -> some View {
        List {
            Section {
                HStack(spacing: 16) {
                    if let prep = r.prepTime {
                        metaStat(label: "Prep", value: prep)
                    }
                    if let cook = r.cookTime {
                        metaStat(label: "Cook", value: cook)
                    }
                    if let servings = r.servings {
                        metaStat(label: "Serves", value: "\(servings)")
                    }
                }
            }

            if let ingredients = r.ingredients, !ingredients.isEmpty {
                ForEach(ingredients, id: \.name) { group in
                    Section(group.name.isEmpty ? "Ingredients" : group.name) {
                        ForEach(group.items, id: \.self) { item in
                            Text(item)
                                .font(.subheadline)
                        }
                    }
                }
            }

            if let steps = r.steps, !steps.isEmpty {
                Section("Steps") {
                    ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.subheadline.bold())
                                .foregroundStyle(.accent)
                                .frame(width: 22, alignment: .trailing)
                            Text(step)
                                .font(.subheadline)
                        }
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
    }

    private func metaStat(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }
}
```

- [ ] **Step 4: Delete old RecipesView.swift from Recipes directory (if leftover)**

Verify no files remain in `ios/FamilyHub/FamilyHub/Features/Recipes/`:

```bash
ls ios/FamilyHub/FamilyHub/Features/Recipes/ 2>&1
```

Expected: "No such file or directory" (already removed in step 1).

- [ ] **Step 5: Run meals and recipes tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/MealsViewModelTests -only-testing:FamilyHubTests/RecipesViewModelTests 2>&1 | tail -10
```

Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat(ios): rewrite MealsView with scope bar, merge Recipes into Meals tab"
```

---

### Task 6: Rewrite CalendarView with native patterns

Strip Theme references, use system colors, add color-coded agenda sidebar indicators, add today highlight.

**Files:**
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Calendar/CalendarView.swift`

- [ ] **Step 1: Rewrite CalendarView**

Replace `ios/FamilyHub/FamilyHub/Features/Calendar/CalendarView.swift`:

```swift
import SwiftUI

struct CalendarView: View {
    @State private var viewModel: CalendarViewModel
    private let columns = Array(repeating: GridItem(.flexible()), count: 7)
    private let weekdaySymbols = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]

    private static let monthFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "MMMM yyyy"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: CalendarViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 0) {
                    if case .failed(let error) = viewModel.state {
                        Text(error.localizedDescription)
                            .font(.caption)
                            .foregroundStyle(.red)
                            .padding()
                    }
                    calendarGrid
                        .padding(.horizontal)
                    Divider()
                    agendaSection
                }
            }
            .refreshable { await viewModel.load() }
            .navigationTitle(Self.monthFormatter.string(from: viewModel.currentMonth))
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button { viewModel.previousMonth() } label: {
                        Image(systemName: "chevron.left")
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { viewModel.nextMonth() } label: {
                        Image(systemName: "chevron.right")
                    }
                }
            }
        }
        .task { await viewModel.load() }
    }

    private var calendarGrid: some View {
        VStack(spacing: 4) {
            LazyVGrid(columns: columns, spacing: 0) {
                ForEach(Array(weekdaySymbols.enumerated()), id: \.offset) { _, symbol in
                    Text(symbol)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.tertiary)
                        .frame(maxWidth: .infinity)
                }
            }
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(Array(daysInMonth.enumerated()), id: \.offset) { _, day in
                    if let day {
                        DayCell(
                            date: day,
                            isSelected: Calendar.current.isDate(day, inSameDayAs: viewModel.selectedDay ?? .distantPast),
                            isToday: Calendar.current.isDateInToday(day),
                            hasChores: !viewModel.chores(for: day).isEmpty
                        )
                        .onTapGesture { viewModel.selectedDay = day }
                    } else {
                        Color.clear.frame(height: 36)
                    }
                }
            }
        }
        .padding(.vertical, 10)
    }

    private var agendaSection: some View {
        Group {
            if let selectedDay = viewModel.selectedDay {
                let chores = viewModel.chores(for: selectedDay)
                if chores.isEmpty {
                    ContentUnavailableView(
                        "No chores on this day",
                        systemImage: "calendar",
                        description: Text("All clear!")
                    )
                } else {
                    List(chores) { chore in
                        HStack(spacing: 10) {
                            RoundedRectangle(cornerRadius: 2)
                                .fill(chore.status == .overdue ? .red : .accentColor)
                                .frame(width: 4, height: 32)
                            VStack(alignment: .leading, spacing: 2) {
                                Text(chore.name)
                                    .font(.body)
                                if let badge = chore.badge {
                                    Text(badge.label)
                                        .font(.caption)
                                        .foregroundStyle(badge.color)
                                }
                            }
                            Spacer()
                        }
                    }
                    .listStyle(.insetGrouped)
                }
            } else {
                ContentUnavailableView("Select a day", systemImage: "calendar")
            }
        }
    }

    private var daysInMonth: [Date?] {
        let calendar = Calendar(identifier: .iso8601)
        guard let range = calendar.range(of: .day, in: .month, for: viewModel.currentMonth),
              let firstDay = calendar.date(from: calendar.dateComponents([.year, .month], from: viewModel.currentMonth))
        else { return [] }
        let weekdayOffset = (calendar.component(.weekday, from: firstDay) + 5) % 7
        var days: [Date?] = Array(repeating: nil, count: weekdayOffset)
        days += range.compactMap { day in
            calendar.date(byAdding: .day, value: day - 1, to: firstDay)
        }
        return days
    }
}

private struct DayCell: View {
    let date: Date
    let isSelected: Bool
    let isToday: Bool
    let hasChores: Bool

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    var body: some View {
        VStack(spacing: 2) {
            Text(Self.dayFormatter.string(from: date))
                .font(.subheadline)
                .foregroundStyle(isSelected ? .white : .primary)
                .frame(width: 30, height: 30)
                .background(isSelected ? Color.accentColor : (isToday ? Color.accentColor.opacity(0.15) : Color.clear))
                .clipShape(Circle())
            Circle()
                .fill(hasChores ? .orange : Color.clear)
                .frame(width: 4, height: 4)
        }
    }
}
```

- [ ] **Step 2: Run calendar tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:FamilyHubTests/CalendarViewModelTests 2>&1 | tail -10
```

Expected: All CalendarViewModel tests pass.

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "refactor(ios): rewrite CalendarView with native styling and color-coded agenda"
```

---

### Task 7: Rewrite ProfileView, SetupView, ConfigurationFormView

Strip all Theme references from remaining view files.

**Files:**
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Profile/ProfileView.swift`
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Settings/SetupView.swift`
- Rewrite: `ios/FamilyHub/FamilyHub/Features/Settings/ConfigurationFormView.swift`

- [ ] **Step 1: Rewrite ProfileView**

Replace `ios/FamilyHub/FamilyHub/Features/Profile/ProfileView.swift`:

```swift
import SwiftUI

struct ProfileView: View {
    @Environment(AuthManager.self) private var authManager
    @Environment(ConfigStore.self) private var configStore
    @Environment(\.dismiss) private var dismiss

    @State private var showingEditConfigConfirmation = false
    @State private var showingEditConfig = false

    var body: some View {
        NavigationStack {
            List {
                Section {
                    HStack(spacing: 14) {
                        UserAvatar(user: nil, size: 52)
                        VStack(alignment: .leading, spacing: 4) {
                            Text(authManager.displayName)
                                .font(.body.weight(.semibold))
                            if !authManager.email.isEmpty {
                                Text(authManager.email)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    .padding(.vertical, 4)
                }

                Section {
                    Button("Edit Configuration") {
                        showingEditConfigConfirmation = true
                    }
                }

                Section {
                    Button("Sign Out", role: .destructive) {
                        authManager.signOut()
                        dismiss()
                    }
                }
            }
            .listStyle(.insetGrouped)
            .navigationTitle("Profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .confirmationDialog(
                "Editing your configuration will sign you out. Continue?",
                isPresented: $showingEditConfigConfirmation,
                titleVisibility: .visible
            ) {
                Button("Edit Configuration", role: .destructive) {
                    showingEditConfig = true
                }
                Button("Cancel", role: .cancel) {}
            }
            .sheet(isPresented: $showingEditConfig) {
                NavigationStack {
                    ConfigurationFormView(
                        configStore: configStore,
                        discoveryService: URLSessionOIDCDiscoveryService(),
                        onSave: { authManager.signOut() }
                    )
                    .navigationTitle("Edit Configuration")
                    .navigationBarTitleDisplayMode(.inline)
                }
            }
        }
    }
}
```

- [ ] **Step 2: Rewrite SetupView**

Replace `ios/FamilyHub/FamilyHub/Features/Settings/SetupView.swift`:

```swift
import SwiftUI

struct SetupView: View {
    @Environment(ConfigStore.self) private var configStore

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                VStack(spacing: 8) {
                    Text("Welcome to Family Hub")
                        .font(.title2.bold())
                    Text("Enter your server URL to get started.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }
                .padding(.top, 48)
                .padding(.horizontal)

                ConfigurationFormView(
                    configStore: configStore,
                    discoveryService: URLSessionOIDCDiscoveryService(),
                    onSave: {}
                )
            }
            .navigationTitle("Setup")
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}
```

- [ ] **Step 3: Rewrite ConfigurationFormView**

Replace `ios/FamilyHub/FamilyHub/Features/Settings/ConfigurationFormView.swift`:

```swift
import SwiftUI

struct ConfigurationFormView: View {
    let configStore: ConfigStore
    let discoveryService: OIDCDiscoveryService
    let onSave: () -> Void

    @State private var baseURL: String
    @State private var isDiscovering = false
    @State private var discoveryError: String?

    @Environment(\.dismiss) private var dismiss

    init(configStore: ConfigStore, discoveryService: OIDCDiscoveryService, onSave: @escaping () -> Void) {
        self.configStore = configStore
        self.discoveryService = discoveryService
        self.onSave = onSave
        _baseURL = State(initialValue: configStore.baseURL)
    }

    var body: some View {
        List {
            Section("Server") {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Base URL")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    TextField("https://hub.example.com", text: $baseURL)
                        .font(.subheadline)
                        .keyboardType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                }
                .padding(.vertical, 2)

                if let error = discoveryError {
                    Text(error)
                        .font(.caption)
                        .foregroundStyle(.red)
                }
            }
        }
        .listStyle(.insetGrouped)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                if isDiscovering {
                    ProgressView()
                } else {
                    Button("Connect") {
                        Task { await connect() }
                    }
                    .disabled(baseURL.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            ToolbarItem(placement: .cancellationAction) {
                Button("Cancel") { dismiss() }
            }
        }
    }

    @MainActor
    private func connect() async {
        discoveryError = nil
        isDiscovering = true
        defer { isDiscovering = false }

        guard let url = URL(string: baseURL.trimmingCharacters(in: .whitespaces)),
              url.scheme == "http" || url.scheme == "https" else {
            discoveryError = "Enter a valid http or https URL"
            return
        }

        do {
            let result = try await discoveryService.discover(baseURL: url)
            configStore.baseURL = url.absoluteString
            configStore.applyDiscovery(result)
            configStore.save()
            onSave()
            dismiss()
        } catch {
            discoveryError = error.localizedDescription
        }
    }
}
```

- [ ] **Step 4: Verify LoginView has no Theme references**

```bash
grep -c "Theme\." ios/FamilyHub/FamilyHub/Auth/LoginView.swift
```

Expected: `0` — LoginView already uses system colors.

- [ ] **Step 5: Build and run all tests**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -20
```

Expected: BUILD SUCCEEDED, all tests pass.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "refactor(ios): strip Theme from ProfileView, SetupView, ConfigurationFormView"
```

---

### Task 8: Update UserAvatar, delete Theme.swift and custom components, final cleanup

Now that no view references `Theme.x`, `StatCard`, `SectionCard`, or `StatusBadge`, delete them. Update UserAvatar to use system colors. Remove old Dashboard files. Update Xcode project. Set iOS 26 deployment target.

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Components/UserAvatar.swift`
- Delete: `ios/FamilyHub/FamilyHub/DesignSystem/Theme.swift`
- Delete: `ios/FamilyHub/FamilyHub/Components/StatCard.swift`
- Delete: `ios/FamilyHub/FamilyHub/Components/SectionCard.swift`
- Delete: `ios/FamilyHub/FamilyHub/Components/StatusBadge.swift`
- Delete: `ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardView.swift`
- Delete: `ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardViewModel.swift`
- Delete: `ios/FamilyHub/FamilyHubTests/Dashboard/DashboardViewModelTests.swift`
- Delete: `ios/FamilyHub/FamilyHubTests/Components/StatusBadgeTests.swift`
- Modify: `ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj`

- [ ] **Step 1: Update UserAvatar to use system colors**

Replace `ios/FamilyHub/FamilyHub/Components/UserAvatar.swift`:

```swift
import SwiftUI

struct UserAvatar: View {
    let user: User?
    let size: CGFloat

    var body: some View {
        Circle()
            .fill(Color(.tertiarySystemFill))
            .frame(width: size, height: size)
            .overlay {
                if let user, !user.avatarURL.isEmpty, let url = URL(string: user.avatarURL) {
                    AsyncImage(url: url) { phase in
                        switch phase {
                        case .success(let image):
                            image.resizable().scaledToFill()
                        default:
                            initialsLabel(for: user)
                        }
                    }
                    .clipShape(Circle())
                } else {
                    initialsLabel(for: user)
                }
            }
            .clipShape(Circle())
    }

    private func initialsLabel(for user: User?) -> some View {
        Text(user?.initials ?? "?")
            .font(.system(size: size * 0.38, weight: .semibold))
            .foregroundStyle(.white)
    }
}
```

- [ ] **Step 2: Verify no Theme references remain in source files**

```bash
grep -r "Theme\." ios/FamilyHub/FamilyHub/ --include="*.swift"
```

Expected: Only hits in the files we're about to delete: `Theme.swift`, `StatCard.swift`, `SectionCard.swift`, `StatusBadge.swift`, `DashboardView.swift`, `DashboardViewModel.swift` (if any reference Theme indirectly). The `UserAvatar.swift` preview block was the only reference — now removed.

- [ ] **Step 3: Delete files**

```bash
rm ios/FamilyHub/FamilyHub/DesignSystem/Theme.swift
rmdir ios/FamilyHub/FamilyHub/DesignSystem
rm ios/FamilyHub/FamilyHub/Components/StatCard.swift
rm ios/FamilyHub/FamilyHub/Components/SectionCard.swift
rm ios/FamilyHub/FamilyHub/Components/StatusBadge.swift
rm ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardView.swift
rm ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardViewModel.swift
rmdir ios/FamilyHub/FamilyHub/Features/Dashboard
rm ios/FamilyHub/FamilyHubTests/Dashboard/DashboardViewModelTests.swift
rmdir ios/FamilyHub/FamilyHubTests/Dashboard
rm ios/FamilyHub/FamilyHubTests/Components/StatusBadgeTests.swift
rmdir ios/FamilyHub/FamilyHubTests/Components
```

- [ ] **Step 4: Update project.pbxproj**

Remove all references to deleted files from the Xcode project:
- `Theme.swift`
- `StatCard.swift`
- `SectionCard.swift`
- `StatusBadge.swift`
- `StatusBadgeTests.swift`
- `DashboardView.swift`
- `DashboardViewModel.swift`
- `DashboardViewModelTests.swift`
- `ChoresView.swift` (deleted in Task 4)

Verify all new files are properly referenced:
- `HomeView.swift` and `HomeViewModel.swift` in `Features/Home/`
- `HomeViewModelTests.swift` in test target
- `ChoreBadgeTests.swift` in test target
- `ChoresListView.swift` in `Features/Chores/`
- Recipe files under `Features/Meals/` (not `Features/Recipes/`)

- [ ] **Step 5: Update iOS deployment target to 26.0**

In `project.pbxproj`, update `IPHONEOS_DEPLOYMENT_TARGET` from the current value to `26.0` for all build configurations (Debug and Release) and all targets (FamilyHub, FamilyHubTests, FamilyHubUITests).

- [ ] **Step 6: Clean build**

```bash
cd ios/FamilyHub && xcodebuild clean build -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -10
```

Expected: BUILD SUCCEEDED.

- [ ] **Step 7: Run full test suite**

```bash
cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 16' 2>&1 | tail -20
```

Expected: All tests pass. No compilation warnings about missing files.

- [ ] **Step 8: Final verification — no Theme references, correct structure**

```bash
grep -r "Theme\." ios/FamilyHub/ --include="*.swift"
```

Expected: Zero results.

```bash
find ios/FamilyHub/FamilyHub -name "*.swift" | sort
```

Expected directory structure:
- `Auth/AuthManager.swift`, `KeychainStore.swift`, `LoginView.swift`
- `Components/UserAvatar.swift`
- `Config/ConfigStore.swift`, `OIDCDiscoveryService.swift`
- `FamilyHubApp.swift`
- `Features/Calendar/CalendarView.swift`, `CalendarViewModel.swift`
- `Features/Chores/ChoreDetailView.swift`, `ChoresListView.swift`, `ChoresViewModel.swift`
- `Features/ContentView.swift`
- `Features/Home/HomeView.swift`, `HomeViewModel.swift`
- `Features/Meals/MealsView.swift`, `MealsViewModel.swift`, `RecipeDetailView.swift`, `RecipesView.swift`, `RecipesViewModel.swift`
- `Features/Profile/ProfileView.swift`
- `Features/Settings/ConfigurationFormView.swift`, `SetupView.swift`
- `Models/...`
- `Networking/...`
- NO `DesignSystem/` directory
- NO `Features/Dashboard/` directory
- NO `Features/Recipes/` directory

- [ ] **Step 9: Commit**

```bash
git add -A && git commit -m "chore(ios): delete Theme.swift, custom components, old Dashboard files, set iOS 26 target"
```
