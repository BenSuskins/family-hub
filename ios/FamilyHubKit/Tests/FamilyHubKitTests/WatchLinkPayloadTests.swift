import XCTest
@testable import FamilyHubKit

final class WatchLinkPayloadTests: XCTestCase {

    func testCredentialsRoundTripThroughAWatchPayload() throws {
        let sent = WatchCredentials(apiToken: "tok-123", baseURL: "https://hub.example.com")

        let payload = try sent.watchPayload()
        let received = try WatchCredentials.fromWatchPayload(payload)

        XCTAssertEqual(received, sent)
    }

    func testCredentialsPayloadCarriesTheJSONAsData() throws {
        // WatchConnectivity accepts property-list types only, so the payload is
        // one `Data` value rather than a decomposed dictionary.
        let payload = try WatchCredentials(apiToken: "tok", baseURL: "https://x.test").watchPayload()

        XCTAssertEqual(payload.count, 1)
        let data = try XCTUnwrap(payload.values.first as? Data)
        let json = try XCTUnwrap(String(data: data, encoding: .utf8))
        XCTAssertTrue(json.contains("\"apiToken\":\"tok\""))
        XCTAssertTrue(isWatchTransportable(payload))
    }

    // The regression this file exists for. `stepDurations` is `[Int?]`, so a
    // step with no recognisable timing encoded as JSON `null`, which became
    // `NSNull` once the payload was decomposed. `NSNull` is not a property-list
    // type, so WatchConnectivity silently dropped the transfer and the watch sat
    // on its empty screen while the phone reported the recipe sent.
    func testHandoffWithUntimedStepsIsStillTransportable() throws {
        let recipe = Recipe(
            id: "r1",
            title: "Soup",
            steps: ["Chop the onions", "Simmer for 20 minutes", "Season"],
            ingredients: [IngredientGroup(name: "", items: ["2 onions"])],
            stepDurations: [nil, 1200, nil]
        )

        let payload = try CookHandoff(recipe: recipe).watchPayload()

        XCTAssertTrue(isWatchTransportable(payload))
        let received = try CookHandoff.fromWatchPayload(payload)
        XCTAssertEqual(received.recipe.stepDurations?.count, 3)
        XCTAssertEqual(received.recipe.stepDurations?[1], 1200)
    }

    func testATransportabilityCheckActuallyRejectsNull() {
        // Guards the guard: a check that returned true for everything would have
        // let the original bug through.
        XCTAssertFalse(isWatchTransportable(["a": NSNull()]))
        XCTAssertFalse(isWatchTransportable(["a": [1, NSNull()]]))
        XCTAssertTrue(isWatchTransportable(["a": ["b": [1, 2]], "c": Data()]))
    }

    func testLegacyDecomposedPayloadsStillDecode() throws {
        // A watch updated ahead of its phone, or one replaying the application
        // context the previous build left behind.
        let legacy: [String: Any] = ["apiToken": "tok", "baseURL": "https://x.test"]

        let received = try WatchCredentials.fromWatchPayload(legacy)

        XCTAssertEqual(received, WatchCredentials(apiToken: "tok", baseURL: "https://x.test"))
    }

    func testHandoffCarriesEverythingCookModeNeeds() throws {
        let recipe = Recipe(
            id: "r1",
            title: "Soup",
            steps: ["Chop", "Simmer for 20 minutes"],
            ingredients: [IngredientGroup(name: "Main", items: ["2 onions"])],
            servings: 4,
            prepTime: "10 mins",
            cookTime: "20 mins",
            stepDurations: [nil, 1200]
        )

        let payload = try CookHandoff(recipe: recipe).watchPayload()
        let received = try CookHandoff.fromWatchPayload(payload)

        XCTAssertEqual(received.recipe.title, "Soup")
        XCTAssertEqual(received.recipe.steps, ["Chop", "Simmer for 20 minutes"])
        XCTAssertEqual(received.recipe.ingredients?.first?.items, ["2 onions"])
        XCTAssertEqual(received.recipe.stepDurations?[1], 1200)
    }

    func testHandoffPreservesSentAtSoTheNewestWins() throws {
        let earlier = Date(timeIntervalSince1970: 1_000_000)
        let handoff = CookHandoff(recipe: Recipe(id: "r1", title: "Soup"), sentAt: earlier)

        let received = try CookHandoff.fromWatchPayload(try handoff.watchPayload())

        XCTAssertEqual(received.sentAt.timeIntervalSince1970, earlier.timeIntervalSince1970, accuracy: 1)
    }

    func testRecipeDecodesServerStepDurations() throws {
        // Capitalised keys, matching the Go API's untagged struct field names.
        let json = """
        {"ID":"r1","Title":"Soup","Steps":["Chop","Simmer"],"Ingredients":null,
         "MealType":null,"Servings":null,"PrepTime":null,"CookTime":null,
         "SourceURL":null,"CategoryID":null,"HasImage":false,
         "StepDurations":[null,1200]}
        """.data(using: .utf8)!

        let recipe = try JSONDecoder().decode(Recipe.self, from: json)

        XCTAssertEqual(recipe.stepDurations?.count, 2)
        XCTAssertNil(recipe.stepDurations?[0] ?? nil)
        XCTAssertEqual(recipe.stepDurations?[1], 1200)
    }

    func testRecipeStillDecodesWithoutStepDurations() throws {
        // An older server, or the list endpoint, omits the key entirely.
        let json = """
        {"ID":"r1","Title":"Soup","Steps":null,"Ingredients":null,"MealType":null,
         "Servings":null,"PrepTime":null,"CookTime":null,"SourceURL":null,
         "CategoryID":null,"HasImage":false}
        """.data(using: .utf8)!

        let recipe = try JSONDecoder().decode(Recipe.self, from: json)

        XCTAssertNil(recipe.stepDurations)
    }
}

final class CookTimerTests: XCTestCase {

    func testRemainingCountsDownFromTheClock() {
        let start = Date(timeIntervalSince1970: 1_000_000)
        let timer = CookTimer(stepIndex: 1, duration: 600, startedAt: start)

        XCTAssertEqual(timer.remaining(at: start), 600)
        XCTAssertEqual(timer.remaining(at: start.addingTimeInterval(60)), 540)
    }

    func testRemainingIsFlooredAtZeroAndReportsFired() {
        let start = Date(timeIntervalSince1970: 1_000_000)
        let timer = CookTimer(stepIndex: 0, duration: 60, startedAt: start)
        let later = start.addingTimeInterval(120)

        XCTAssertEqual(timer.remaining(at: later), 0)
        XCTAssertTrue(timer.hasFired(at: later))
        XCTAssertFalse(timer.hasFired(at: start))
    }

    func testProgressIsClampedToZeroThroughOne() {
        let start = Date(timeIntervalSince1970: 1_000_000)
        let timer = CookTimer(stepIndex: 0, duration: 100, startedAt: start)

        XCTAssertEqual(timer.progress(at: start), 0, accuracy: 0.01)
        XCTAssertEqual(timer.progress(at: start.addingTimeInterval(50)), 0.5, accuracy: 0.01)
        XCTAssertEqual(timer.progress(at: start.addingTimeInterval(500)), 1, accuracy: 0.01)
    }

    func testNotificationIDIsPerStepSoARestartReplacesIt() {
        let first = CookTimer(stepIndex: 2, duration: 60)
        let restarted = CookTimer(stepIndex: 2, duration: 300)
        XCTAssertEqual(first.notificationID, restarted.notificationID)
        XCTAssertNotEqual(first.notificationID, CookTimer(stepIndex: 3, duration: 60).notificationID)
    }

    func testCountdownFormatting() {
        XCTAssertEqual(formatCountdown(0), "0:00")
        XCTAssertEqual(formatCountdown(59), "0:59")
        XCTAssertEqual(formatCountdown(600), "10:00")
        XCTAssertEqual(formatCountdown(3661), "1:01:01")
        XCTAssertEqual(formatCountdown(-5), "0:00")
    }

    func testDurationDescriptions() {
        XCTAssertEqual(describeDuration(30), "30 sec")
        XCTAssertEqual(describeDuration(60), "1 min")
        XCTAssertEqual(describeDuration(1200), "20 min")
        XCTAssertEqual(describeDuration(90), "1 min 30 sec")
        XCTAssertEqual(describeDuration(3600), "1 hr")
        XCTAssertEqual(describeDuration(5400), "1 hr 30 min")
    }
}
