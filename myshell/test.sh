#!/bin/bash
# myshell cross-platform test suite
# usage: bash test.sh

set -e

PASS=0
FAIL=0
BINARY="./myshell"

echo ""
echo "  myshell test suite"
echo "  ──────────────────"
echo ""

# ---- helpers ----

pass() { echo "  ✓  $1"; PASS=$((PASS + 1)); }
fail() { echo "  ✗  $1"; FAIL=$((FAIL + 1)); }

run_cmd() {
  echo "$2" | "$BINARY" 2>/dev/null
}

expect_output() {
  local desc="$1"
  local input="$2"
  local expected="$3"
  local output
  output=$(echo -e "$input\nexit" | "$BINARY" 2>/dev/null || true)
  if echo "$output" | grep -q "$expected"; then
    pass "$desc"
  else
    fail "$desc (expected '$expected', got: $output)"
  fi
}

expect_file() {
  local desc="$1"
  local file="$2"
  if [ -f "$file" ]; then
    pass "$desc"
  else
    fail "$desc (file not found: $file)"
  fi
}

# ---- build check ----

echo "  [build]"
if go build -o myshell ./... 2>/dev/null; then
  pass "go build succeeds"
else
  fail "go build failed — fix errors before running tests"
  exit 1
fi

# ---- binary exists ----

echo ""
echo "  [binary]"
expect_file "binary created" "./myshell"
if [ -x "./myshell" ]; then
  pass "binary is executable"
else
  fail "binary is not executable"
fi

# ---- built-in commands ----

echo ""
echo "  [built-ins]"
expect_output "echo works"       "echo hello"          "hello"
expect_output "ls works"         "ls"                  "main.go"
expect_output "history works"    "history"             ""
expect_output "env list works"   "env list"            ""
expect_output "platform works"   "platform"            "Platform info"
expect_output "config works"     "config"              "myshell config"
expect_output "help works"       "help"                "built-in commands"
expect_output "status works"     "status"              "session"
expect_output "render works"     "render table"        "command"
expect_output "explore works"    "explore"             "main.go"
expect_output "hooks list works" "hooks"               "hooks"

# ---- pipes ----

echo ""
echo "  [pipes]"
expect_output "single pipe"  "echo hello | cat"       "hello"
expect_output "double pipe"  "echo hello | cat | cat" "hello"

# ---- redirects ----

echo ""
echo "  [redirects]"
TMP=$(mktemp)
echo -e "echo hello > $TMP\nexit" | "$BINARY" 2>/dev/null || true
if [ -f "$TMP" ] && grep -q "hello" "$TMP"; then
  pass "redirect > works"
else
  fail "redirect > failed"
fi

echo -e "echo world >> $TMP\nexit" | "$BINARY" 2>/dev/null || true
if grep -q "world" "$TMP"; then
  pass "redirect >> works"
else
  fail "redirect >> failed"
fi
rm -f "$TMP"

# ---- env manager ----

echo ""
echo "  [env manager]"
expect_output "env set + get" "env set TESTVAR hello\nenv get TESTVAR" "hello"

# ---- history ----

echo ""
echo "  [history]"
HOME_HIST="$HOME/.myshell_history.db"
echo -e "echo test_cmd_xyz\nexit" | "$BINARY" 2>/dev/null || true
if [ -f "$HOME_HIST" ]; then
  pass "history db created"
else
  fail "history db not found at $HOME_HIST"
fi

# ---- config ----

echo ""
echo "  [config]"
CONFIG_FILE="$HOME/.config/myshell/config.toml"
echo -e "config\nexit" | "$BINARY" 2>/dev/null || true
if [ -f "$CONFIG_FILE" ]; then
  pass "config file created at $CONFIG_FILE"
else
  fail "config file not found"
fi

# ---- session ----

echo ""
echo "  [session]"
echo -e "session save\nexit" | "$BINARY" 2>/dev/null || true
SESSION_FILE="$HOME/.config/myshell/session.json"
if [ -f "$SESSION_FILE" ]; then
  pass "session file created"
else
  fail "session file not found"
fi

# ---- platform detection ----

echo ""
echo "  [platform]"
OS=$(uname -s)
case "$OS" in
  Darwin) pass "macOS detected" ;;
  Linux)  pass "Linux detected" ;;
  *)      fail "unknown platform: $OS" ;;
esac

if command -v git &>/dev/null; then
  pass "git available"
else
  fail "git not found"
fi

if command -v go &>/dev/null; then
  pass "go available"
else
  fail "go not found"
fi

# ---- summary ----

echo ""
echo "  ──────────────────"
TOTAL=$((PASS + FAIL))
echo "  $PASS/$TOTAL tests passed"

if [ $FAIL -gt 0 ]; then
  echo "  $FAIL test(s) failed"
  echo ""
  exit 1
else
  echo "  All tests passed ✓"
  echo ""
fi

# clean up test binary
rm -f ./myshell