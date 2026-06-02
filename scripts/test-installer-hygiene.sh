#!/usr/bin/env bash
#
# Regression test for the container-tools installer cache/backup hygiene
# (build-container-tools.sh, the upgrade/backup block around lines 409-445).
#
# It builds a fixture install tree, runs the THREE hygiene operations the
# installer performs — in the same order — against the fixture (with SUDO=""),
# and asserts the invariants:
#   (i)   the kept backup/prior exists,
#   (ii)  the transient cache is NOT archived under backup/prior,
#   (iii) backups do not nest (no backup/prior/backup),
#   (iv)  the live var/cache is wiped on install,
#   (v)   real prior install content survives in backup/prior (rollback intact).
#
# The command snippets below mirror build-container-tools.sh and MUST be kept
# in sync with it. If you change the installer's hygiene logic, update this test.
#
# Exit status: 0 = all invariants hold; 1 = a regression was detected.

set -euo pipefail

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

SUDO=""
YOUR_MCP="p2kb-mcp"
TARGET="$WORK/opt/container-tools"
INST="$TARGET/$YOUR_MCP"
SCRIPT_DIR="$WORK/freshpkg"

# --- Fresh package being installed (what cp -r copies in) ---
mkdir -p "$SCRIPT_DIR/bin/platforms"
echo "newbin" > "$SCRIPT_DIR/bin/$YOUR_MCP"

# --- Existing install: real content + runtime cache + nested backups + cache
#     wrongly archived under backup by an older installer version ---
mkdir -p "$INST/bin/platforms"
echo "oldbin"  > "$INST/bin/$YOUR_MCP"
echo "oldplat" > "$INST/bin/platforms/${YOUR_MCP}-linux-amd64"
echo "readme"  > "$INST/README.md"
mkdir -p "$INST/var/cache"; echo "runtime" > "$INST/var/cache/p2kbPasm2Add.yaml"
mkdir -p "$INST/backup/prior/backup/prior"                       # geometric nesting (depth > 1)
mkdir -p "$INST/backup/prior/var/cache"; echo "old" > "$INST/backup/prior/var/cache/old.yaml"
echo "priorReadme" > "$INST/backup/prior/README.md"              # legit prior content that MUST survive

# ===================== hygiene operations (mirror installer) =====================

# (3) Self-healing purge — installer runs this inside `if [ -d "$TARGET/$YOUR_MCP" ]`
if [ -d "$INST/backup" ]; then
    find "$INST/backup" -type d -name "cache" -exec $SUDO rm -rf {} + 2>/dev/null || true
    $SUDO rm -rf "$INST/backup/prior/backup" 2>/dev/null || true
fi

# move existing install to tmp (installer: mv "$TARGET/$YOUR_MCP" "/tmp/$YOUR_MCP-prior")
PRIOR_TEMP="$WORK/tmp/$YOUR_MCP-prior"
mkdir -p "$WORK/tmp"
$SUDO mv "$INST" "$PRIOR_TEMP"

# install fresh (installer: cp -r "$SCRIPT_DIR" "$TARGET/$YOUR_MCP")
$SUDO cp -r "$SCRIPT_DIR" "$INST"

# (2) wipe live cache on every install
$SUDO rm -rf "$INST/var/cache"

# (1) strip var/ + backup/ from the prior snapshot, then archive it
if [ -n "$PRIOR_TEMP" ] && [ -d "$PRIOR_TEMP" ]; then
    $SUDO mkdir -p "$INST/backup"
    $SUDO rm -rf "$INST/backup/prior"
    $SUDO rm -rf "$PRIOR_TEMP/var" "$PRIOR_TEMP/backup"
    $SUDO mv "$PRIOR_TEMP" "$INST/backup/prior"
fi

# ===================== assertions =====================
fail=0
check() { # check <description> <test-expr...>
    local desc="$1"; shift
    if "$@"; then echo "PASS: $desc"; else echo "FAIL: $desc"; fail=1; fi
}

check "(i)   backup/prior exists"                       test -d "$INST/backup/prior"
check "(ii)  backup/prior/var stripped (no archived cache)" test ! -e "$INST/backup/prior/var"
check "(iii) backup/prior/backup absent (no nesting)"   test ! -e "$INST/backup/prior/backup"
check "(iv)  live var/cache wiped on install"           test ! -e "$INST/var/cache"
check "(v)   prior bin survived (rollback intact)"      test -f "$INST/backup/prior/bin/$YOUR_MCP"
check "(v)   prior README survived (rollback intact)"   test -f "$INST/backup/prior/README.md"
check "      fresh install in place"                    test "$(cat "$INST/bin/$YOUR_MCP")" = "newbin"

echo
if [ "$fail" -eq 0 ]; then
    echo "=== installer hygiene: ALL INVARIANTS HOLD ==="
    exit 0
fi
echo "=== installer hygiene: REGRESSION DETECTED ==="
exit 1
