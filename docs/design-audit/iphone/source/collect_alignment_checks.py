"""Save passing xcresult summaries and separate diagnostic results."""
import datetime
import json
from pathlib import Path
import re
import subprocess
import sys

root = Path(__file__).resolve().parents[1] / 'alignment'
evidence = root / 'evidence'
evidence.mkdir(parents=True, exist_ok=True)
runs = []
diagnostics = []
for argument in sys.argv[1:]:
    diagnostic = argument.startswith('--diagnostic=')
    if diagnostic:
        argument = argument.removeprefix('--diagnostic=')
    name, bundle = argument.split('=', 1)
    if not re.fullmatch('[a-z0-9-]+', name):
        raise SystemExit('Use a short lower-case run name.')
    result = subprocess.run(
        ['xcrun', 'xcresulttool', 'get', 'test-results', 'summary', '--path', bundle],
        capture_output=True, text=True, check=True,
    )
    summary = json.loads(result.stdout)
    if not diagnostic and (summary['result'] != 'Passed' or summary['failedTests'] or not summary['totalTestCount']):
        raise SystemExit(f'{name} does not contain a non-empty passing test run.')
    filename = name + '.json'
    (evidence / filename).write_text(json.dumps(summary, indent=2) + '\n')
    records = diagnostics if diagnostic else runs
    records.append({
        'name': name,
        'result': summary['result'],
        'passed': summary['passedTests'],
        'failed': summary['failedTests'],
        'skipped': summary['skippedTests'],
        'total': summary['totalTestCount'],
        'evidence': 'evidence/' + filename,
        'runtimeWarnings': summary.get('runtimeWarnings', []),
    })
if not runs:
    raise SystemExit('Supply run-name=/absolute/path/to/results.xcresult arguments.')
checks = {
    'generatedAt': datetime.datetime.now(datetime.timezone.utc).isoformat(),
    'scope': 'Targeted iPhone visual and interaction checks. The app unit suite runs separately. This is not a whole UI-suite result.',
    'runs': runs,
    'diagnostics': diagnostics,
    'comparisons': 33,
    'captures': {folder: len(json.loads((root / folder / 'manifest.json').read_text()))
                 for folder in ['captures', 'small-captures']},
    'limits': [
        'Earlier diagnostic runs include failed selectors and a zero-test run. They do not count as passing checks.',
        'Simulator summaries retain frame-dimension warnings without a source location.',
        'Demo mode does not prove sign-in, location capture, notification delivery, physical VoiceOver, or Siri.',
    ],
}
(root / 'checks.json').write_text(json.dumps(checks, indent=2) + '\n')
print(json.dumps({run['name']: {'passed': run['passed'], 'skipped': run['skipped']} for run in runs}))
