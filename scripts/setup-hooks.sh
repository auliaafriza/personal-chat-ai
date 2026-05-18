#!/usr/bin/env bash
# One-time setup for git + husky hooks.
# Run after `yarn install`:
#   bash scripts/setup-hooks.sh
set -e

# 1. Init git if not already a repo
if [ ! -d ".git" ]; then
  echo "→ Initializing git repository…"
  git init -b main
fi

# 2. Init husky (creates .husky/ folder)
echo "→ Initializing husky…"
yarn husky

# 3. Write pre-commit hook (lint + type-check)
cat > .husky/pre-commit << 'EOF'
yarn lint && yarn type-check
EOF

# 4. Write commit-msg hook (commitlint)
cat > .husky/commit-msg << 'EOF'
yarn commitlint --edit "$1"
EOF

# 5. Make executable
chmod +x .husky/pre-commit .husky/commit-msg

echo ""
echo "✓ Husky hooks installed."
echo "  • pre-commit  → runs lint + type-check"
echo "  • commit-msg  → enforces Conventional Commits"
echo ""
echo "Test with:"
echo "  git add . && git commit -m 'chore: initial commit'"
