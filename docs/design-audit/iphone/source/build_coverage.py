"""Record capture scope and the results of each simulator run."""
import json
import re
from pathlib import Path
root = Path(__file__).resolve().parents[1]
profiles = [
 ('iphone-light','iPhone 17 Pro','Standard','Light; Increase Contrast off'),
 ('iphone-repairs','iPhone 17 Pro','Standard','Light; repaired capture selectors'),
 ('iphone-final','iPhone 17 Pro','Standard','Light; settled popovers and remaining forms'),
 ('iphone-supplement','iPhone 17 Pro','Standard','Light; calendar menu and recurrence variants'),
 ('iphone-small','iPhone SE (3rd generation)','Accessibility XXXL','Light; core tour and calendar agenda'),
 ('iphone-small-extra','iPhone SE (3rd generation)','Accessibility XXXL','Light; extended tour'),
 ('iphone-small-final','iPhone SE (3rd generation)','Accessibility XXXL','Light; corrected family-name selector'),
 ('iphone-contrast','iPhone 17 Pro','Standard','Dark; Increase Contrast on; Reduce Motion and Reduce Transparency preferences enabled'),
 ('iphone-dark','iPhone 17 Pro','Standard','Dark; other inherited preferences not recorded before capture'),
]
runs=[]
for name in ['baseline','full','full-retry','small','repairs','final','supplement','small-extra','small-final','small-restock','options-final','contrast']:
 path=Path('/private/tmp/family-hub-audit-'+name+'.log')
 if not path.exists():continue
 text=path.read_text(errors='replace')
 cases=[{'test':m.group(1),'result':m.group(2),'seconds':float(m.group(3))} for m in re.finditer(r"Test Case '-\[FamilyHubUITests\.(.*?)\]' (passed|failed) \(([\d.]+) seconds\)",text)]
 complete='** TEST SUCCEEDED **' in text or '** TEST EXECUTE SUCCEEDED **' in text or '** TEST FAILED **' in text or '** TEST EXECUTE FAILED **' in text
 runs.append({'run':name,'complete':complete,'passed':sum(c['result']=='passed' for c in cases),'failed':sum(c['result']=='failed' for c in cases),'cases':cases,'runnerStartupFailure':'test runner crashed before establishing connection' in text})
(root/'evidence'/'test-runs.json').write_text(json.dumps(runs,indent=2)+'\n')
lines=['# Capture coverage and checks','', '## Capture matrix','', '| Profile | Device | Text | Display / scope | PNG captures |','|---|---|---|---|---|']
for name,device,text,display in profiles:
 count=len(list((root/'captures'/name).glob('*.png')))
 if count: lines.append(f'| {name} | {device} | {text} | {display} | {count} |')
lines += ['','All devices use iOS 26.5. The primary iPhone 17 Pro viewport is 402 × 874 points. The iPhone SE viewport is 375 × 667 points. Captures include forms, keyboards, menus, confirmations, long-screen scroll positions, and selected empty/result states.','', '## Simulator runs','', '| Run | Passed | Failed | State |','|---|---|---|---|']
for run in runs:lines.append(f'| {run["run"]} | {run["passed"]} | {run["failed"]} | {"Complete" if run["complete"] else "In progress"}{"; runner startup crash" if run["runnerStartupFailure"] else ""} |')
lines += ['','[Per-test results](evidence/test-runs.json) record the capture harness results. Initial failures came from capture selectors, scroll direction, native popover dismissal, or test-runner startup. Later runs correct those paths. The table keeps the original failures visible. These are not reported as production app failures.','', '## Coverage limits','',
 '- Some early captures include a sheet or popover transition. Prefer the final profile where available. The audit does not treat transition frames as layout defects.',
 '- Two early extended-SE captures labelled as new-item forms actually show a persistent delete popover. They are renamed as delete-popover evidence. The core SE item-form capture and the final Pro captures show the actual new-item form.',
 '- The external identity-provider sign-in, consent, cancellation, and authenticated return screens are not captured. The demo does not provide an identity-provider session.',
 '- The system photo picker is captured while loading. Its settled gallery, selected-photo return, and upload failure states are not reviewed.',
 '- Notification permission prompts, device registration, actual push delivery, and the non-admin account view are not captured. Demo mode disables device registration and uses an admin account.',
 '- Location permission, denied-location, and captured-location confirmation states are not captured. The saved demo home information is captured.',
 '- Server outage, cache staleness, generic loading, and every action-error variant are not forced. The invalid-URL error, no-result search, empty shopping section, empty inventory area, and zero-recipient message result are captured.',
 '- The offline mockup is a proposed recovery pattern. It has no matching simulator capture. [View the concept](mockups/offline.png).',
 '- The review covers the screen register and material layout variants. It does not enumerate every combination of content, picker value, account role, and permission state.',
 '- Dark appearance has a core tour. The contrast run samples the main tabs, settings, reminders, a chore form, recipe details, and cook mode. It does not repeat every screen under every display preference.',
 '- The contrast run records preference values after a simulator restart. Static captures establish appearance only. They do not prove the timing or continuity of motion, or full accessibility conformance.',
 '- The large-text mockups demonstrate an adaptive layout. Their HTML typography approximates native Dynamic Type and is not an exact iOS accessibility-size rendering.',
 '', '## What the audit confirms','',
 '- The core dark tour passes. The core iPhone SE tour and its accessibility calendar interaction pass.',
 '- Calendar month/year/day/week navigation, date selection, overlap inspection, swipe, pinch, and Today have simulator interaction evidence.',
 '- Meal selection, recipe links, cook-page progression, shopping purchase/restock, and local demo item creation have simulator interaction evidence.',
 '- PNG files retain the simulator output. Mockups are separate PNG files with a visible PROPOSAL label.',
 '- Each registered screen has its own goal, usability review, interaction review, main-action review, Apple assessment, and proposed change.',
 '', '## Checks for a later app implementation','',
 '1. Preserve all features, access rules, Friday–Thursday meal weeks, and independent ingredient actions.',
 '2. Compare Today with no departure, a near departure, no chores, overdue chores, and long event descriptions.',
 '3. Check readable calendar titles, overlapping events, month zoom, Today, and maximum text size on the smallest supported iPhone.',
 '4. Verify repeat summaries against the existing monthly/custom rules and both stop conditions.',
 '5. Verify meal edit versus recipe open, selected-week retention, recipe-link removal, and cook progress labels.',
 '6. Verify count and level thresholds at equality. The iOS model marks a level equal to its threshold as low.',
 '7. Check notification permission, server push availability, non-admin settings, zero-recipient results, retry, and partial sends.',
 '8. Check VoiceOver names, reading order, focus after sheets, colour contrast, Reduce Motion, and Reduce Transparency on a physical iPhone.',
 '', '## Files and repeatability','',
 'The capture harness lives in `source/IPhoneAuditUITests.swift`. Copy it into `ios/FamilyHub/FamilyHubUITests/` only for an audit run. Remove that temporary copy afterward. It does not belong in the default CI tour.',
 '', 'Use `source/export_captures.py` with an `xcresulttool export attachments` directory and a profile name. Use `source/render_mockups.cjs` to render the editable layouts. Use `source/build_report.py`, `source/build_coverage.py`, and `source/validate_report.py` to rebuild and check the report. Python tools use Pillow. The renderer uses Playwright and the local Chrome executable.',
 '', 'Xcode result bundles remain under `/private/tmp/family-hub-audit-*.xcresult` for this session. Durable named PNG captures, attachment manifests, and per-test results are included in this package.',
 '', '## Workboard','', 'Workboard tools are unavailable in this session. No project or status post is created. No app connection settings change.']
(root/'coverage.md').write_text('\n'.join(lines)+'\n')
print('Coverage register written')
