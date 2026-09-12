// swift-tools-version: 6.2
import PackageDescription

// Shared model and networking layer for the Family Hub clients.
//
// Deliberately Foundation-only: no SwiftUI, no UIKit. That is what lets the same
// code back the iOS app and a watchOS target, and what lets CI build and test it
// on a Linux runner in seconds rather than waiting on a macOS one.
//
// The Swift settings mirror the app target's build settings
// (SWIFT_DEFAULT_ACTOR_ISOLATION = MainActor, SWIFT_VERSION = 5.0) so isolation
// behaves identically on both sides of the module boundary.
let package = Package(
    name: "FamilyHubKit",
    platforms: [
        .iOS(.v26),
        .watchOS(.v26),
    ],
    products: [
        .library(name: "FamilyHubKit", targets: ["FamilyHubKit"]),
    ],
    targets: [
        .target(
            name: "FamilyHubKit",
            swiftSettings: [
                .swiftLanguageMode(.v5),
                .defaultIsolation(MainActor.self),
            ]
        ),
        // No .defaultIsolation here: it would make XCTestCase subclasses
        // MainActor-isolated, which cannot override XCTestCase's nonisolated
        // init(name:testClosure:). The library keeps it; the tests don't need it.
        .testTarget(
            name: "FamilyHubKitTests",
            dependencies: ["FamilyHubKit"],
            swiftSettings: [
                .swiftLanguageMode(.v5),
            ]
        ),
    ]
)
