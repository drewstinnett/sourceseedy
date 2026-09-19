#!/usr/bin/env bash
# Works out whether this run should cut a release, and which version it is.
#
# It writes current, next and release to $GITHUB_OUTPUT, and a note to
# $GITHUB_STEP_SUMMARY. release is true only for a push to main that has a
# releasable commit (feat or fix) since the last tag. Anything else, like a PR
# or a tag someone pushed by hand, just reports what the version would be.
#
# Needs svu on PATH and the full history with tags. Reads GITHUB_EVENT_NAME and
# GITHUB_REF, which Actions sets. Without $GITHUB_OUTPUT it prints to stdout, so
# it can be tried locally.
set -euo pipefail

out=${GITHUB_OUTPUT:-/dev/stdout}
summary=${GITHUB_STEP_SUMMARY:-/dev/null}

current=$(svu current)
next=$(svu next)

release=false
if [ "${GITHUB_EVENT_NAME:-}" = push ] && [ "${GITHUB_REF:-}" = refs/heads/main ] && [ "$next" != "$current" ]; then
  release=true
fi

{
  echo "current=$current"
  echo "next=$next"
  echo "release=$release"
} >>"$out"

{
  echo "### Version"
  if [ "$next" = "$current" ]; then
    echo "Nothing to release: no \`feat:\` or \`fix:\` commits since \`$current\`."
  elif [ "$release" = true ]; then
    echo "Releasing \`$next\` (was \`$current\`)."
  else
    echo "Merging this would release \`$next\` (now \`$current\`)."
  fi
} >>"$summary"
