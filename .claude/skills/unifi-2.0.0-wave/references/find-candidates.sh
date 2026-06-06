#!/usr/bin/env bash
# Find ready-to-work 2.0.0 issues WITHOUT dumping their bodies into context.
#
# One `gh` call lists every open milestone-2.0.0 issue; bodies are parsed HERE (for `Depends on #N`
# lines) and digested into a compact table — only metadata + status leaves this script, never the
# descriptions. Pull a full body lazily (`gh issue view <N>`) only for issues you actually select.
#
# Dependency state is COMPUTED fresh every run, never stored: a `Depends on #N` whose target is still
# open means BLOCKED; once every dep is closed/merged the issue is READY. There is no `blocked` label
# to set, clear, or go stale — the script is the single source of truth.
#
# This script is intentionally GENERIC: no hardcoded issue numbers, slugs, paths, or skeleton knowledge.
# Skeleton-first falls out for free — while the skeleton is open, everything that `Depends on` it is
# BLOCKED, so the skeleton is the only READY candidate.
#
# Usage:
#   find-candidates.sh            # human table: READY first, then BLOCKED, then CLAIMED
#   find-candidates.sh --json     # machine-readable array (number,type,breaking,status,deps,unmet,...)
#
# Status legend:
#   READY    — no unmet dependency, not claimed; eligible for a wave NOW
#   BLOCKED  — has a `Depends on #N` whose target is still open; sequence it for a later wave
#   CLAIMED  — carries in-progress/in-review (owned by a running/open wave); skip unless the claim is stale

set -euo pipefail

MILESTONE="${WAVE_MILESTONE:-2.0.0}"
MODE="table"
case "${1:-}" in
  --json) MODE="json" ;;
  "" ) ;;
  *) echo "unknown arg: $1 (use --json)" >&2; exit 2 ;;
esac

# Single network call. Enrich each issue with: type label, breaking?, claimed?, parsed deps, the deps
# still open (unmet), and computed status — no bodies survive past this jq pipeline.
ENRICHED="$(gh issue list --milestone "$MILESTONE" --state open --limit 300 \
  --json number,title,labels,body | jq '
  ([.[].number]) as $open
  | map(
      . as $i
      | ([$i.labels[].name]) as $L
      | ([$i.body // "" | scan("(?i)depends on[^\n]*")] | join(" ")) as $depline
      | ([$depline | scan("#([0-9]+)") | .[0] | tonumber] | unique - [$i.number]) as $deps
      | ($deps | map(select(. as $d | $open | index($d)))) as $unmet
      | {
          number: $i.number,
          title:  $i.title,
          type:   ([$L[] | select(. == ("feat","fix","refactor","docs","chore","test","ci"))] | first // "-"),
          breaking: (($L | index("breaking")) != null),
          claimed:  ((($L | index("in-progress")) != null) or (($L | index("in-review")) != null)),
          deps:     $deps,
          unmet:    $unmet
        }
      | .status = (if .claimed then "CLAIMED" elif (.unmet | length) > 0 then "BLOCKED" else "READY" end)
    )
  | sort_by(if .status=="READY" then 0 elif .status=="BLOCKED" then 1 else 2 end, .number)
')"

# Workable units only: README §1 requires a type label, so an untyped issue is the epic or an authoring
# defect — never a wave candidate. Surface the count rather than hiding it silently.
WORKABLE="$(printf '%s' "$ENRICHED" | jq '[.[] | select(.type != "-")]')"
UNTYPED=$(printf '%s' "$ENRICHED" | jq '[.[] | select(.type == "-")] | length')

if [ "$MODE" = "json" ]; then
  printf '%s\n' "$WORKABLE"
  exit 0
fi

# Compact table — NO bodies. `BLOCKED-BY` shows the unmet deps so BLOCKED is self-explaining.
printf '%s' "$WORKABLE" | jq -r '
  (["NUM","TYPE","BRK","STATUS","BLOCKED-BY","TITLE"] | @tsv),
  (.[] | [
    "#\(.number)",
    .type,
    (if .breaking then "!" else "-" end),
    .status,
    (if (.unmet|length)>0 then (.unmet | map("#"+(.|tostring)) | join(",")) else "-" end),
    (.title | if length>60 then .[0:57]+"..." else . end)
  ] | @tsv)
' | column -t -s $'\t'

READY=$(printf '%s' "$WORKABLE" | jq '[.[] | select(.status=="READY")] | length')
echo
echo "READY: $READY  |  BLOCKED: $(printf '%s' "$WORKABLE" | jq '[.[]|select(.status=="BLOCKED")]|length')  |  CLAIMED: $(printf '%s' "$WORKABLE" | jq '[.[]|select(.status=="CLAIMED")]|length')${UNTYPED:+  |  untyped/epic hidden: $UNTYPED}"
[ "$READY" -eq 0 ] && echo "No READY issues — nothing to launch (everything blocked or claimed)." >&2 || true
