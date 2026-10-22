# Commute card

[Open the interactive proposal](index.html).
[Read the feature guide](../../../commute.md).

The native feature uses the Warm Sage palette and card style from `DESIGN.md`.
Delayed departures and arrivals show the scheduled time with a strikethrough before the expected time.
Today opens Settings from the user's profile photo, with initials as the fallback.
The other tabs do not show this button.

## Native captures

These captures use demo data on an iPhone simulator.
They do not show live train services.

- [Small iPhone: Today](native/iphone-small-today.png)
- [Small iPhone: train card](native/iphone-small-trains.png)
- [Small iPhone: settings](native/iphone-small-settings.png)
- [Larger iPhone: dark train card](native/iphone-dark-trains.png)
- [Larger iPhone: dark settings](native/iphone-dark-settings.png)
- [Larger iPhone: maximum text size](native/iphone-large-text.png)
- [Larger iPhone: maximum text size in Settings](native/iphone-large-settings.png)

## Validation

- The shared package suite reports 129 tests, with two optional tests skipped and no failures.
- The app unit suite reports 197 tests, with 44 abstract or live-contract tests skipped and no failures.
- The commute UI checks pass on an iPhone SE and an iPhone 17 Pro. Captures cover light, dark, and maximum text size.
- The Watch simulator build passes.
- The live train API test requires an explicit URL and token. It does not run in this session.
- Physical device behaviour and a manual VoiceOver check remain unverified.
