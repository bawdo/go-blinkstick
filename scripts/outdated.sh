#!/usr/bin/env bash
#
# Reports outdated dependencies and offers to upgrade them.
#
#   Go modules   Direct requirements in go.mod with a newer release in the
#                same major version, plus any that are deprecated or whose
#                current version is retracted. Indirect modules are counted
#                only: they follow along when go mod tidy runs.
#   Tool pins    golangci-lint and govulncheck wherever they are pinned (the
#                workflows and CONTRIBUTING.md), and the Go versions the CI
#                matrix tests. Report only, as this script never edits those
#                files. Pins that disagree with each other are flagged.
#
# Releases younger than COOLDOWN_DAYS (default 7, matching dependabot.yml)
# are flagged "too fresh" and left out of "all", but can still be picked.
#
# On a terminal it then asks: [a]ll, [p]ick or [N]one. An upgrade runs
# go get, go mod tidy, make test and go vet -tags hardware, then shows the
# go.mod diff. Nothing is committed. Without a terminal it prints the report
# and exits.
#
# Usage: scripts/outdated.sh    (or: make outdated)

set -euo pipefail

COOLDOWN_DAYS="${COOLDOWN_DAYS:-7}"

if ! [[ "$COOLDOWN_DAYS" =~ ^[0-9]+$ ]]; then
    echo "outdated: COOLDOWN_DAYS must be a whole number of days" >&2
    exit 1
fi

if [ -t 1 ]; then
    BOLD=$'\033[1m' DIM=$'\033[2m' RED=$'\033[31m' GREEN=$'\033[32m'
    YELLOW=$'\033[33m' CYAN=$'\033[36m' NC=$'\033[0m'
else
    BOLD='' DIM='' RED='' GREEN='' YELLOW='' CYAN='' NC=''
fi

for bin in go jq; do
    command -v "$bin" > /dev/null || { echo "outdated: $bin is not installed" >&2; exit 1; }
done

# An explicit template, because a bare mktemp -d on macOS ignores TMPDIR.
tmp=$(mktemp -d "${TMPDIR:-/tmp}/outdated.XXXXXX")
trap 'rm -rf "$tmp"' EXIT

header() { printf '\n%s%s%s\n' "$BOLD" "$1" "$NC"; }

# jq filter: whole days since an RFC 3339 time, or -1 when there is none.
# Fractional seconds are stripped because fromdateiso8601 rejects them.
AGE='if . then ((now - (sub("\\.[0-9]+"; "") | fromdateiso8601)) / 86400 | floor) else -1 end'

# Prints the age column, marking releases inside the cooldown.
age_label() {
    local days=$1
    if [ "$days" -lt 0 ]; then
        printf '%s' "age unknown"
    elif [ "$days" -lt "$COOLDOWN_DAYS" ]; then
        printf '%s%dd, too fresh%s' "$YELLOW" "$days" "$NC"
    else
        printf '%dd' "$days"
    fi
}

# ---------------------------------------------------------------- Go modules

echo "Checking Go modules against the module proxy..."
if ! go list -m -u -json all > "$tmp/all.json" 2> "$tmp/all.err"; then
    echo "${RED}outdated: go list failed:${NC}" >&2
    cat "$tmp/all.err" >&2
    exit 1
fi

# Direct modules with an update: path, current, latest, age in days.
jq -r "
    select(.Main != true and (.Indirect | not) and .Update)
    | [.Path, .Version, .Update.Version, (.Update.Time | $AGE)] | @tsv
" "$tmp/all.json" > "$tmp/direct.tsv"

# Direct modules that are deprecated or sit on a retracted version, one line
# per warning (read collapses empty tab-separated fields, so none are emitted).
jq -r '
    select(.Main != true and (.Indirect | not))
    | (if .Deprecated then [.Path, .Version, "deprecated: " + .Deprecated] else empty end),
      (if .Retracted then [.Path, .Version, "retracted: " + (.Retracted | join("; "))] else empty end)
    | @tsv
' "$tmp/all.json" > "$tmp/warn.tsv"

indirect_count=$(jq -s '[.[] | select(.Main != true and .Indirect and .Update)] | length' "$tmp/all.json")

header "Go modules (direct)"
if [ -s "$tmp/direct.tsv" ]; then
    printf '  %s%-40s %-14s %-14s %s%s\n' "$DIM" "MODULE" "CURRENT" "LATEST" "AGE" "$NC"
    while IFS=$'\t' read -r path cur latest days; do
        printf '  %-40s %-14s %s%-14s%s %s\n' \
            "$path" "$cur" "$GREEN" "$latest" "$NC" "$(age_label "$days")"
    done < "$tmp/direct.tsv"
else
    echo "  ${GREEN}All direct modules are up to date.${NC}"
fi

if [ -s "$tmp/warn.tsv" ]; then
    echo
    while IFS=$'\t' read -r path cur msg; do
        printf '  %s!%s %s %s: %s\n' "$RED" "$NC" "$path" "$cur" "$msg"
    done < "$tmp/warn.tsv"
fi

if [ "$indirect_count" -gt 0 ]; then
    printf '\n  %s%d indirect module(s) also have updates; go mod tidy moves those along.%s\n' \
        "$DIM" "$indirect_count" "$NC"
fi

# ---------------------------------------------------------------- Tool pins

# Files that may pin a tool version. Missing ones are skipped.
shopt -s nullglob
workflows=(.github/workflows/*.yml .github/workflows/*.yaml)
shopt -u nullglob
pin_files=("${workflows[@]}")
[ -f CONTRIBUTING.md ] && pin_files+=(CONTRIBUTING.md)

# Prints "<version> <age-days>" for a module's latest release, or nothing.
latest_of() {
    go list -m -json "$1@latest" 2> /dev/null | jq -r "[.Version, (.Time | $AGE)] | @tsv" 2> /dev/null || true
}

# Reports one tool. Reads "file:line:version" lines (one per pin) from the
# file named by $2, and compares each pin with the module's latest release.
tool_report() {
    local name=$1 refs=$2 module=$3 latest days pinned count
    if [ ! -s "$refs" ]; then
        printf '  %-14s %spin not found in the workflows or CONTRIBUTING.md%s\n' "$name" "$YELLOW" "$NC"
        return
    fi
    IFS=$'\t' read -r latest days <<< "$(latest_of "$module")" || true
    count=$(cut -d: -f3 "$refs" | sort -u | wc -l | tr -d ' ')
    if [ "$count" -gt 1 ]; then
        pinned="mixed"
    else
        pinned=$(head -n 1 "$refs" | cut -d: -f3)
    fi

    if [ -z "${latest:-}" ]; then
        printf '  %-14s %-10s %slatest version lookup failed%s\n' "$name" "$pinned" "$YELLOW" "$NC"
    elif [ "$pinned" = "$latest" ]; then
        printf '  %-14s %-10s %sup to date%s\n' "$name" "$pinned" "$GREEN" "$NC"
    else
        printf '  %-14s %-10s -> %s%-10s%s %s\n' "$name" "$pinned" \
            "$GREEN" "$latest" "$NC" "$(age_label "${days:--1}")"
    fi
    if [ "$count" -gt 1 ]; then
        printf '  %-14s %s! the pins disagree with each other%s\n' "" "$RED" "$NC"
    fi

    local file line ver
    while IFS=: read -r file line ver; do
        if [ -n "${latest:-}" ] && [ "$ver" != "$latest" ]; then
            printf '  %-14s %s%-10s %s:%s%s\n' "" "$CYAN" "$ver" "$file" "$line" "$NC"
        else
            printf '  %-14s %s%-10s %s:%s%s\n' "" "$DIM" "$ver" "$file" "$line" "$NC"
        fi
    done < "$refs"
}

header "Tool pins (report only)"
if [ "${#pin_files[@]}" -eq 0 ]; then
    echo "  ${YELLOW}No workflow files or CONTRIBUTING.md found, skipping.${NC}"
else
    # golangci-lint: the first "version:" input after each use of the action.
    awk 'FNR == 1 { found = 0 }
        /golangci-lint-action@/ { found = 1; next }
        found && /^[[:space:]]*version:/ {
            v = $2; gsub(/["'\'']/, "", v); print FILENAME ":" FNR ":" v; found = 0 }' "${pin_files[@]}" > "$tmp/lint.refs"
    tool_report golangci-lint "$tmp/lint.refs" github.com/golangci/golangci-lint/v2

    # govulncheck: every "govulncheck@vX.Y.Z", as file:line:vX.Y.Z.
    { grep -n -H -o 'govulncheck@v[0-9][0-9.]*' "${pin_files[@]}" || true; } \
        | sed 's/:govulncheck@/:/' > "$tmp/vuln.refs"
    tool_report govulncheck "$tmp/vuln.refs" golang.org/x/vuln

    # Go: literal versions in the matrix ("go: [...]") and in go-version
    # inputs, either quote style, as major.minor. ${{ matrix.go }} is skipped.
    go_lines=$(
        { grep -h -E '^[[:space:]]*(go|go-version):' "${workflows[@]}" 2> /dev/null || true; } \
            | grep -o -E '[0-9]+\.[0-9]+' | sort -u -t. -k1,1n -k2,2n || true
    )
    go_newest=$(tail -n 1 <<< "$go_lines")
    go_oldest=$(head -n 1 <<< "$go_lines")
    go_latest=$(go list -m -f '{{.Version}}' go@latest 2> /dev/null || true)
    go_latest_line=$(cut -d. -f1-2 <<< "$go_latest")
    go_directive=$(awk '$1 == "go" { print $2; exit }' go.mod)

    if [ -z "$go_newest" ] || [ -z "$go_latest" ]; then
        printf '  %-14s %scould not compare (CI newest "%s", latest "%s")%s\n' \
            "Go" "$YELLOW" "$go_newest" "$go_latest" "$NC"
    elif [ "$go_newest" = "$go_latest_line" ]; then
        printf '  %-14s %-10s %sup to date (latest %s)%s\n' "Go" "$go_newest" "$GREEN" "$go_latest" "$NC"
    else
        printf '  %-14s %-10s -> %s%-10s%s newer release line\n' \
            "Go" "$go_newest" "$GREEN" "$go_latest" "$NC"
        printf '  %-14s %sadd it to the CI matrix and go-version in .github/workflows/*.yml%s\n' \
            "" "$CYAN" "$NC"
    fi
    if [ -n "$go_oldest" ]; then
        printf '  %-14s %sCI tests %s to %s. go.mod says go %s, the oldest Go this library\n' \
            "" "$DIM" "$go_oldest" "$go_newest" "$go_directive"
        printf '  %-14s supports; raise it deliberately, as it is a breaking change for users.%s\n' "" "$NC"
        if [ "$(cut -d. -f1-2 <<< "$go_directive")" != "$go_oldest" ]; then
            printf '  %-14s %s! go.mod (go %s) and the oldest CI version (%s) differ%s\n' \
                "" "$YELLOW" "$go_directive" "$go_oldest" "$NC"
        fi
    fi
fi

# ---------------------------------------------------------------- Upgrade

echo
if [ ! -s "$tmp/direct.tsv" ]; then
    exit 0
fi

if [ ! -t 0 ] || [ ! -t 1 ]; then
    echo "Not on a terminal, so not offering upgrades."
    exit 0
fi

if git rev-parse --is-inside-work-tree > /dev/null 2>&1 \
    && ! { git diff --quiet -- go.mod go.sum && git diff --cached --quiet -- go.mod go.sum; }; then
    echo "${YELLOW}go.mod or go.sum has uncommitted changes. Commit or stash them first, so the"
    echo "upgrade diff is only the upgrade.${NC}"
    exit 0
fi

eligible=$(awk -F'\t' -v c="$COOLDOWN_DAYS" '$4 >= c' "$tmp/direct.tsv" | wc -l | tr -d ' ')
read -r -p "Upgrade? [a]ll ($eligible past the cooldown), [p]ick, [N]one: " choice || choice=""

: > "$tmp/chosen"
case "$choice" in
    a | A)
        awk -F'\t' -v c="$COOLDOWN_DAYS" '$4 >= c { print $1 "@" $3 }' "$tmp/direct.tsv" > "$tmp/chosen"
        ;;
    p | P)
        # Read answers from the terminal: stdin of this loop is the module list.
        while IFS=$'\t' read -r path cur latest days; do
            note=""
            [ "$days" -lt "$COOLDOWN_DAYS" ] && note=" ${YELLOW}($(age_label "$days"))${NC}"
            read -r -p "  $path $cur -> $latest$note? [y/N] " yn < /dev/tty || yn=""
            case "$yn" in y | Y) echo "$path@$latest" >> "$tmp/chosen" ;; esac
        done < "$tmp/direct.tsv"
        ;;
esac

if [ ! -s "$tmp/chosen" ]; then
    echo "Nothing upgraded."
    exit 0
fi

# Runs one step, and on failure says how to back out and stops.
step() {
    printf '%s==>%s %s\n' "$CYAN" "$NC" "$*"
    if ! "$@"; then
        echo "${RED}Failed: $*${NC}"
        echo "The changes are left in place. To back out: git checkout -- go.mod go.sum"
        exit 1
    fi
}

modules=()
while IFS= read -r m; do modules+=("$m"); done < "$tmp/chosen"

step go get "${modules[@]}"
step go mod tidy
step make test
# Hardware tests need sticks attached, but vet still type-checks those files.
step go vet -tags hardware ./...

header "go.mod changes"
git --no-pager diff --stat -- go.mod go.sum
git --no-pager diff -- go.mod
echo
echo "${GREEN}Upgraded ${#modules[@]} module(s). make test and the hardware vet pass. Nothing has been committed.${NC}"
