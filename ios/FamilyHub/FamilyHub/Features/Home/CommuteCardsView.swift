import SwiftUI
import FamilyHubKit

struct CommuteCardsView: View {
    let model: CommuteViewModel
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    var body: some View {
        if !model.activeDirections.isEmpty {
            HomeSectionHeader(title: "Commute") { EmptyView() }
            VStack(spacing: 16) {
                ForEach(model.activeDirections, id: \.self) { direction in
                    VStack(alignment: .leading, spacing: 0) {
                        VStack(alignment: .leading, spacing: 6) {
                            Label(direction.title, systemImage: "tram.fill")
                                .font(.headline)
                                .foregroundStyle(AppTheme.accent)
                            Text(direction == .toWork ? "Home → Work" : "Work → Home")
                                .font(.subheadline)
                                .foregroundStyle(AppTheme.secondary)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        .padding(.vertical, 16)
                        Divider()
                        boardContent(direction)
                    }
                    .padding(.horizontal, 16)
                    .groupedCard()
                    .accessibilityIdentifier("commute-\(direction.rawValue)")
                }
            }
            .padding(.horizontal, 20)
        }
    }

    @ViewBuilder
    private func boardContent(_ direction: CommuteDirection) -> some View {
        switch model.states[direction] ?? .idle {
        case .idle, .loading:
            HStack(spacing: 12) {
                ProgressView()
                Text("Loading trains…").foregroundStyle(AppTheme.secondary)
            }.padding(.vertical, 24)
        case .failed:
            VStack(alignment: .leading, spacing: 12) {
                Text("Train times unavailable").font(.headline)
                Text("Try again to get current train times.")
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                Button("Try again") { Task { await model.retry(direction) } }
                    .buttonStyle(AppSecondaryButtonStyle())
            }.padding(.vertical, 20)
        case .loaded(let board):
            if board.trains.isEmpty {
                VStack(alignment: .leading, spacing: 6) {
                    Text("No upcoming trains").font(.headline)
                    Text("No services appear in the next two hours.")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                }.padding(.vertical, 20)
            } else {
                ForEach(Array(board.trains.prefix(3).enumerated()), id: \.element.id) { index, train in
                    if index > 0 { Divider() }
                    trainRow(train)
                }
            }
            if let fetched = board.fetchedDate {
                Text("Updated at \(fetched.formatted(Date.FormatStyle(date: .omitted, time: .shortened, timeZone: TimeZone(identifier: board.timezone) ?? .current)))")
                    .font(.caption)
                    .foregroundStyle(AppTheme.secondary)
                    .padding(.bottom, 14)
            }
        }
    }

    private func trainRow(_ train: CommuteTrain) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            ViewThatFits(in: .horizontal) {
                HStack(alignment: .firstTextBaseline, spacing: 12) {
                    departureTime(train)
                    Spacer(minLength: 0)
                    status(train)
                }
                VStack(alignment: .leading, spacing: 4) {
                    departureTime(train)
                    status(train)
                }
            }
            if train.isCancelled {
                Text(train.cancellationReason ?? "This service does not run.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
            } else {
                arrivalTime(train)
                Text(train.platform.map { "Platform \($0)" } ?? "Platform unconfirmed")
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                if let reason = train.delayReason, !reason.isEmpty {
                    Text(reason).font(.subheadline).foregroundStyle(AppTheme.secondary)
                }
            }
        }
        .fixedSize(horizontal: false, vertical: true)
        .padding(.vertical, 14)
        .accessibilityElement(children: .combine)
        .accessibilityIdentifier("commute-train-\(train.serviceID)")
    }

    private func departureTime(_ train: CommuteTrain) -> some View {
        let layout = dynamicTypeSize.isAccessibilitySize
            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 2))
            : AnyLayout(HStackLayout(alignment: .firstTextBaseline, spacing: 8))
        return layout {
            if let original = train.departure.original {
                Text(original).strikethrough().foregroundStyle(AppTheme.secondary)
            }
            Text(train.departure.display).strikethrough(train.isCancelled)
        }
        .font(.title2.weight(.semibold).monospacedDigit())
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(train.departure.accessibilityLabel("departure"))
    }

    private func arrivalTime(_ train: CommuteTrain) -> some View {
        let hasTime = train.arrival.scheduled != nil || train.arrival.expected != nil
        var text = AttributedString(train.arrival.expected == nil ? "Scheduled arrival " : "Arrives ")
        if let original = train.arrival.original {
            var scheduled = AttributedString(original)
            scheduled.strikethroughStyle = .single
            text += scheduled
            text += AttributedString(" ")
        }
        text += AttributedString(train.arrival.display)
        if !hasTime { text = AttributedString("Arrival time unavailable") }
        if let duration = train.durationMinutes { text += AttributedString(" · \(duration) min") }
        return Text(text)
            .font(.subheadline.monospacedDigit())
            .foregroundStyle(AppTheme.secondary)
            .accessibilityLabel(train.arrival.accessibilityLabel("arrival") + (train.durationMinutes.map { ", \($0) minutes" } ?? ""))
    }

    private func status(_ train: CommuteTrain) -> some View {
        Text(train.statusLabel)
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(train.isCancelled || train.status == "delayed" ? AppTheme.clay : train.status == "on_time" ? AppTheme.accent : AppTheme.secondary)
    }
}
