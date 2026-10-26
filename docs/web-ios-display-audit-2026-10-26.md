# Family Hub web and iOS display audit

> Handover of prototype gaps, implementation defects, API dependencies, and release checks.

## Audit baseline and scope

- Date: 26 October 2026, Europe/London.
- Source baseline: `a02f029` on local `main`.
- Regression commit: `a7b3c3296c3c826ea6dd40c706048efc5c04d06c`, included through PR #112.
- Reference: [interactive prototype](prototypes/wall-display.html).
- Production URL: <https://family-display.suskins.co.uk/display>.
- Evidence: current source, Git history, the user's deployed overview and logo screenshots, and an independent browser visit that reached the live pairing page.
- This audit changes documentation only. It does not repair, merge, deploy, or release application code.

This document covers the complete wall-display experience represented by the prototype and the iOS flows that pair and configure it. It also covers API gaps that block those flows. It is not a full defect audit of unrelated iOS features, widgets, Watch, or the share extension.

The deployed overview was not paired in the audit browser. Its exact image revision, API failures, and real-device behavior remain unverified. Source findings apply to the baseline above. The supplied screenshots confirm the visible frame, sample household name, inactive-looking launcher, empty cards, connection banner, and placeholder logo; source inspection establishes why those elements behave as reported.

## Outcome

The current display does not meet the prototype's functional scope. It copies part of the visual layout but omits the adaptive card system and touch flows. The latest display-only rewrite also removes previously implemented display panels and actions.

Do not treat a successful build, a pairing screenshot, or a green server test as proof that the paired display works. Earlier Workboard completion claims do not describe this baseline. The preceding code had more working actions, but that does not prove it met the whole prototype either.

### Priority and evidence terms

- **P1:** Blocks a core flow or shows materially misleading data.
- **P2:** Significant design, usability, or resilience gap.
- **P3:** Supporting polish or documentation gap.
- **Confirmed:** Directly established by source or supplied screenshot.
- **Contract gap:** Required user action has no corresponding display API route.
- **Decision:** Sources disagree or the production rule needs to be explicit.
- **Verify:** Needs an execution check; do not present it as a reproduced defect.

## Source map

Paths below are repository-relative. Use the baseline commit when reviewing this handover.

| Ref | Source | Relevant code |
|---|---|---|
| PROTO | [Prototype](prototypes/wall-display.html) | `rankedSections`, `departure`, `card`, `renderModal`, click handlers, idle timer |
| WEB | [Display client](../web/src/components/display-client.tsx) | `PairDisplay`, `DisplayClient`, `refresh`, overview markup |
| CSS | [Web styles](../web/src/app/globals.css) | `.wall-display-shell`, `.wall-display`, grids, pairing mark, responsive rules |
| IOS | [Display management](../ios/FamilyHub/FamilyHub/Features/Settings/DisplayManagementView.swift) | Model, settings view, QR scanner |
| PROFILE | [Profile](../ios/FamilyHub/FamilyHub/Features/Profile/ProfileView.swift) | Admin entry to Wall displays |
| COMMUTE | [Commute settings](../ios/FamilyHub/FamilyHub/Features/Settings/CommuteSettingsView.swift) | `load`, `save`, direction schedules |
| STORE | [Commute preferences](../ios/FamilyHubKit/Sources/FamilyHubKit/CommutePreferences.swift) | `CommutePreferencesStore`, server/user keys, `UserDefaults` |
| PAIRMODEL | [Paired display model](../ios/FamilyHubKit/Sources/FamilyHubKit/PairedDisplay.swift) | Device name, pairing owner, enabled sections |
| ROUTES | [Server routes](../server/internal/server/server.go) | Pairing, admin device routes, display route allowlist |
| PAIRAPI | [Pairing handler](../server/internal/handlers/display_pairing.go) | Challenge, approval, claim, settings, revocation |
| AUTH | [Display auth](../server/internal/middleware/display_auth.go) | Scope checks, creator resolution |
| CALAPI | [Calendar handler](../server/internal/handlers/api.go) | `ListCalendar`, `SaveMeal`, `GetRecipe` |
| HOMEAPI | [Household handler](../server/internal/handlers/household.go) | Weather/departure status handling |
| BINS | [Bin handler](../server/internal/handlers/bin_collections.go) | `GetForDisplay` |
| INVENTORY | [Inventory handler](../server/internal/handlers/api_inventory.go) | Area reads and stock writes |
| NOTES | [Scratchpad handler](../server/internal/handlers/scratchpad.go) | Drawing reads/writes and validation |
| DESIGN | [Design guide](DESIGN.md) | Product boundary, timing, interactions, tokens |
| ENDPOINTS | [Endpoint reference](endpoints.md) | Current route contracts |
| LOGO | [Existing Family Hub artwork](../ios/FamilyHub/FamilyHub/Assets.xcassets/AppIcon.appiconset/AppIcon.png) | House illustration; dark/tinted variants also exist |
| CI | [Web CI](../.github/workflows/web.yml), [package scripts](../web/package.json) | Lint/build only; no web test script |

## 1. Branding, layout, and automatic card selection

| ID | Priority / evidence | Gap and impact | Required result / acceptance check |
|---|---|---|---|
| W01 | P2 Confirmed | WEB hardcodes `Willow household`. It is prototype sample copy, not household data. | Remove the fictitious identity. Use a verified configured identity if one is available, otherwise omit it. No sample household names appear in production. |
| W02 | P2 Confirmed | Pairing uses a literal italic `f` styled in CSS. It does not load LOGO. The prototype pairing screen itself uses a home symbol; the production `f` is not a faithful asset transfer. | Reuse existing Family Hub artwork. Check pairing, loading, browser icon, and light/dark presentation for consistent branding. |
| W03 | P1 Confirmed | CSS adds outer padding, a thick border, rounded device corners, shadow, and a 1920×1080 cap. The frame belongs to a prototype review presentation, not the installed display surface. | Fill the browser viewport at target wall resolutions. Remove simulated hardware framing. Separately verify kiosk/browser fullscreen; CSS cannot hide browser chrome. |
| W04 | P1 Confirmed | WEB uses fixed primary/secondary grids. There is no equivalent of PROTO `rankedSections`. | Rank useful cards by current context. Promote imminent departures and relevant meals. Reflow when cards appear or disappear. Verify morning/day/evening and no-content scenarios. |
| W05 | P1 Confirmed | Enabled sections render even without useful data: journeys, bins, calendar, chores, and meals retain empty content. | Hide empty overview cards and reclaim their space. Keep useful empty/setup feedback inside an opened feature. Do not confuse unknown or failed data with a confirmed empty result. This follows the user's latest instruction; the prototype alone is not a complete empty-data specification. |
| W06 | P2 Confirmed | Unicode glyphs replace the prototype SVG icon set. Some appear as unrelated symbols. | Use a consistent, accessible icon set. Compare icons and type hierarchy with the prototype on the actual display. |
| W07 | P2 Verify | Fixed card heights, a capped display height, and `overflow:hidden` can clip content at short heights or browser zoom. Only width ≤760px receives the scrolling layout. | Check 16:9, 16:10, portrait, short landscape, 200% zoom, long titles, and large lists. Preserve access to every control. |
| W08 | P2 Confirmed | Greeting depends on whether a departure exists. A morning departure can produce “Home for the day.” | Use coherent time/context rules. Verify text before departure, after departure, and in the evening. |

## 2. Touch features and detail screens

Every destination must work within the shared display. Removing personal member web pages does not authorize removing shared display actions.

| ID | Priority / evidence | Gap and impact | Required result / acceptance check |
|---|---|---|---|
| W09 | P1 Confirmed | All six launcher destinations are `span` elements. There is no navigation handler or detail screen. | Shopping, Notes, Calendar, Recipes, Meal plan, and Inventory open working display views. Support touch, keyboard, and a clear return to overview. |
| W10 | P1 Confirmed | “Edit meals” is a `div`; meal rows have no actions. | Open the editor at the selected date and slot. Save, reload, and verify persistence from both web and iOS. Show saving/failure feedback. |
| W11 | P1 Confirmed | Chore circles are decorative. No completion or person-selection flow exists. | Complete assigned chores with correct attribution. Ask who completed an unassigned chore, as in PROTO. Avoid attributing all actions to the person who paired the display. Requires A01. |
| W12 | P1 Confirmed | Calendar rows do not open event details. The weekly view and previous/next/Today controls are absent. | Restore the weekly timeline and navigation. Open event details with date, time, location, and relevant departure context. |
| W13 | P1 Confirmed | Weather is static markup. Hourly and seven-day details are absent. | Open the hourly strip and seven-day forecast. Show real conditions, rain probability, and unavailable states. |
| W14 | P1 Confirmed | Shopping list, add, purchase, and Undo flows are absent. | Add items, mark purchased, offer Undo, and remove purchased rows after the intended delay. Persist and verify across clients. Adding needs A02. |
| W15 | P1 Confirmed | Notes drawing UI and `kiosk-scratchpad.tsx` were removed. | Restore drawing-only notes, color choice, undo, clear, and persistence. Verify finger/stylus input and failure recovery against NOTES. Do not substitute a text-only note form. |
| W16 | P1 Confirmed | Recipe browsing and ingredient/method views are absent. | Open recipes from the launcher and linked meals. Preserve recipe IDs; the current WEB meal type contains only date/type/name. |
| W17 | P1 Confirmed | Inventory destination and stock controls are absent. | Browse areas and items. Apply supported stock changes with persistence and failure feedback. Stock writes need A03. |
| W18 | P2 Confirmed | The prototype's detail-dialog behavior and five-minute idle return are absent along with detail views. | Restore explicit close/back, Escape, focus restoration, modal focus handling, and idle return. Do not discard an active save or drawing without a defined rule. |

## 3. Data selection and truthfulness

| ID | Priority / evidence | Gap and impact | Required result / acceptance check |
|---|---|---|---|
| W19 | P1 Confirmed | Weather always uses a sun glyph and “Mostly sunny” when `current` exists. Weather `code` and response `status` are ignored. The server returns a value-type `current` even in setup-required responses. | Derive condition text/icon from the code. Respect `setup_required` and unavailable statuses. Do not present zero-value weather as a valid forecast. |
| W20 | P1 Confirmed | Any next collection is labeled “Bins out tonight”, even several days away. Only one collection survives `.slice(0,1)`. | Show the card when action is relevant. Label the actual collection day. Group all relevant same-day bin types. Monday must not tell the user that a Wednesday collection goes out tonight. |
| W21 | P1 Confirmed | Departure headline shows `startTime`, not `leaveAt`. Selection uses event start rather than the prototype's near-leave window, then falls back to the first record. Past departures can remain with a clamped zero countdown. | Show leave time prominently, event start separately, and only relevant departures. Verify before/at/after leave time and after event start. Respect API status. |
| W22 | P2 Confirmed | Calendar overview includes all today's events without filtering ended events. The web model omits end time and all-day state. It cannot reproduce the prototype's upcoming/next-day behavior correctly. | Carry the required fields. Show ongoing/upcoming events under a defined policy; move to the next useful day when appropriate. Verify all-day and overnight events. |
| W23 | P2 Confirmed | Chores come from a week calendar request. The client filters status only, takes the first three, and labels that truncated count “left”. It does not select today's due work or show actual assignee names. | Select/sort relevant due and overdue work. Show accurate total and names. Verify more than three chores and chores outside today's date. |
| W24 | P1 Confirmed | After 18:00 the client selects tomorrow's meals but still labels the card “Today's meals”. The week request cannot supply next Monday when today is Sunday. | Label tomorrow correctly. Fetch a range that includes the selected meal date. Test Sunday evening, month/year change, and household timezone boundaries. |
| W25 | P2 Decision | PROTO switches tomorrow's meals at 20:00 and uses different meal cutoffs; DESIGN and WEB use 18:00, 11:00, and 16:00. | Choose one rule and update code, prototype, design guide, and tests together. Do not silently select whichever source is convenient. |

## 4. Refresh, offline behavior, and pairing

| ID | Priority / evidence | Gap and impact | Required result / acceptance check |
|---|---|---|---|
| W26 | P1 Confirmed | Any failed request sets a global “Connection lost” banner. Other requests can succeed. All six requests run even when their sections are disabled. | Report partial failures accurately. A weather/departure provider failure must not imply that the whole API is disconnected. Fetch only what is needed or handle disabled sections without global failure. Identify the failed endpoint during diagnosis. |
| W27 | P1 Confirmed | `updatedAt` advances on every refresh, including total failure. A first failed fetch creates an empty snapshot that later looks like saved information. | Track last successful data time per resource. Do not call fabricated empty defaults saved household data. Retain real cached content and expose its age. |
| W28 | P2 Confirmed | Fetch has no timeout/abort. `Promise.allSettled` waits for every request before rendering refreshed data. Refreshes can overlap. | Bound requests and prevent older responses from replacing newer state. One stalled provider must not freeze unrelated cards. Test a request that never resolves. |
| W29 | P2 Confirmed | Token and snapshot use fixed origin-wide local-storage keys. Cached JSON is cast without schema validation. Storage operations have no consistent failure handling. | Bind cached data to the paired device/API and version its schema. Clear incompatible data. Test corrupt cache, old cache, changed pairing/API, and unavailable browser storage. |
| W30 | P2 Confirmed | A failed claim sets a pairing error that later normal 409 waiting responses do not clear. In-flight claim results are not checked against the latest challenge. | Recover from transient errors without a misleading persistent warning. Ignore results for an expired/replaced challenge. Test renewal while a claim is pending. |
| W31 | P2 Decision / Verify | PROTO shows scanned/waiting approval and rejection states. The live API exposes create/approve/claim; the web cannot distinguish ordinary waiting from a scan. It uses five minutes, while the prototype simulates two. | Define truthful production states and one lifetime. Do not pretend to detect scanning without server evidence. Verify expiration, refusal/cancel, duplicate claim, offline retry, and revocation end to end. |

## 5. iOS display management and shared commute

| ID | Priority / evidence | Gap and impact | Required result / acceptance check |
|---|---|---|---|
| I01 | P1 Confirmed | iOS offers a saved Trains toggle; WEB never requests or renders trains. No display-scoped train route is registered. | Deliver the train capability across settings, API, and display. Until then, do not claim the toggle enables a working feature. |
| I02 | P1 Confirmed | The display's Commute route link opens personal COMMUTE settings. STORE writes only local `UserDefaults`, keyed by server/user. No route or schedule is sent to the paired screen. | Store display/household commute configuration server-side. Keep personal iOS commute settings distinct. Verify that changes on one phone reach the screen and another administrator's phone. |
| I03 | P2 Confirmed | The route summary reads preferences for `pairedByUserID`, but the destination settings screen loads/saves the current signed-in user. Another admin can view one person's summary and edit another person's preferences. | Read and write the same explicit shared configuration. Test two administrators and a display paired by the other admin. |
| I04 | P1 Confirmed | iOS says the display brings useful information forward, but WEB has no relevance engine. Personal commute direction/time toggles do not provide shared display automation. | Let availability settings control what may appear. Let the display automatically choose timing and direction from agreed shared rules. Automatic timing still needs a configured route; do not infer a destination without data. |
| I05 | P2 Confirmed | `message` initially says “Settings saved to this display” before any action or confirmation. Successful PATCH confirms server persistence, not that the display fetched it. | Use neutral initial copy. Confirm server save accurately; show applied/last-seen status only if the server can substantiate it. |
| I06 | P2 Confirmed | Disconnect awaits `model.remove`, whose errors are swallowed into a message, then always dismisses the settings screen. | Keep the screen open on failure and show a retry. Dismiss only after confirmed revocation. Disable duplicate submission while pending. |
| I07 | P2 Confirmed | Scanner fallback shows only an unavailable-camera message. There is no explicit close/retry/manual alternative. A start-scanning error returns an empty string, then appears as a different-server error. | Distinguish invalid QR, wrong server, camera denied/unavailable, and scanner failure. Provide clear exit/retry and an agreed fallback pairing path. Verify supported and unsupported hardware. |
| I08 | P2 Confirmed | iOS accepts any nonempty display name. Server validation allows 1–60 bytes. Rejection is reported generically as expired/could-not-approve. | Match validation and error messages. Test long names, whitespace, and multibyte names. Keep entered names on recoverable failure. |
| I09 | P2 Confirmed | The web setup says Settings → Displays → Scan QR code; the actual native screen is Wall displays and its action is Scan a display code, reached through the profile/settings flow. | Use the same screen/action names and real path in both clients. Confirm the flow with a person who has not used setup before. |
| I10 | P2 Decision | The prototype includes a named device and configuration flows, but production has no rename API/UI, appearance setting, or shared commute configuration model. | Decide which device properties admins can edit. If rename/appearance controls are required, persist them per display and test reload. Do not restore configuration screens in the kiosk contrary to the current product boundary. |

### Commute correction

The prototype does contain section toggles and a route control. Their existence is not itself a design deviation. The intended distinction is between **what may appear** and **when it should appear**. The current implementation saves some availability choices but neither shares route settings nor implements the automatic train card. The prototype's fixed sample windows also do not prove a complete production automation policy, including weekdays, holidays, multiple commuters, and direction changes.

## 6. API and ownership gaps

The display token must remain limited. Do not fix missing display actions by giving it a member token or allowing all member routes.

| ID | Priority / evidence | Gap | Required contract |
|---|---|---|---|
| A01 | P1 Contract gap | ROUTES has no display-scoped chore completion or household-person selection endpoint. Member completion exists separately. | Add a narrow action with explicit person attribution and suitable validation. Prove a display cannot complete work as an arbitrary external user. |
| A02 | P1 Contract gap | Display shopping supports GET and PATCH for an existing item, but no add-item route. | Add the narrow create contract needed by PROTO. Preserve version/conflict handling and Undo semantics. |
| A03 | P1 Contract gap | Display inventory supports GET only. Member stock writes exist separately. | Add only the stock operations required by the shared display. Test quantity/level rules and persistence. |
| A04 | P1 Contract gap | No shared display commute configuration or display-scoped train query exists. | Define route, direction, relevance windows, timezone, and ownership. Support useful unavailable/empty/stale train states. |
| A05 | P2 Confirmed | Display settings returns only enabled sections. Home info contains location/timezone/margin, not household identity. Device model contains a device name, which is not a household name. | Omit identity or define its owning server field. Never use a personal name or device name as an implicit household label. |
| A06 | P1 Confirmed | `ListCalendar` logs iCal fetch and meal repository failures but returns HTTP 200 with empty arrays for those parts. WEB cannot distinguish no events/meals from failure. | Return resource availability metadata or a suitable error contract. Preserve valid cached data. Test provider and repository failures. |
| A07 | P2 Verify | Calendar week/day bounds use UTC, while WEB selects the day in household timezone. Meals preview can also cross the fetched week. | Verify BST/DST, near-midnight events, and non-UK zones. Align the range contract with household-local dates. The Sunday meal gap is already confirmed as W24. |
| A08 | P2 Decision | PAIRAPI approval requires an authenticated member via `RequireUser`, while list/update/revoke are admin-only. DESIGN describes admin approval. The UI admin entry does not enforce server authorization. | Decide whether members may approve screens. If approval is admin-only, enforce it on the server and add a member-denial test. This is a policy mismatch, not proof of an unauthenticated bypass. |
| A09 | P2 Confirmed / Decision | AUTH resolves the device token's creator on each request. Deleting the pairing account invalidates device access. Shared writes can inherit the creator identity unless the action defines another actor. | Define device lifetime and attribution independently from personal sign-in where required. Test account deletion/role changes and explicit action attribution. |

Existing support worth preserving: real random single-use pairing secrets, hashed credentials, display token scope, revocation, per-display enabled sections, meal save, shopping update, scratchpad read/write, recipe read, calendar read, inventory read, weather, departures, and bins. Existing pairing route tests exercise scope and revocation. Their presence does not prove all UI paths work.

## 7. Source conflicts and release evidence

| ID | Priority / evidence | Gap | Handover action |
|---|---|---|---|
| V01 | P1 Confirmed | `a7b3c32` removes `KioskApp`, display detail views, meal editing, shopping updates, recipe detail loading, calendar navigation, forecast view, and the drawing component. | Inspect `git show a7b3c32 -- web/src/components/display-client.tsx`. Recover useful implementation selectively. Keep removed personal web navigation out of the kiosk. |
| V02 | P2 Confirmed | CI runs lint/build, with no web test command. It cannot detect inert launcher items or wrong card relevance. | Add behavior tests for selection/time rules and API-backed actions, plus browser checks for all launcher destinations. Use meaningful integration tests rather than markup snapshots alone. |
| V03 | P2 Confirmed | DESIGN differs from executable code on dark tokens and meal timing; the prototype is sample-data-only. | Reconcile the references before implementation. Prioritize the user's explicit fullscreen, branding, and empty-card corrections. Keep confirmed requirements distinct from demo mechanics. |
| V04 | P1 Verify | The screenshot reports connection loss, but the failing endpoint is unknown. The audit browser only reached pairing. | Capture redacted endpoint/status diagnostics from a paired display. Check API origin, CORS, response bodies, backend revision, and image revision before diagnosing the outage. Never publish tokens or QR secrets. |
| V05 | P1 Verify | No paired-device acceptance run occurred in this audit. No new application tests or CI runs were performed for this documentation change. | Record local tests, CI, deployment, and real-device results separately. Verify the deployed web/API pair and installed iOS build. |

## 8. Delivery sequence

1. Agree on conflicting rules: meal cutoffs, shared commute automation, approval roles, and editable device properties. Use the user's fullscreen and branding corrections as settled requirements.
2. Restore the real logo and remove fictitious household text and hardware framing.
3. Define shared server state and missing narrow action routes. Update the endpoint reference with every route change.
4. Implement pure relevance and date-selection functions. Cover empty data, partial failures, leave windows, bins, meal rollover, and commute direction.
5. Restore all display destinations and mutations. Preserve drawing-only Notes and person-aware chores.
6. Repair iOS shared settings, validation, scanner errors, and disconnect behavior.
7. Repair freshness/error reporting and cache lifecycle. Add observable applied state only if supported.
8. Run the acceptance checklist below. Deploy only after the web, API, and iOS contracts agree.

## 9. Acceptance checklist for the next developer

### Visual and automatic overview

- [ ] Display fills each target browser viewport without a simulated frame.
- [ ] Existing Family Hub artwork appears in setup; no placeholder `f` or Willow copy remains.
- [ ] Morning/day/evening screenshots match the agreed prototype behavior.
- [ ] No empty overview card consumes space; unavailable data is not represented as an empty success.
- [ ] Priority and layout change with imminent departures, bins, meals, and commute relevance.
- [ ] No clipped content at target sizes, long names, keyboard navigation, and zoom.

### Functional display

- [ ] Every launcher destination opens and returns to overview.
- [ ] Calendar previous/next/Today and event details work.
- [ ] Meal editing opens the correct date/slot and persists, including Sunday evening.
- [ ] Recipe links open ingredients and method.
- [ ] Assigned/unassigned chores save with correct attribution.
- [ ] Shopping add/purchase/Undo survives refresh and appears correctly in iOS.
- [ ] Inventory by area and stock changes persist.
- [ ] Drawing color/undo/clear/save work on touch hardware.
- [ ] Weather shows real conditions and hourly/seven-day details.
- [ ] Departure headline uses leave time; stale departures disappear.
- [ ] Bins show correct dates and all relevant types.
- [ ] Idle return, Escape/back, focus, saving, and failure recovery work.

### iOS and pairing

- [ ] One admin scans, names, approves, configures, and revokes a real display.
- [ ] A second admin sees and edits the same shared configuration.
- [ ] Personal commute schedules remain separate from shared display automation.
- [ ] Trains toggle, route, direction, and time relevance affect the actual display.
- [ ] Initial/save/applied text reflects what the system has confirmed.
- [ ] Disconnect failure keeps the screen open and allows retry.
- [ ] Camera unavailable/denied, invalid QR, wrong server, expiry, duplicate claim, and transient failures have correct feedback.
- [ ] Name limits and multibyte names have matching client/server validation.
- [ ] Approval permission matches the agreed server policy.

### Reliability and release

- [ ] One failed provider leaves successful cards usable and shows accurate partial failure.
- [ ] A hung request cannot indefinitely block all refreshed content.
- [ ] Cache freshness does not advance on failure; corrupt/old cache recovers safely.
- [ ] Revocation clears the device's credentials and data and returns to pairing.
- [ ] Time tests cover household timezone, BST/DST, midnight, Sunday, month, and year rollover.
- [ ] Server integration tests prove new display routes and member/admin isolation.
- [ ] Web behavior tests and paired browser tests cover real actions.
- [ ] iOS tests and real scanner/device checks cover the setup/settings paths.
- [ ] Record web commit/image, API commit/image, iOS version/build, CI links, and deployed smoke-test results.

## Handover completion rule

Close each finding only with evidence for its acceptance check. Keep unresolved decisions and unverified live behavior visible. Do not mark the display complete because a PR opened, a build passed, or the unpaired page rendered.
