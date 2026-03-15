# iOS App Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the Family Hub iOS app to match the dark-navy Suskins Hub web UI aesthetic, with proper iOS design conventions and three targeted UX fixes (pull-to-refresh, inline errors, logout).

**Architecture:** Build a shared design system (Theme + 4 components) first, then replace each screen's visuals top-down. All ViewModels are kept; only the View layer and one ViewModel computed-property change (Chores grouping) are touched. User data (names, avatars) is loaded once via a new `fetchUsers()` API call and passed down as a lookup dictionary.

**Tech Stack:** SwiftUI, Swift 5.9+, `@Observable`, XCTest + FakeAPIClient, Xcode 15+

---

> **Xcode file creation note:** Whenever a step says "Create new file in Xcode", use **File → New → Swift File** in Xcode, enter the filename, and ensure **only the correct target** is checked (`FamilyHub` for app files, `FamilyHubTests` for test files). This registers the file in the `.xcodeproj`; creating files in Finder alone will not compile.

> **Run tests:** `cd ios/FamilyHub && xcodebuild test -scheme FamilyHub -destination 'platform=iOS Simulator,OS=latest,name=iPhone 15' 2>&1 | xcpretty` (install xcpretty with `gem install xcpretty` if missing, or omit the pipe).

> **Known constraints:**
> - `Chore.assignedToUserID` is a user ID string, not a display name. User names come from `GET /api/users`, which is added in Task 2.
> - The leaderboard has no REST API endpoint — the Dashboard Leaderboard section is scaffolded with an empty state pending a future `GET /api/dashboard/leaderboard` server endpoint.

---

## Chunk 1: Design System

### Task 1: Theme — colour tokens

**Files:**
- Create: `ios/FamilyHub/FamilyHub/DesignSystem/Theme.swift` (target: FamilyHub)

- [ ] **Step 1: Create the file in Xcode**

  File → New → Swift File → `Theme.swift` inside a new Group `DesignSystem` under `FamilyHub/`. Target: FamilyHub.

- [ ] **Step 2: Write Theme.swift**

```swift
import SwiftUI

/// All design-system colour tokens for the dark-navy theme.
enum Theme {
    static let background      = Color(hex: "0f172a")
    static let surface         = Color(hex: "1e293b")
    static let surfaceElevated = Color(hex: "334155")
    static let borderDivider   = Color(hex: "0f172a") // matches background — creates slot-gap between surface rows
    static let textPrimary     = Color(hex: "f1f5f9")
    static let textSecondary   = Color(hex: "94a3b8")
    static let textMuted       = Color(hex: "475569")
    static let accent          = Color(hex: "60a5fa")
    static let statusRed       = Color(hex: "ef4444")
    static let statusAmber     = Color(hex: "f59e0b")
    static let statusGreen     = Color(hex: "4ade80")
    static let doneButtonBg    = Color(hex: "1e3a2f")
    static let doneButtonBorder = Color(hex: "16a34a").opacity(0.2)
    static let avatarFallback  = Color(hex: "6366f1")
}

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var value: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&value)
        let red   = Double((value >> 16) & 0xFF) / 255
        let green = Double((value >> 8)  & 0xFF) / 255
        let blue  = Double(value         & 0xFF) / 255
        self.init(red: red, green: green, blue: blue)
    }
}
```

- [ ] **Step 3: Build to confirm it compiles**

  In Xcode: ⌘B. Expected: Build Succeeded, zero errors.

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/DesignSystem/Theme.swift ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj
git commit -m "feat(ios): add Theme design system with dark-navy colour tokens"
```

---

### Task 2: User model + API

Adding user data so chore rows can show names and avatars.

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Models/User.swift` (target: FamilyHub)
- Modify: `ios/FamilyHub/FamilyHub/Networking/APIClientProtocol.swift`
- Modify: `ios/FamilyHub/FamilyHub/Networking/APIClient.swift`
- Modify: `ios/FamilyHub/FamilyHubTests/FakeAPIClient.swift`
- Create: `ios/FamilyHub/FamilyHubTests/Models/UserTests.swift` (target: FamilyHubTests)

- [ ] **Step 1: Write the failing test**

  Create `ios/FamilyHub/FamilyHubTests/Models/UserTests.swift`:

```swift
import XCTest
@testable import FamilyHub

final class UserTests: XCTestCase {
    func testDecodesFromJSON() throws {
        let json = """
        {"ID":"u1","Name":"Ben Suskins","Email":"ben@example.com","AvatarURL":"","Role":"admin"}
        """.data(using: .utf8)!
        let user = try JSONDecoder().decode(User.self, from: json)
        XCTAssertEqual(user.id, "u1")
        XCTAssertEqual(user.name, "Ben Suskins")
    }

    func testInitialsFromTwoWordName() {
        let user = User(id: "u1", name: "Ben Suskins", email: "", avatarURL: "")
        XCTAssertEqual(user.initials, "BS")
    }

    func testInitialsFromSingleWordName() {
        let user = User(id: "u2", name: "Admin", email: "", avatarURL: "")
        XCTAssertEqual(user.initials, "A")
    }

    func testInitialsFromEmptyName() {
        let user = User(id: "u3", name: "", email: "", avatarURL: "")
        XCTAssertEqual(user.initials, "?")
    }
}
```

- [ ] **Step 2: Run tests to confirm they fail**

  ⌘U in Xcode. Expected: compile error — `User` not found.

- [ ] **Step 3: Create User.swift**

  File → New → Swift File → `User.swift` inside `Models/`. Target: FamilyHub.

```swift
import Foundation

struct User: Codable, Identifiable {
    let id: String
    let name: String
    let email: String
    let avatarURL: String

    var initials: String {
        let parts = name.split(separator: " ").prefix(2)
        guard !parts.isEmpty else { return "?" }
        return parts.compactMap { $0.first.map(String.init) }.joined()
    }

    enum CodingKeys: String, CodingKey {
        case id       = "ID"
        case name     = "Name"
        case email    = "Email"
        case avatarURL = "AvatarURL"
    }
}
```

- [ ] **Step 4: Add `fetchUsers()` to the protocol**

  In `APIClientProtocol.swift`, add:

```swift
func fetchUsers() async throws -> [User]
```

- [ ] **Step 5: Implement in APIClient**

  In `APIClient.swift`, add:

```swift
func fetchUsers() async throws -> [User] {
    try await get("/api/users")
}
```

- [ ] **Step 6: Add to FakeAPIClient**

  In `FakeAPIClient.swift`, add:

```swift
var usersResult: Result<[User], Error> = .success([])

func fetchUsers() async throws -> [User] { try usersResult.get() }
```

- [ ] **Step 7: Run tests**

  ⌘U. Expected: `UserTests` — 4 tests pass.

- [ ] **Step 8: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Models/User.swift \
        ios/FamilyHub/FamilyHub/Networking/APIClientProtocol.swift \
        ios/FamilyHub/FamilyHub/Networking/APIClient.swift \
        ios/FamilyHub/FamilyHubTests/FakeAPIClient.swift \
        ios/FamilyHub/FamilyHubTests/Models/UserTests.swift \
        ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj
git commit -m "feat(ios): add User model and fetchUsers API"
```

---

### Task 3: UserAvatar component

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Components/UserAvatar.swift` (target: FamilyHub)

- [ ] **Step 1: Create the file in Xcode**

  File → New → Swift File → `UserAvatar.swift` inside new Group `Components/`. Target: FamilyHub.

- [ ] **Step 2: Write UserAvatar.swift**

```swift
import SwiftUI

/// Circular avatar showing a user photo (async) or initials fallback.
///
/// Usage:
///   UserAvatar(user: user, size: 32)
///   UserAvatar(user: nil, size: 24)  // shows "?" placeholder
struct UserAvatar: View {
    let user: User?
    let size: CGFloat

    var body: some View {
        Circle()
            .fill(Theme.avatarFallback)
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

#Preview {
    HStack(spacing: 12) {
        UserAvatar(user: User(id: "1", name: "Ben Suskins", email: "", avatarURL: ""), size: 32)
        UserAvatar(user: User(id: "2", name: "Megane Holl", email: "", avatarURL: ""), size: 32)
        UserAvatar(user: nil, size: 32)
        UserAvatar(user: User(id: "1", name: "Ben Suskins", email: "", avatarURL: ""), size: 24)
    }
    .padding()
    .background(Theme.background)
}
```

- [ ] **Step 3: Build and inspect Preview**

  ⌘B, then open the Preview canvas. Should show two avatars with initials "BS" and "MH" in indigo circles, a "?" placeholder, and a smaller 24pt version.

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Components/UserAvatar.swift \
        ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj
git commit -m "feat(ios): add UserAvatar component"
```

---

### Task 4: StatusBadge component

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Components/StatusBadge.swift` (target: FamilyHub)
- Create: `ios/FamilyHub/FamilyHubTests/Components/StatusBadgeTests.swift` (target: FamilyHubTests)

- [ ] **Step 1: Write the failing test**

  Create `ios/FamilyHub/FamilyHubTests/Components/StatusBadgeTests.swift`:

```swift
import XCTest
@testable import FamilyHub

final class StatusBadgeTests: XCTestCase {
    func testOverdueVariantLabel() {
        XCTAssertEqual(StatusBadge.Variant.overdue.label, "Overdue")
    }

    func testDueTodayVariantLabel() {
        XCTAssertEqual(StatusBadge.Variant.dueToday.label, "Today")
    }

    func testDueSoonVariantLabel() {
        XCTAssertEqual(StatusBadge.Variant.dueSoon.label, "Due Soon")
    }

    func testChoreOverdueStatusMapsToOverdueVariant() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .overdue,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertEqual(chore.badgeVariant, .overdue)
    }

    func testChoreCompletedStatusHasNoBadge() {
        let chore = Chore(id: "1", name: "Test", description: "", status: .completed,
                          dueDate: nil, assignedToUserID: nil)
        XCTAssertNil(chore.badgeVariant)
    }

    func testChorePendingDueTodayMapsToTodayVariant() {
        let todayString = ISO8601DateFormatter().string(from: Date())
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: todayString, assignedToUserID: nil)
        XCTAssertEqual(chore.badgeVariant, .dueToday)
    }

    func testChorePendingFutureDateMapsToDueSoonVariant() {
        let futureDate = Calendar.current.date(byAdding: .day, value: 3, to: Date())!
        let futureString = ISO8601DateFormatter().string(from: futureDate)
        let chore = Chore(id: "1", name: "Test", description: "", status: .pending,
                          dueDate: futureString, assignedToUserID: nil)
        XCTAssertEqual(chore.badgeVariant, .dueSoon)
    }
}
```

- [ ] **Step 2: Run tests — confirm they fail**

  ⌘U. Expected: compile error — `StatusBadge` not found.

- [ ] **Step 3: Create StatusBadge.swift**

  File → New → Swift File → `StatusBadge.swift` inside `Components/`. Target: FamilyHub.

```swift
import SwiftUI

struct StatusBadge: View {
    enum Variant: Equatable {
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

        var textColor: Color {
            switch self {
            case .overdue:          return Theme.statusRed
            case .dueToday, .dueSoon: return Theme.statusAmber
            }
        }

        var backgroundColor: Color { textColor.opacity(0.13) }
    }

    let variant: Variant

    var body: some View {
        Text(variant.label)
            .font(.system(size: 10, weight: .semibold))
            .foregroundStyle(variant.textColor)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(variant.backgroundColor)
            .clipShape(RoundedRectangle(cornerRadius: 4))
    }
}

// MARK: - Chore convenience

extension Chore {
    var badgeVariant: StatusBadge.Variant? {
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

#Preview {
    HStack(spacing: 8) {
        StatusBadge(variant: .overdue)
        StatusBadge(variant: .dueToday)
        StatusBadge(variant: .dueSoon)
    }
    .padding()
    .background(Theme.background)
}
```

- [ ] **Step 4: Run tests**

  ⌘U. Expected: `StatusBadgeTests` — 7 tests pass.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Components/StatusBadge.swift \
        ios/FamilyHub/FamilyHubTests/Components/StatusBadgeTests.swift \
        ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj
git commit -m "feat(ios): add StatusBadge component with Chore convenience extension"
```

---

### Task 5: StatCard and SectionCard components

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Components/StatCard.swift` (target: FamilyHub)
- Create: `ios/FamilyHub/FamilyHub/Components/SectionCard.swift` (target: FamilyHub)

- [ ] **Step 1: Create StatCard.swift**

  File → New → Swift File → `StatCard.swift` in `Components/`. Target: FamilyHub.

```swift
import SwiftUI

/// Small stat card: label (top), large number, subtitle (bottom).
struct StatCard: View {
    let label: String
    let value: Int
    let subtitle: String
    var subtitleColor: Color = Theme.textMuted

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.system(size: 9, weight: .semibold))
                .textCase(.uppercase)
                .tracking(0.6)
                .foregroundStyle(Theme.textMuted)
            Text("\(value)")
                .font(.system(size: 26, weight: .bold))
                .foregroundStyle(Theme.textPrimary)
            Text(subtitle)
                .font(.system(size: 11))
                .foregroundStyle(subtitleColor)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

#Preview {
    HStack(spacing: 8) {
        StatCard(label: "Chores", value: 12, subtitle: "5 overdue", subtitleColor: Theme.statusRed)
        StatCard(label: "Events", value: 4, subtitle: "Next 7 days")
        StatCard(label: "Meals", value: 12, subtitle: "of 21 planned")
    }
    .padding()
    .background(Theme.background)
}
```

- [ ] **Step 2: Create SectionCard.swift**

  File → New → Swift File → `SectionCard.swift` in `Components/`. Target: FamilyHub.

```swift
import SwiftUI

/// Surface-coloured card with a header row (icon + title) and slot for content rows.
///
/// Usage:
///   SectionCard(icon: "checkmark.circle", iconColor: .teal, title: "Chores Due") {
///       ForEach(rows) { row in ChoreRow(chore: row) }
///   }
struct SectionCard<Content: View>: View {
    let icon: String
    let iconColor: Color
    let title: String
    let content: Content

    init(icon: String, iconColor: Color, title: String, @ViewBuilder content: () -> Content) {
        self.icon = icon
        self.iconColor = iconColor
        self.title = title
        self.content = content()
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Image(systemName: icon)
                    .foregroundStyle(iconColor)
                    .font(.system(size: 15))
                Text(title)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(Theme.textPrimary)
                Spacer()
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)

            Rectangle()
                .fill(Theme.borderDivider)
                .frame(height: 1)

            content
        }
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
```

- [ ] **Step 3: Build and inspect Previews**

  ⌘B. Open Preview canvas for StatCard: should show 3 cards with navy backgrounds. Build must succeed.

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Components/StatCard.swift \
        ios/FamilyHub/FamilyHub/Components/SectionCard.swift \
        ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj
git commit -m "feat(ios): add StatCard and SectionCard design system components"
```

---

## Chunk 2: Dashboard + Profile

### Task 6: DashboardViewModel — add users lookup

The Dashboard needs user names for chore rows.

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardViewModel.swift`
- Modify: `ios/FamilyHub/FamilyHubTests/Dashboard/DashboardViewModelTests.swift`

- [ ] **Step 1: Write the new failing test**

  Open `DashboardViewModelTests.swift`. Add:

```swift
func testLoadFetchesUsersAlongWithStats() async {
    let fake = FakeAPIClient()
    fake.dashboardResult = .success(
        DashboardStats(choresDueToday: 1, choresOverdue: 0, choresDueTodayList: [], choresOverdueList: [])
    )
    fake.usersResult = .success([User(id: "u1", name: "Ben Suskins", email: "", avatarURL: "")])
    let viewModel = DashboardViewModel(apiClient: fake)

    await viewModel.load()

    XCTAssertEqual(viewModel.users["u1"]?.name, "Ben Suskins")
}
```

- [ ] **Step 2: Run test — confirm it fails**

  ⌘U. Expected: compile error — `users` property missing on `DashboardViewModel`.

- [ ] **Step 3: Update DashboardViewModel.swift**

  Replace the entire file with:

```swift
import Foundation
import Observation

@Observable
@MainActor
final class DashboardViewModel {
    var state: ViewState<DashboardStats> = .idle
    var users: [String: User] = [:]

    private let apiClient: any APIClientProtocol

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
}
```

- [ ] **Step 4: Run all tests**

  ⌘U. Expected: `DashboardViewModelTests` — all tests pass including the new one.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardViewModel.swift \
        ios/FamilyHub/FamilyHubTests/Dashboard/DashboardViewModelTests.swift
git commit -m "feat(ios): load users in DashboardViewModel for assignee display"
```

---

### Task 7: DashboardView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardView.swift`

- [ ] **Step 1: Replace DashboardView.swift entirely**

```swift
import SwiftUI

struct DashboardView: View {
    @State private var viewModel: DashboardViewModel
    @State private var showProfile = false

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: DashboardViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack(alignment: .top) {
                Theme.background.ignoresSafeArea()
                ScrollView {
                    VStack(spacing: 0) {
                        if case .failed(let error) = viewModel.state {
                            inlineError(error.localizedDescription)
                        }
                        if case .loaded(let stats) = viewModel.state {
                            dashboardContent(stats)
                        } else if case .loading = viewModel.state {
                            ProgressView()
                                .tint(Theme.textSecondary)
                                .padding(.top, 60)
                        }
                    }
                    .padding(.horizontal, 14)
                }
                .refreshable { await viewModel.load() }
            }
            .navigationTitle("Overview")
            .navigationBarTitleDisplayMode(.large)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showProfile = true
                    } label: {
                        // Intentional stub: shows "?" until OIDC claim storage is wired up (out of scope).
                        // This is expected — do not treat the "?" placeholder as a bug.
                        UserAvatar(user: nil, size: 32)
                    }
                    .buttonStyle(.plain)
                }
            }
            .sheet(isPresented: $showProfile) {
                ProfileView()
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Content

    @ViewBuilder
    private func dashboardContent(_ stats: DashboardStats) -> some View {
        VStack(spacing: 14) {
            // Stat row
            HStack(spacing: 8) {
                StatCard(
                    label: "Chores",
                    value: stats.choresDueToday + stats.choresOverdue,
                    subtitle: stats.choresOverdue > 0 ? "\(stats.choresOverdue) overdue" : "None overdue",
                    subtitleColor: stats.choresOverdue > 0 ? Theme.statusRed : Theme.textMuted
                )
                StatCard(label: "Events", value: 0, subtitle: "Next 7 days")
                StatCard(label: "Meals", value: 0, subtitle: "of 21 planned")
            }
            .padding(.top, 8)

            // Chores Due section
            let allDue = stats.choresOverdueList + stats.choresDueTodayList
            if !allDue.isEmpty {
                SectionCard(icon: "checkmark.circle", iconColor: .teal, title: "Chores Due") {
                    ForEach(allDue) { chore in
                        choreDueRow(chore)
                    }
                }
            }

            // Today's Meals section
            SectionCard(icon: "flame", iconColor: Theme.statusAmber, title: "Today's Meals") {
                mealRow(label: "Lunch", name: nil)
                mealRow(label: "Dinner", name: nil)
            }

            // Leaderboard section (placeholder — requires server endpoint)
            SectionCard(icon: "trophy", iconColor: Theme.statusAmber, title: "Leaderboard") {
                Text("Leaderboard coming soon")
                    .font(.system(size: 13))
                    .foregroundStyle(Theme.textMuted)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 12)
            }
        }
        .padding(.bottom, 20)
    }

    // MARK: - Row helpers

    private func choreDueRow(_ chore: Chore) -> some View {
        HStack(spacing: 10) {
            UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(Theme.textPrimary)
                HStack(spacing: 6) {
                    if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                        Text(user.name)
                            .font(.system(size: 12))
                            .foregroundStyle(Theme.textSecondary)
                    }
                    if let variant = chore.badgeVariant {
                        StatusBadge(variant: variant)
                    }
                    if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.system(size: 11))
                            .foregroundStyle(chore.status == .overdue ? Theme.statusRed : Theme.statusAmber)
                    }
                }
            }
            Spacer()
            DoneButton(choreID: chore.id, viewModel: viewModel)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)

        Rectangle()
            .fill(Theme.borderDivider)
            .frame(height: 1)
            .padding(.leading, 14 + 32 + 10)
    }

    private func mealRow(label: String, name: String?) -> some View {
        HStack(spacing: 12) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(Theme.textMuted)
                .frame(width: 50, alignment: .leading)
            Text(name ?? "— not planned —")
                .font(.system(size: 14))
                .foregroundStyle(name != nil ? Theme.textPrimary : Theme.textMuted)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)

        Rectangle()
            .fill(Theme.borderDivider)
            .frame(height: 1)
    }

    private func inlineError(_ message: String) -> some View {
        Text(message)
            .font(.system(size: 13))
            .foregroundStyle(Theme.statusRed)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
    }
}

// MARK: - Done button

private struct DoneButton: View {
    let choreID: String
    let viewModel: DashboardViewModel
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 4) {
            Button {
                Task {
                    isLoading = true
                    // Dashboard doesn't share ChoresViewModel; re-fetch after complete
                    do {
                        try await viewModel.apiClient.completeChore(id: choreID)
                        await viewModel.load()
                    } catch {
                        errorMessage = error.localizedDescription
                    }
                    isLoading = false
                }
            } label: {
                HStack(spacing: 4) {
                    if isLoading {
                        ProgressView().scaleEffect(0.7).tint(Theme.statusGreen)
                    } else {
                        Image(systemName: "checkmark")
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(Theme.statusGreen)
                        Text("Done")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(Theme.statusGreen)
                    }
                }
                .padding(.horizontal, 9)
                .padding(.vertical, 5)
                .background(Theme.doneButtonBg)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(Theme.doneButtonBorder, lineWidth: 1)
                )
                .clipShape(RoundedRectangle(cornerRadius: 8))
            }
            .buttonStyle(.plain)
            .disabled(isLoading)

            if let errorMessage {
                Text(errorMessage)
                    .font(.system(size: 9))
                    .foregroundStyle(Theme.statusRed)
                    .lineLimit(2)
                    .frame(maxWidth: 80)
            }
        }
    }
}
```

- [ ] **Step 2: Expose apiClient on DashboardViewModel**

  In `DashboardViewModel.swift`, change `private let apiClient` to `let apiClient` (remove `private`). The `DoneButton` code in Step 1 already uses `viewModel.apiClient.completeChore(id: choreID)` — no other change needed.

- [ ] **Step 3: Add `formattedDueDate` to Chore**

  In `Chore.swift`, add an extension:

```swift
extension Chore {
    var formattedDueDate: String? {
        guard let dueDate else { return nil }
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        let iso = ISO8601DateFormatter()
        guard let date = iso.date(from: dueDate) ?? formatter.date(from: String(dueDate.prefix(10))) else {
            return nil
        }
        let display = DateFormatter()
        display.dateFormat = "MMM d"
        return display.string(from: date)
    }
}
```

- [ ] **Step 4: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 5: Run app on simulator — verify Dashboard**

  ⌘R. Dashboard should show dark navy background, stat cards, chore due rows with avatar + assignee + badge + Done button. Pull down to refresh. No Refresh button in toolbar.

- [ ] **Step 6: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardView.swift \
        ios/FamilyHub/FamilyHub/Features/Dashboard/DashboardViewModel.swift \
        ios/FamilyHub/FamilyHub/Models/Chore.swift
git commit -m "feat(ios): redesign DashboardView with dark-navy theme, stat cards, chore rows"
```

---

### Task 8: ProfileView

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Features/Profile/ProfileView.swift` (target: FamilyHub)

- [ ] **Step 1: Create the file in Xcode**

  File → New → Swift File → `ProfileView.swift` inside new group `Features/Profile/`. Target: FamilyHub.

- [ ] **Step 2: Write ProfileView.swift**

```swift
import SwiftUI

struct ProfileView: View {
    @Environment(AuthManager.self) private var authManager
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                List {
                    Section {
                        HStack(spacing: 14) {
                            UserAvatar(user: currentUser, size: 52)
                            VStack(alignment: .leading, spacing: 4) {
                                Text(authManager.displayName)
                                    .font(.system(size: 16, weight: .semibold))
                                    .foregroundStyle(Theme.textPrimary)
                                Text(authManager.email)
                                    .font(.system(size: 13))
                                    .foregroundStyle(Theme.textSecondary)
                            }
                        }
                        .padding(.vertical, 4)
                        .listRowBackground(Theme.surface)
                    }

                    Section {
                        Button(role: .destructive) {
                            authManager.logout()
                            dismiss()
                        } label: {
                            Text("Sign Out")
                                .font(.system(size: 15, weight: .medium))
                                .foregroundStyle(Theme.statusRed)
                        }
                        .listRowBackground(Theme.surface)
                    }
                }
                .scrollContentBackground(.hidden)
                .listStyle(.insetGrouped)
            }
            .navigationTitle("Profile")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                        .foregroundStyle(Theme.accent)
                }
            }
        }
    }

    private var currentUser: User? { nil } // AuthManager doesn't expose User model; initials come from displayName below
}
```

> **Step 2a — Add `displayName` and `email` to AuthManager:**
>
> `AuthManager` currently has no exposed `displayName` or `email` properties. Add stubs — they will show placeholder values until a future task wires up OIDC claim storage.
>
> In `AuthManager.swift`, add (after `var isAuthenticated`):
>
> ```swift
> var displayName: String { "Family Member" }
> var email: String { "" }
> ```
>
> This is intentionally a stub. The Profile sheet will show "Family Member" with a blank email. This is the expected output at Step 4 — not a failure. Wiring up real OIDC claim storage is out of scope for this redesign.

- [ ] **Step 3: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 4: Test sign out flow**

  Run app. Tap the avatar button on the Dashboard → Profile sheet should open. Tap Sign Out → should dismiss and return to LoginView.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Profile/ProfileView.swift \
        ios/FamilyHub/FamilyHub/Auth/AuthManager.swift \
        ios/FamilyHub/FamilyHub.xcodeproj/project.pbxproj
git commit -m "feat(ios): add ProfileView sheet with Sign Out via AuthManager.logout()"
```

---

## Chunk 3: Chores

### Task 9: ChoresViewModel — split pending into overdue and due-soon

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Chores/ChoresViewModel.swift`
- Modify: `ios/FamilyHub/FamilyHubTests/Chores/ChoresViewModelTests.swift`

- [ ] **Step 1: Write the failing tests**

  Add to `ChoresViewModelTests.swift`:

```swift
func testOverdueChoresAreGroupedSeparately() async {
    let fake = FakeAPIClient()
    fake.choresResult = .success([
        makeChore(id: "1", status: .overdue),
        makeChore(id: "2", status: .pending),
    ])
    let viewModel = ChoresViewModel(apiClient: fake)

    await viewModel.load()

    XCTAssertEqual(viewModel.overdueChores.count, 1)
    XCTAssertEqual(viewModel.dueSoonChores.count, 1)
}

func testCompletedChoreDoesNotAppearInOverdueOrDueSoon() async {
    let fake = FakeAPIClient()
    fake.choresResult = .success([makeChore(id: "1", status: .completed)])
    let viewModel = ChoresViewModel(apiClient: fake)

    await viewModel.load()

    XCTAssertEqual(viewModel.overdueChores.count, 0)
    XCTAssertEqual(viewModel.dueSoonChores.count, 0)
    XCTAssertEqual(viewModel.completedChores.count, 1)
}
```

- [ ] **Step 2: Run tests — confirm they fail**

  ⌘U. Expected: compile error — `overdueChores`, `dueSoonChores` not found.

- [ ] **Step 3: Update ChoresViewModel.swift**

  Add two new computed properties (keep existing `pendingChores` for backward compatibility until all callers are updated in Task 10):

```swift
var overdueChores: [Chore] {
    guard case .loaded(let chores) = state else { return [] }
    return chores.filter { $0.status == .overdue }
}

var dueSoonChores: [Chore] {
    guard case .loaded(let chores) = state else { return [] }
    return chores.filter { $0.status == .pending }
}
```

  Also add `users: [String: User] = [:]` and load them in `load()` the same way as `DashboardViewModel`:

```swift
var users: [String: User] = [:]

func load() async {
    state = .loading
    async let choresTask = apiClient.fetchChores()
    async let usersTask  = apiClient.fetchUsers()
    do {
        let (chores, userList) = try await (choresTask, usersTask)
        users = Dictionary(uniqueKeysWithValues: userList.map { ($0.id, $0) })
        state = .loaded(chores)
    } catch let error as APIError {
        state = .failed(error)
    } catch {
        state = .failed(.network(error))
    }
}
```

- [ ] **Step 4: Run all tests**

  ⌘U. Expected: all existing tests + 2 new tests pass.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Chores/ChoresViewModel.swift \
        ios/FamilyHub/FamilyHubTests/Chores/ChoresViewModelTests.swift
git commit -m "feat(ios): split ChoresViewModel pendingChores into overdueChores/dueSoonChores, load users"
```

---

### Task 10: ChoresView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Chores/ChoresView.swift`

- [ ] **Step 1: Replace ChoresView.swift entirely**

```swift
import SwiftUI

struct ChoresView: View {
    @State private var viewModel: ChoresViewModel
    @State private var selectedTab: Tab = .pending

    enum Tab { case pending, completed }

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: ChoresViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                VStack(spacing: 0) {
                    segmentControl
                        .padding(.horizontal, 14)
                        .padding(.vertical, 10)

                    if case .failed(let error) = viewModel.state {
                        Text(error.localizedDescription)
                            .font(.system(size: 13))
                            .foregroundStyle(Theme.statusRed)
                            .padding(.horizontal, 14)
                            .padding(.vertical, 8)
                    }

                    List {
                        if selectedTab == .pending {
                            pendingContent
                        } else {
                            completedContent
                        }
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                    .refreshable { await viewModel.load() }
                }
            }
            .navigationTitle("Chores")
            .navigationBarTitleDisplayMode(.large)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Image(systemName: "line.3.horizontal.decrease")
                        .foregroundStyle(Theme.textMuted)
                }
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Segment control

    private var segmentControl: some View {
        HStack(spacing: 0) {
            segmentButton("Pending", tab: .pending)
            segmentButton("Completed", tab: .completed)
        }
        .padding(3)
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }

    private func segmentButton(_ label: String, tab: Tab) -> some View {
        Button {
            withAnimation(.easeInOut(duration: 0.15)) { selectedTab = tab }
        } label: {
            Text(label)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(selectedTab == tab ? Theme.textPrimary : Theme.textMuted)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 6)
                .background(selectedTab == tab ? Theme.surfaceElevated : Color.clear)
                .clipShape(RoundedRectangle(cornerRadius: 8))
        }
        .buttonStyle(.plain)
    }

    // MARK: - List sections

    @ViewBuilder
    private var pendingContent: some View {
        if !viewModel.overdueChores.isEmpty {
            Section {
                ForEach(viewModel.overdueChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            } header: {
                sectionHeader("Overdue", color: Theme.statusRed)
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Theme.surface)
            .listRowSeparatorTint(Theme.borderDivider)
        }

        if !viewModel.dueSoonChores.isEmpty {
            Section {
                ForEach(viewModel.dueSoonChores) { chore in
                    choreRow(chore, isCompleted: false)
                }
            } header: {
                sectionHeader("Due Soon", color: Theme.statusAmber)
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Theme.surface)
            .listRowSeparatorTint(Theme.borderDivider)
        }

        if viewModel.overdueChores.isEmpty && viewModel.dueSoonChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("All done!", systemImage: "checkmark.circle.fill")
                    .listRowBackground(Color.clear)
            }
        }
    }

    @ViewBuilder
    private var completedContent: some View {
        if viewModel.completedChores.isEmpty {
            if case .loaded = viewModel.state {
                ContentUnavailableView("No completed chores", systemImage: "clock")
                    .listRowBackground(Color.clear)
            }
        } else {
            Section {
                ForEach(viewModel.completedChores) { chore in
                    choreRow(chore, isCompleted: true)
                }
            }
            .listRowInsets(EdgeInsets())
            .listRowBackground(Theme.surface)
            .listRowSeparatorTint(Theme.borderDivider)
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
                        .foregroundStyle(Theme.statusGreen)
                        .font(.system(size: 32))
                        .frame(width: 32, height: 32)
                } else {
                    UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                }
                VStack(alignment: .leading, spacing: 2) {
                    Text(chore.name)
                        .font(.system(size: 15, weight: .medium))
                        .foregroundStyle(Theme.textPrimary)
                    HStack(spacing: 6) {
                        if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                            Text(user.name)
                                .font(.system(size: 12))
                                .foregroundStyle(Theme.textSecondary)
                        }
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 11))
                                .foregroundStyle(isCompleted ? Theme.textMuted :
                                    (chore.status == .overdue ? Theme.statusRed : Theme.statusAmber))
                        }
                    }
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(Theme.textMuted)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)
        }
        .buttonStyle(.plain)
    }

    private func sectionHeader(_ title: String, color: Color) -> some View {
        Text(title)
            .font(.system(size: 11, weight: .semibold))
            .textCase(.uppercase)
            .tracking(0.6)
            .foregroundStyle(color)
            .padding(.horizontal, 14)
            .padding(.vertical, 6)
            .listRowInsets(EdgeInsets())
    }
}
```

- [ ] **Step 2: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 3: Run app — verify Chores tab**

  Switch to Chores tab. Should see Pending/Completed segment, rows grouped into Overdue (red) and Due Soon (amber) sections. Pull to refresh works.

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Chores/ChoresView.swift
git commit -m "feat(ios): redesign ChoresView with segment control and overdue/due-soon grouping"
```

---

### Task 11: ChoreDetailView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Chores/ChoreDetailView.swift`

- [ ] **Step 1: Replace ChoreDetailView.swift**

```swift
import SwiftUI

struct ChoreDetailView: View {
    let chore: Chore
    let viewModel: ChoresViewModel

    @Environment(\.dismiss) private var dismiss
    @State private var isCompleting = false
    @State private var completionError: String?

    var body: some View {
        ZStack {
            Theme.background.ignoresSafeArea()
            List {
                // Assignee row
                Section {
                    HStack(spacing: 12) {
                        UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(viewModel.users[chore.assignedToUserID ?? ""]?.name ?? "Unassigned")
                                .font(.system(size: 14, weight: .medium))
                                .foregroundStyle(Theme.textPrimary)
                            if let badge = chore.badgeVariant {
                                StatusBadge(variant: badge)
                            }
                        }
                        Spacer()
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 13))
                                .foregroundStyle(chore.status == .overdue ? Theme.statusRed : Theme.textSecondary)
                        }
                    }
                    .listRowBackground(Theme.surface)
                }

                // Description
                if !chore.description.isEmpty {
                    Section("Description") {
                        Text(chore.description)
                            .font(.system(size: 14))
                            .foregroundStyle(Theme.textSecondary)
                            .listRowBackground(Theme.surface)
                    }
                }

                // Mark complete button
                if chore.status != .completed {
                    Section {
                        VStack(spacing: 8) {
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
                                        ProgressView().tint(Theme.statusGreen)
                                    } else {
                                        Image(systemName: "checkmark")
                                            .foregroundStyle(Theme.statusGreen)
                                        Text("Mark Complete")
                                            .font(.system(size: 15, weight: .semibold))
                                            .foregroundStyle(Theme.statusGreen)
                                    }
                                    Spacer()
                                }
                                .padding(.vertical, 14)
                                .background(Theme.doneButtonBg)
                                .overlay(
                                    RoundedRectangle(cornerRadius: 12)
                                        .stroke(Theme.doneButtonBorder, lineWidth: 1)
                                )
                                .clipShape(RoundedRectangle(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                            .disabled(isCompleting)

                            if let completionError {
                                Text(completionError)
                                    .font(.system(size: 12))
                                    .foregroundStyle(Theme.statusRed)
                                    .multilineTextAlignment(.center)
                            }
                        }
                        .listRowBackground(Color.clear)
                        .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                    }
                }
            }
            .listStyle(.insetGrouped)
            .scrollContentBackground(.hidden)
        }
        .navigationTitle(chore.name)
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(Theme.background, for: .navigationBar)
    }
}
```

- [ ] **Step 2: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 3: Test inline error**

  With network disconnected (Device → Network → Offline in Simulator), open a chore detail and tap Mark Complete. Should show red error text below the button — no alert dialog.

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Chores/ChoreDetailView.swift
git commit -m "feat(ios): redesign ChoreDetailView with inline error and dark theme"
```

---

## Chunk 4: Meals, Recipes, Calendar, and ContentView

### Task 12: MealsView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Meals/MealsView.swift`

- [ ] **Step 1: Replace MealsView.swift**

  Keep all existing ViewModel integration (`viewModel.currentWeek`, `viewModel.previousWeek()`, etc.) and date formatters. Replace only the visual structure:

```swift
import SwiftUI

struct MealsView: View {
    @State private var viewModel: MealsViewModel

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
        _viewModel = State(wrappedValue: MealsViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                Group {
                    if case .failed(let error) = viewModel.state {
                        VStack {
                            Text(error.localizedDescription)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.statusRed)
                                .padding()
                            Spacer()
                        }
                    } else if case .loaded(let meals) = viewModel.state {
                        mealsContent(meals)
                    } else {
                        ProgressView().tint(Theme.textSecondary)
                    }
                }
            }
            .navigationTitle(weekTitle)
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button {
                        viewModel.previousWeek()
                        Task { await viewModel.load() }
                    } label: {
                        Image(systemName: "chevron.left")
                            .foregroundStyle(Theme.accent)
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        viewModel.nextWeek()
                        Task { await viewModel.load() }
                    } label: {
                        Image(systemName: "chevron.right")
                            .foregroundStyle(Theme.accent)
                    }
                }
            }
        }
        .task { await viewModel.load() }
    }

    private var weekTitle: String {
        let start = viewModel.currentWeek
        let end = Calendar.current.date(byAdding: .day, value: 6, to: start)!
        return "\(Self.weekTitleFormatter.string(from: start)) – \(Self.weekTitleFormatter.string(from: end))"
    }

    private func mealsContent(_ meals: [MealPlan]) -> some View {
        List {
            ForEach(0..<7, id: \.self) { offset in
                let date = Calendar.current.date(byAdding: .day, value: offset, to: viewModel.currentWeek)!
                let dateKey = Self.dateKeyFormatter.string(from: date)
                Section {
                    ForEach(mealTypes, id: \.self) { mealType in
                        let plan = meals.first(where: { $0.date == dateKey && $0.mealType == mealType })
                        HStack(spacing: 12) {
                            Text(mealType.capitalized)
                                .font(.system(size: 12))
                                .foregroundStyle(Theme.textMuted)
                                .frame(width: 70, alignment: .leading)
                            Text(plan?.name ?? "—")
                                .font(.system(size: 14))
                                .foregroundStyle(plan != nil ? Theme.textPrimary : Theme.textMuted)
                        }
                        .padding(.vertical, 4)
                        .listRowBackground(Theme.surface)
                        .listRowSeparatorTint(Theme.borderDivider)
                    }
                } header: {
                    Text(Self.dayFormatter.string(from: date))
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(Theme.textSecondary)
                        .textCase(nil)
                }
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
        .refreshable { await viewModel.load() }
    }
}
```

- [ ] **Step 2: Build and verify**

  ⌘B. Run app → Meals tab should show week sections with dark navy background and styled rows.

- [ ] **Step 3: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Meals/MealsView.swift
git commit -m "feat(ios): redesign MealsView with dark theme and pull-to-refresh"
```

---

### Task 13: RecipesView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Recipes/RecipesView.swift`
- Modify: `ios/FamilyHub/FamilyHub/Features/Recipes/RecipesViewModel.swift`

- [ ] **Step 1: Add search filtering to RecipesViewModel**

  Open `RecipesViewModel.swift`. Add:

```swift
var searchQuery: String = ""

var filteredRecipes: [Recipe] {
    guard case .loaded(let recipes) = state else { return [] }
    guard !searchQuery.isEmpty else { return recipes }
    return recipes.filter { $0.title.localizedCaseInsensitiveContains(searchQuery) }
}
```

- [ ] **Step 2: Replace RecipesView.swift**

```swift
import SwiftUI

struct RecipesView: View {
    @State private var viewModel: RecipesViewModel
    private let apiClient: any APIClientProtocol
    private let columns = [GridItem(.flexible()), GridItem(.flexible())]

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Theme.background.ignoresSafeArea()
                Group {
                    if case .failed(let error) = viewModel.state {
                        VStack {
                            Text(error.localizedDescription)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.statusRed)
                                .padding()
                            Spacer()
                        }
                    } else {
                        ScrollView {
                            LazyVGrid(columns: columns, spacing: 10) {
                                ForEach(viewModel.filteredRecipes) { recipe in
                                    NavigationLink {
                                        RecipeDetailView(recipe: recipe, apiClient: apiClient)
                                    } label: {
                                        RecipeCard(recipe: recipe)
                                    }
                                    .buttonStyle(.plain)
                                }
                            }
                            .padding(.horizontal, 14)
                            .padding(.top, 8)
                        }
                        .refreshable { await viewModel.load() }
                    }
                }
            }
            .navigationTitle("Recipes")
            .navigationBarTitleDisplayMode(.large)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .searchable(text: $viewModel.searchQuery, prompt: "Search recipes")
        }
        .task { await viewModel.load() }
    }
}

private struct RecipeCard: View {
    let recipe: Recipe

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            RoundedRectangle(cornerRadius: 8)
                .fill(Theme.surfaceElevated)
                .aspectRatio(4/3, contentMode: .fit)
                .overlay {
                    Image(systemName: "fork.knife")
                        .foregroundStyle(Theme.textMuted)
                        .font(.system(size: 22))
                }
            VStack(alignment: .leading, spacing: 4) {
                Text(recipe.title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(Theme.textPrimary)
                    .lineLimit(2)
                HStack(spacing: 8) {
                    if let prep = recipe.prepTime {
                        Label("\(prep)m prep", systemImage: "clock")
                            .font(.system(size: 10))
                            .foregroundStyle(Theme.textMuted)
                    }
                    if let servings = recipe.servings {
                        Label("\(servings)", systemImage: "person.2")
                            .font(.system(size: 10))
                            .foregroundStyle(Theme.textMuted)
                    }
                }
            }
            .padding(.horizontal, 6)
            .padding(.bottom, 8)
        }
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
```

- [ ] **Step 3: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 4: Verify search**

  Run app → Recipes tab. Type in search bar — grid should filter. Pull to refresh works.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Recipes/RecipesView.swift \
        ios/FamilyHub/FamilyHub/Features/Recipes/RecipesViewModel.swift
git commit -m "feat(ios): redesign RecipesView with search, dark theme, pull-to-refresh"
```

---

### Task 14: RecipeDetailView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Recipes/RecipeDetailView.swift`

- [ ] **Step 1: Replace RecipeDetailView.swift**

```swift
import SwiftUI
import UIKit

struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol

    @State private var viewModel: RecipeDetailViewModel
    @State private var cookModeActive = false

    init(recipe: Recipe, apiClient: any APIClientProtocol) {
        self.recipe = recipe
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipeDetailViewModel(recipe: recipe, apiClient: apiClient))
    }

    var body: some View {
        ZStack {
            Theme.background.ignoresSafeArea()
            Group {
                if case .loading = viewModel.state {
                    ProgressView().tint(Theme.textSecondary)
                } else if case .failed(let error) = viewModel.state {
                    Text(error.localizedDescription)
                        .foregroundStyle(Theme.statusRed)
                        .padding()
                } else if case .loaded(let detail) = viewModel.state {
                    recipeContent(detail)
                } else {
                    recipeContent(recipe)
                }
            }
        }
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbarBackground(Theme.background, for: .navigationBar)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button {
                    cookModeActive.toggle()
                } label: {
                    Label(cookModeActive ? "Exit Cook Mode" : "Cook Mode",
                          systemImage: cookModeActive ? "flame.fill" : "flame")
                        .font(.system(size: 13))
                        .foregroundStyle(cookModeActive ? Theme.statusAmber : Theme.accent)
                }
            }
        }
        .onAppear {
            if cookModeActive { UIApplication.shared.isIdleTimerDisabled = true }
        }
        .onDisappear {
            UIApplication.shared.isIdleTimerDisabled = false
        }
        .onChange(of: cookModeActive) { _, active in
            UIApplication.shared.isIdleTimerDisabled = active
        }
        .task { await viewModel.load() }
    }

    private func recipeContent(_ r: Recipe) -> some View {
        List {
            // Metadata
            Section {
                HStack(spacing: 16) {
                    if let prep = r.prepTime {
                        metaStat(label: "Prep", value: "\(prep)m")
                    }
                    if let cook = r.cookTime {
                        metaStat(label: "Cook", value: "\(cook)m")
                    }
                    if let servings = r.servings {
                        metaStat(label: "Serves", value: "\(servings)")
                    }
                }
                .listRowBackground(Theme.surface)
            }

            // Ingredients
            if !r.ingredients.isEmpty {
                ForEach(r.ingredients) { group in
                    Section(group.name) {
                        ForEach(group.items, id: \.self) { item in
                            Text(item)
                                .font(.system(size: 14))
                                .foregroundStyle(Theme.textPrimary)
                                .listRowBackground(Theme.surface)
                                .listRowSeparatorTint(Theme.borderDivider)
                        }
                    }
                }
            }

            // Steps
            if !r.steps.isEmpty {
                Section("Steps") {
                    ForEach(Array(r.steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.system(size: 13, weight: .bold))
                                .foregroundStyle(Theme.accent)
                                .frame(width: 22, alignment: .trailing)
                            Text(step)
                                .font(.system(size: 14))
                                .foregroundStyle(Theme.textPrimary)
                        }
                        .listRowBackground(Theme.surface)
                        .listRowSeparatorTint(Theme.borderDivider)
                    }
                }
            }
        }
        .listStyle(.insetGrouped)
        .scrollContentBackground(.hidden)
    }

    private func metaStat(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.system(size: 18, weight: .bold))
                .foregroundStyle(Theme.textPrimary)
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(Theme.textMuted)
        }
        .frame(maxWidth: .infinity)
    }
}
```

> Note: `RecipeDetailViewModel` is the existing `RecipesViewModel`-adjacent VM. The existing `RecipeDetailView` takes `recipe: Recipe` and `apiClient:`. Keep using the existing `RecipeDetailViewModel` class (whatever it's named in `RecipeDetailView.swift` — adapt if needed).

- [ ] **Step 2: Build**

  ⌘B. Expected: Build Succeeded. Fix any ViewModel type mismatches from the existing `RecipeDetailView` implementation.

- [ ] **Step 3: Verify Cook Mode**

  Open a recipe. Tap Cook Mode — screen should not sleep. Navigate back — sleep timer should restore. Verify by waiting 30 seconds with Cook Mode active (no sleep) vs inactive (sleeps normally).

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Recipes/RecipeDetailView.swift
git commit -m "feat(ios): redesign RecipeDetailView with dark theme and scoped Cook Mode"
```

---

### Task 15: CalendarView redesign

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Calendar/CalendarView.swift`

- [ ] **Step 1: Replace CalendarView.swift**

  Keep existing ViewModel integration and `daysInMonth` logic. Replace visuals:

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
            ZStack {
                Theme.background.ignoresSafeArea()
                ScrollView {
                    VStack(spacing: 0) {
                        if case .failed(let error) = viewModel.state {
                            Text(error.localizedDescription)
                                .font(.system(size: 13))
                                .foregroundStyle(Theme.statusRed)
                                .padding(.horizontal, 14)
                                .padding(.vertical, 8)
                        }
                        calendarGrid
                            .padding(.horizontal, 14)
                        Rectangle().fill(Theme.borderDivider).frame(height: 1)
                        agendaSection
                    }
                }
                .refreshable { await viewModel.load() }
            }
            .navigationTitle(Self.monthFormatter.string(from: viewModel.currentMonth))
            .navigationBarTitleDisplayMode(.inline)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button { viewModel.previousMonth() } label: {
                        Image(systemName: "chevron.left").foregroundStyle(Theme.accent)
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button { viewModel.nextMonth() } label: {
                        Image(systemName: "chevron.right").foregroundStyle(Theme.accent)
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
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(Theme.textMuted)
                        .frame(maxWidth: .infinity)
                }
            }
            LazyVGrid(columns: columns, spacing: 4) {
                ForEach(Array(daysInMonth.enumerated()), id: \.offset) { _, day in
                    if let day {
                        DayCell(
                            date: day,
                            isSelected: Calendar.current.isDate(day, inSameDayAs: viewModel.selectedDay ?? .distantPast),
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
                    .frame(maxHeight: .infinity)
                } else {
                    List(chores) { chore in
                        HStack(spacing: 10) {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(chore.name)
                                    .font(.system(size: 15, weight: .medium))
                                    .foregroundStyle(Theme.textPrimary)
                                if let badge = chore.badgeVariant {
                                    StatusBadge(variant: badge)
                                }
                            }
                            Spacer()
                        }
                        .padding(.vertical, 4)
                        .listRowBackground(Theme.surface)
                        .listRowSeparatorTint(Theme.borderDivider)
                    }
                    .listStyle(.plain)
                    .scrollContentBackground(.hidden)
                }
            } else {
                ContentUnavailableView("Select a day", systemImage: "calendar")
                    .frame(maxHeight: .infinity)
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
    let hasChores: Bool

    private static let dayFormatter: DateFormatter = {
        let f = DateFormatter(); f.dateFormat = "d"; f.locale = Locale(identifier: "en_US_POSIX"); return f
    }()

    var body: some View {
        VStack(spacing: 2) {
            Text(Self.dayFormatter.string(from: date))
                .font(.system(size: 14))
                .foregroundStyle(isSelected ? .white : Theme.textPrimary)
                .frame(width: 30, height: 30)
                .background(isSelected ? Theme.accent : Color.clear)
                .clipShape(Circle())
            Circle()
                .fill(hasChores ? Theme.statusAmber : Color.clear)
                .frame(width: 4, height: 4)
        }
    }
}
```

- [ ] **Step 2: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 3: Verify Calendar**

  Run app → Calendar tab. Selected day highlighted in blue. Days with chores show amber dot. Agenda shows chores for selected day with `StatusBadge`.

- [ ] **Step 4: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/Calendar/CalendarView.swift
git commit -m "feat(ios): redesign CalendarView with dark theme, amber chore dots, pull-to-refresh"
```

---

### Task 16: ContentView — update tab labels and background

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/ContentView.swift`

- [ ] **Step 1: Replace ContentView.swift**

```swift
import SwiftUI

struct ContentView: View {
    let apiClient: any APIClientProtocol

    var body: some View {
        TabView {
            DashboardView(apiClient: apiClient)
                .tabItem { Label("Home", systemImage: "house.fill") }

            ChoresView(apiClient: apiClient)
                .tabItem { Label("Chores", systemImage: "checkmark.circle") }

            MealsView(apiClient: apiClient)
                .tabItem { Label("Meals", systemImage: "fork.knife") }

            RecipesView(apiClient: apiClient)
                .tabItem { Label("Recipes", systemImage: "book.closed") }

            CalendarView(apiClient: apiClient)
                .tabItem { Label("Calendar", systemImage: "calendar") }
        }
        .tint(Theme.accent)
    }
}
```

- [ ] **Step 2: Build**

  ⌘B. Expected: Build Succeeded.

- [ ] **Step 3: Run full verification**

  ⌘R on iPhone 15 Simulator. Walk through the verification checklist from the spec:
  1. Dashboard: stat cards, chore rows with avatar + assignee + badge + Done button, Today's Meals, Leaderboard placeholder
  2. Chores: Pending/Completed segment; Overdue and Due Soon grouping; tap row → detail; Mark Complete → pops back
  3. Recipes: search filters by title; pull-to-refresh; Cook Mode toolbar button
  4. Calendar: day selection updates agenda; amber dot on days with chores
  5. Pull-to-refresh on all 5 tabs
  6. Network offline → Dashboard error text appears (no alert)
  7. Tap avatar → Profile sheet; Sign Out → LoginView

- [ ] **Step 4: Run all tests**

  ⌘U. Expected: all test targets pass.

- [ ] **Step 5: Commit**

```bash
git add ios/FamilyHub/FamilyHub/Features/ContentView.swift
git commit -m "feat(ios): update ContentView tab labels and accent colour"
```

---

## End State

All 16 tasks produce:
- A complete `DesignSystem/Theme.swift` and `Components/` directory with 4 shared components
- A `User` model with `fetchUsers()` wired through the full API layer
- All 5 tabs redesigned to the dark-navy Suskins Hub aesthetic
- Pull-to-refresh on every tab (Refresh toolbar buttons removed)
- Inline error display on every screen (no `.alert` for load errors)
- `ProfileView` sheet with functional Sign Out
- `ChoresView` grouped into Overdue and Due Soon sections
- `RecipesView` with client-side search
- Cook Mode scoped to `RecipeDetailView` lifetime
