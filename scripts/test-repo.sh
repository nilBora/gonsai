#!/usr/bin/env bash
# Creates a throwaway git repo in /tmp and launches gonsai against it.
set -e

REPO=/tmp/gonsai-smoke-$$
BINARY="$(cd "$(dirname "$0")/.." && pwd)/gonsai"

mkdir -p "$REPO"
cd "$REPO"
git init -q
git config user.email "test@test.com"
git config user.name "Test"
git commit --allow-empty -q -m "init"

# Create 8 merged feature branches
for i in $(seq 1 8); do
  git branch "feature/old-$i"
done

# Create 1 unmerged branch with a commit
git checkout -q -b experiment/wip
echo "wip" > wip.txt
git add wip.txt
git commit -q -m "work in progress"
git checkout -q main 2>/dev/null || git checkout -q master

echo "Smoke-test repo created at $REPO"
echo "Launching gonsai..."
"$BINARY"

rm -rf "$REPO"
