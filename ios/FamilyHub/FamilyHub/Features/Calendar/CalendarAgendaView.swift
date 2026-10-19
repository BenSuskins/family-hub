import SwiftUI
import FamilyHubKit

struct CalendarDisplayItem: Identifiable {
    enum Content {
        case event(CalendarEvent)
        case chore(Chore)
        case meal(MealPlan)
    }

    let content: Content

    var id: String {
        switch content {
        case .event(let event): "event-\(event.id)"
        case .chore(let chore): "chore-\(chore.id)"
        case .meal(let meal): "meal-\(meal.id)"
        }
    }

    var title: String {
        switch content {
        case .event(let event): event.title
        case .chore(let chore): chore.name
        case .meal(let meal): meal.name
        }
    }

    var kind: String {
        switch content {
        case .event(let event): event.allDay ? "All day" : event.startTime.formatted(date: .omitted, time: .shortened)
        case .chore: "Chore"
        case .meal(let meal): meal.mealType.capitalized
        }
    }

    var symbol: String? {
        switch content {
        case .event: nil
        case .chore: "checkmark.circle"
        case .meal: "fork.knife"
        }
    }

    var color: Color {
        switch content {
        case .event(let event): CalendarPalette.color(event.color)
        case .chore: .orange
        case .meal: .green
        }
    }
}

enum CalendarPalette {
    static func color(_ value: String) -> Color {
        if let color = Color(hex: value) { return color }
        switch value.lowercased() {
        case "red": return .red
        case "orange": return .orange
        case "yellow": return .yellow
        case "green": return .green
        case "blue": return .blue
        case "purple": return .purple
        case "pink": return .pink
        case "teal": return .teal
        default: return .indigo
        }
    }
}

extension CalendarViewModel {
    func items(for day: Date, untimedOnly: Bool = false) -> [CalendarDisplayItem] {
        events(for: day).filter { !untimedOnly || $0.allDay }.map { CalendarDisplayItem(content: .event($0)) }
        + chores(for: day).map { CalendarDisplayItem(content: .chore($0)) }
        + meals(for: day).map { CalendarDisplayItem(content: .meal($0)) }
    }
}

struct CalendarAgendaView: View {
    let viewModel: CalendarViewModel
    let day: Date
    let apiClient: any APIClientProtocol
    var showDate = true

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if showDate {
                Text(day.formatted(.dateTime.weekday(.wide).day().month(.wide)))
                    .font(.headline).padding(.vertical, 12)
                    .accessibilityAddTraits(.isHeader)
            }
            if !viewModel.hasLoaded(day) {
                Text(viewModel.loadError == nil ? "Loading items…" : "Items unavailable")
                    .foregroundStyle(.secondary).padding(.vertical, 16)
            } else if !viewModel.hasItems(for: day) {
                Text("No items").foregroundStyle(.secondary).padding(.vertical, 16)
            } else {
                ForEach(viewModel.items(for: day)) { item in
                    CalendarItemRow(item: item, apiClient: apiClient)
                    Divider()
                }
            }
        }.frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct CalendarItemRow: View {
    let item: CalendarDisplayItem
    let apiClient: any APIClientProtocol

    var body: some View {
        Group {
            if case .event(let event) = item.content {
                NavigationLink { EventDetailsView(event: event, apiClient: apiClient) } label: { row }
                    .buttonStyle(.plain)
            } else {
                row
            }
        }.accessibilityElement(children: .combine)
    }

    private var row: some View {
        HStack(alignment: .top, spacing: 12) {
            RoundedRectangle(cornerRadius: 2).fill(item.color).frame(width: 4)
            VStack(alignment: .leading, spacing: 4) {
                if let symbol = item.symbol {
                    Label(item.title, systemImage: symbol)
                } else { Text(item.title) }
                Text(item.kind).font(.subheadline).foregroundStyle(.secondary)
                if case .event(let event) = item.content, !event.location.isEmpty {
                    Label(event.location, systemImage: "mappin").font(.caption).foregroundStyle(.secondary)
                }
            }
            Spacer(minLength: 0)
            if case .event = item.content {
                Image(systemName: "chevron.right").font(.caption).foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 12)
        .frame(minHeight: 44)
        .contentShape(Rectangle())
    }
}
