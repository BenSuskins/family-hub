import AppIntents
import Foundation
import FamilyHubKit

struct ReadMyChoresIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Read My Chores" }
    nonisolated static var description: IntentDescription { "Read your unfinished chores due today or overdue." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog & ReturnsValue<[String]> {
        let chores = try await SiriEnvironment.service().myChores()
        let lines = chores.map(\.name)
        let spoken = SiriNames.summary(lines, empty: "You have no chores due today or overdue.")
        return .result(value: lines, dialog: "\(spoken)")
    }
}

struct CompleteNamedChoreIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Complete One Chore" }
    nonisolated static var description: IntentDescription { "Complete one of your chores due today or overdue." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }
    @Parameter(title: "Chore", requestValueDialog: "Which chore did you complete?")
    var chore: ChoreEntity

    static var parameterSummary: some ParameterSummary { Summary("Complete \(\.$chore)") }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog {
        let service = try SiriEnvironment.service()
        try await SiriEnvironment.validate(scope: chore.scope, service: service)
        defer { SiriEnvironment.changed() }
        try await service.completeChore(id: chore.rawID)
        return .result(dialog: "Completed \(chore.title).")
    }
}

struct CompleteMyChoresIntent: AppIntent {
    nonisolated static var title: LocalizedStringResource { "Complete My Chores" }
    nonisolated static var description: IntentDescription { "Complete all your unfinished chores due today or overdue, without confirmation." }
    nonisolated static var authenticationPolicy: IntentAuthenticationPolicy { .alwaysAllowed }

    @MainActor
    func perform() async throws -> some IntentResult & ProvidesDialog & ReturnsValue<String> {
        defer { SiriEnvironment.changed() }
        let report = try await SiriEnvironment.service().completeMyChores()
        var answer = report.completed.isEmpty && report.failed.isEmpty && report.uncertain.isEmpty
            ? "You have no chores due today or overdue." : "Completed \(report.completed.count) chores."
        if !report.failed.isEmpty { answer += " Could not complete: \(report.failed.joined(separator: ", "))." }
        if !report.uncertain.isEmpty { answer += " The server did not confirm: \(report.uncertain.joined(separator: ", ")). Check Family Hub before you repeat the request." }
        return .result(value: answer, dialog: "\(answer)")
    }
}
