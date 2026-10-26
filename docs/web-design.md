# Family Hub web design

The web app extends the Warm Sage Cosier direction used by the native client.
Use warm paper backgrounds, quiet green accents, restrained clay and gold
highlights, readable system typography, and familiar rounded surfaces. Keep
information density low enough to scan at a glance.

## Responsive layout

Use a persistent left navigation rail on wide screens and a fixed bottom
navigation bar on phones. Keep the primary action near the content it changes.
The wall display uses larger type and generous spacing for across-room reading.
Make each single-action overview card fully clickable. Give cards that contain
multiple controls separate, independently operable controls instead.

## Interaction and access

Use native links for navigation and buttons for actions. Provide visible focus
states, keyboard access, semantic headings, and at least 44 px touch targets
for primary touch controls. Pair color with text or an icon. Respect reduced
motion and preserve the last visible display data while the network is
unavailable.

## Color tokens

Use the native Warm Sage palette: paper `#f7f1e7`, surface `#fffcf6`, ink
`#343c31`, secondary text `#646357`, accent `#3c654c`, border `#e6dfd2`, sage
`#e6eddd`, clay `#8b503c`, and destructive `#9d3c34`. Keep text contrast at
WCAG AA levels. Add dark mode only with a complete token set; do not mix ad hoc
light and dark values in a component.

## Content states

Every data screen needs loading, empty, stale/offline, saving, and failure
states. Keep local edits visible until the server confirms them. Use calm,
actionable copy when a save fails.

## Wall display

Let the household choose which overview sections appear on each display. Keep
the app launcher available when a section is hidden. Save this preference on
the display so one screen's layout does not change another screen. Provide a
clear settings control, visible focus, and a useful empty state when all
sections are hidden.
