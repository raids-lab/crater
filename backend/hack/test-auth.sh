#!/usr/bin/env bash
# Run authentication tests without loading developer credentials or connecting to
# the cluster database. Go overlays change only startup configuration loading and
# operation-log migration in the test build; authentication code is unchanged.
set -euo pipefail

backend_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
python3 - "$backend_dir" <<'PY'
import json
from pathlib import Path
import subprocess
import sys
import tempfile

root = Path(sys.argv[1])
replacements = {
    "internal/service/operation_log.go": (
        "func init() {", "func migrateOperationLogsForAuthTests() {"
    ),
    "pkg/config/config.go": (
        "config = initConfig()", "config = &Config{}"
    ),
}

with tempfile.TemporaryDirectory(prefix="crater-auth-") as directory:
    temporary = Path(directory)
    overlay = {}
    for relative, (before, after) in replacements.items():
        original = root / relative
        source = original.read_text()
        if source.count(before) != 1:
            raise SystemExit(f"Test isolation needs updating: {relative}")
        replacement = temporary / original.name
        replacement.write_text(source.replace(before, after, 1))
        overlay[str(original)] = str(replacement)
    manifest = temporary / "overlay.json"
    manifest.write_text(json.dumps({"Replace": overlay}))
    result = subprocess.run([
        "go", "test", "-overlay", str(manifest),
        "./internal/util", "./internal/middleware", "./internal/handler", "./internal/bizerr", "./internal/resputil",
        "-run", "^Test(Authentication|Wrap|HandleError|Success)",
        "-count=1", "-timeout=60s", "-v",
    ], cwd=root)
    sys.exit(result.returncode)
PY
