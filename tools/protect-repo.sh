#!/usr/bin/env bash
# Puts the protection on the public GitHub repository. Run it once, after the repository is public
# (GitHub allows these rules on a free account only for public repositories). It is safe to run again.
#
#   bash tools/protect-repo.sh
#
# What it does:
#   - main: only you (the owner, as admin) can push, merge, force-push or delete it. Everyone else can only open a pull
#     request, and only you can merge it. You push to main as before.
#   - release tags (v...): only you can make, move or delete them, so nobody else can make a release that the app would see.
#   - auto-merge is off, and GitHub Actions is off (the repository has no workflows, so a pull request cannot run code).
#   - secret scanning and push protection: GitHub refuses a push that holds a password or a key, and warns about old ones.
#   - private vulnerability reporting, and Dependabot alerts for the Go and npm packages.
set -euo pipefail
repo="${1:-metalpietie/typewriter}"

if [ "$(gh api "repos/$repo" -q .visibility)" != "public" ]; then
  echo "The repository is not public yet. Make it public first, then run this again." >&2
  exit 1
fi

# GitHub's built-in role "admin" has id 5. Only the owner has it in a personal repository.
admin='{"actor_id": 5, "actor_type": "RepositoryRole", "bypass_mode": "always"}'

ruleset() { # name, then the JSON of the ruleset
  local id
  id=$(gh api "repos/$repo/rulesets" -q ".[] | select(.name == \"$1\") | .id")
  if [ -n "$id" ]; then
    echo "$2" | gh api -X PUT "repos/$repo/rulesets/$id" --input - >/dev/null
    echo "updated: $1"
  else
    echo "$2" | gh api -X POST "repos/$repo/rulesets" --input - >/dev/null
    echo "added: $1"
  fi
}

ruleset "main: only the owner" '{
  "name": "main: only the owner", "target": "branch", "enforcement": "active",
  "conditions": {"ref_name": {"include": ["~DEFAULT_BRANCH"], "exclude": []}},
  "bypass_actors": ['"$admin"'],
  "rules": [{"type": "update"}, {"type": "deletion"}, {"type": "non_fast_forward"}]
}'

ruleset "release tags: only the owner" '{
  "name": "release tags: only the owner", "target": "tag", "enforcement": "active",
  "conditions": {"ref_name": {"include": ["refs/tags/v*"], "exclude": []}},
  "bypass_actors": ['"$admin"'],
  "rules": [{"type": "creation"}, {"type": "update"}, {"type": "deletion"}]
}'

gh api -X PATCH "repos/$repo" -F allow_auto_merge=false >/dev/null && echo "auto-merge: off"
gh api -X PUT "repos/$repo/actions/permissions" -F enabled=false >/dev/null && echo "GitHub Actions: off"

gh api -X PATCH "repos/$repo" --input - >/dev/null <<'JSON'
{"security_and_analysis": {"secret_scanning": {"status": "enabled"}, "secret_scanning_push_protection": {"status": "enabled"}}}
JSON
echo "secret scanning and push protection: on"

gh api -X PUT "repos/$repo/private-vulnerability-reporting" >/dev/null && echo "private vulnerability reporting: on"
gh api -X PUT "repos/$repo/vulnerability-alerts" >/dev/null && echo "Dependabot alerts: on"

echo
echo "People with access (only you should be here):"
gh api "repos/$repo/collaborators" -q '.[] | "  \(.login) (\(.role_name))"'
