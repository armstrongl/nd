#!/bin/bash
# Project-specific documentation style linter for nd.
# Checks rules that markdownlint cannot enforce.
# Run: ./scripts/lint-docs.sh [file ...]
# With no arguments, checks all docs + README.

set -euo pipefail

RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

errors=0
warnings=0

# Determine files to check.
if [ $# -gt 0 ]; then
  files=("$@")
else
  files=()
  while IFS= read -r f; do
    files+=("$f")
  done < <(find docs/guide README.md CONTRIBUTING.md ARCHITECTURE.md -name '*.md' -not -path '*/reference/*' 2>/dev/null)
fi

if [ ${#files[@]} -eq 0 ]; then
  echo "No markdown files to check."
  exit 0
fi

check_file() {
  local file="$1"
  local file_errors=0

  # 1. bash code fences (should be shell)
  if grep -n '```bash' "$file" >/dev/null 2>&1; then
    while IFS= read -r match; do
      echo -e "${RED}error${NC}: $file:$match: use \`\`\`shell instead of \`\`\`bash"
      file_errors=$((file_errors + 1))
    done < <(grep -n '```bash' "$file")
  fi

  # 2. Forbidden words (case-insensitive, whole words only)
  for word in "simply" "straightforward" "obviously"; do
    if grep -inw "$word" "$file" >/dev/null 2>&1; then
      while IFS= read -r match; do
        echo -e "${RED}error${NC}: $file:$match: forbidden word '$word'"
        file_errors=$((file_errors + 1))
      done < <(grep -inw "$word" "$file")
    fi
  done

  # "just" and "easy/easily" — only flag outside code blocks
  # (simple grep; code blocks may cause false positives in rare cases)
  for word in "just" "easy" "easily" "simple"; do
    if grep -inw "$word" "$file" | grep -v '^\s*#' | grep -v '```' >/dev/null 2>&1; then
      while IFS= read -r match; do
        # Skip lines inside code fences (heuristic: indented 4+ spaces or starts with common code chars)
        if echo "$match" | grep -qE '^\s{4,}|^\s*`|^\s*\$|^\s*#|^\s*//' 2>/dev/null; then
          continue
        fi
        echo -e "${YELLOW}warning${NC}: $file:$match: likely forbidden word '$word' (verify not in code)"
        warnings=$((warnings + 1))
      done < <(grep -inw "$word" "$file")
    fi
  done

  # 3. Old-style tree notation (+-- or | ) — skip lines inside fenced code blocks
  while IFS= read -r match; do
    echo -e "${RED}error${NC}: $file:$match: use standard tree notation (├──/└──/│) not +--"
    file_errors=$((file_errors + 1))
  done < <(awk 'BEGIN{fence=0} /^```/{fence=!fence; next} !fence && /\+--/{print NR": "$0}' "$file")

  # 4. Title Case headings (H2/H3 with multiple capitalized words)
  # Heuristic: flag headings where 2+ consecutive words start with uppercase
  # Skip H1 (page titles) and lines with proper nouns/acronyms
  if grep -nE '^#{2,} ' "$file" >/dev/null 2>&1; then
    while IFS= read -r line; do
      lineno=$(echo "$line" | cut -d: -f1)
      heading=$(echo "$line" | sed 's/^[0-9]*://' | sed 's/^#* //')
      # Count words starting with uppercase (excluding first word)
      rest=$(echo "$heading" | cut -d' ' -f2-)
      caps=$(echo "$rest" | { grep -oE '\b[A-Z][a-z]+' || true; } | wc -l | tr -d ' ')
      total=$(echo "$rest" | wc -w | tr -d ' ')
      if [ "$total" -gt 0 ] && [ "$caps" -gt 0 ]; then
        # If more than half of remaining words are capitalized, flag it
        threshold=$(( (total + 1) / 2 ))
        if [ "$caps" -ge "$threshold" ] && [ "$total" -gt 1 ]; then
          echo -e "${YELLOW}warning${NC}: $file:${lineno}: possible Title Case heading: '$heading' (use sentence case)"
          warnings=$((warnings + 1))
        fi
      fi
    done < <(grep -nE '^#{2,} ' "$file")
  fi

  # 5. Em-dash separators in list items (- **Term** -- description)
  # Only flag -- surrounded by spaces in non-code lines
  if grep -nE '^\s*-.*\s--\s' "$file" | grep -v '```' | grep -v '^\s*#' >/dev/null 2>&1; then
    while IFS= read -r match; do
      echo -e "${YELLOW}warning${NC}: $file:$match: use ':' separator instead of '--'"
      warnings=$((warnings + 1))
    done < <(grep -nE '^\s*-.*\s--\s' "$file" | grep -v '```')
  fi

  errors=$((errors + file_errors))
}

echo "Checking ${#files[@]} file(s)..."
echo ""

for file in "${files[@]}"; do
  if [ -f "$file" ]; then
    check_file "$file"
  fi
done

echo ""
if [ $errors -gt 0 ] || [ $warnings -gt 0 ]; then
  echo "Results: ${errors} error(s), ${warnings} warning(s)"
fi

if [ $errors -gt 0 ]; then
  echo ""
  echo "Fix errors before committing. Warnings should be reviewed."
  exit 1
fi

if [ $warnings -gt 0 ]; then
  echo "No errors. $warnings warning(s) to review."
  exit 0
fi

echo "All checks passed."
