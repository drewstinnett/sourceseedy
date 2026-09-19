#!/usr/bin/env bash
# Tests release-version.sh against throwaway git repos, one per scenario.
# Needs svu on PATH. Run it from anywhere: .github/scripts/release-version_test.sh
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../.." && pwd)
script="$here/release-version.sh"
failed=0

# scenario name, event, ref, starting tag, expected "current next release", then
# the commit subjects to make (a subject starting with "BODY:" adds a footer)
scenario() {
  local name=$1 event=$2 ref=$3 tag=$4 want=$5
  shift 5
  local dir
  dir=$(mktemp -d)
  (
    cd "$dir"
    git init -q -b main
    git config user.email test@example.com
    git config user.name test
    cp "$root/.svu.yml" .
    git commit -q --allow-empty -m "chore: start"
    git tag "$tag"
    for msg in "$@"; do
      if [[ $msg == *"|BODY:"* ]]; then
        git commit -q --allow-empty -m "${msg%%|BODY:*}" -m "${msg#*|BODY:}"
      else
        git commit -q --allow-empty -m "$msg"
      fi
    done
    : >"$dir/output"
    GITHUB_EVENT_NAME=$event GITHUB_REF=$ref GITHUB_OUTPUT="$dir/output" \
      GITHUB_STEP_SUMMARY="$dir/summary" "$script"
  )
  local got
  got=$(sed -e 's/^current=//' -e 's/^next=//' -e 's/^release=//' "$dir/output" | tr '\n' ' ' | sed 's/ $//')
  if [ "$got" = "$want" ]; then
    echo "ok   $name"
  else
    echo "FAIL $name: got '$got', want '$want'"
    failed=1
  fi
  rm -rf "$dir"
}

main=refs/heads/main

scenario "feat on main releases a minor"          push $main v0.2.6 "v0.2.6 v0.3.0 true"  "feat: a thing"
scenario "fix on main releases a patch"           push $main v0.2.6 "v0.2.6 v0.2.7 true"  "fix: a bug"
scenario "feat beats fix"                         push $main v0.2.6 "v0.2.6 v0.3.0 true"  "fix: a bug" "feat: a thing"
scenario "scoped feat"                            push $main v0.2.6 "v0.2.6 v0.3.0 true"  "feat(cmd): a thing"
scenario "docs and chore don't release"           push $main v0.2.6 "v0.2.6 v0.2.6 false" "docs: words" "chore: tidy" "ci: x" "build: y" "refactor: z" "test: t"
scenario "non-conventional commits are ignored"   push $main v0.2.6 "v0.2.6 v0.2.6 false" "Version bumps" "Merge pull request #1 from a/b"
scenario "breaking on 0.x is a minor, not 1.0.0"  push $main v0.2.6 "v0.2.6 v0.3.0 true"  "feat!: breaking"
scenario "breaking fix on 0.x is a minor too"     push $main v0.2.6 "v0.2.6 v0.3.0 true"  "fix!: breaking"
scenario "BREAKING CHANGE footer on 0.x"          push $main v0.2.6 "v0.2.6 v0.3.0 true"  "feat: x|BODY:BREAKING CHANGE: it changed"
scenario "breaking at 1.x is a major"             push $main v1.2.3 "v1.2.3 v2.0.0 true"  "feat!: breaking"
scenario "a pull request only reports"            pull_request refs/pull/1/merge v0.2.6 "v0.2.6 v0.3.0 false" "feat: a thing"
scenario "a push to another branch only reports"  push refs/heads/topic v0.2.6 "v0.2.6 v0.3.0 false" "feat: a thing"
scenario "a hand pushed tag only reports"         push refs/tags/v0.3.0 v0.2.6 "v0.2.6 v0.3.0 false" "feat: a thing"
scenario "nothing new since the tag"              push $main v0.2.6 "v0.2.6 v0.2.6 false"

exit $failed
