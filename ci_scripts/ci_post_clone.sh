#!/bin/sh
set -e

curl -fsSL https://bun.sh/install | bash
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"

cd "$CI_PRIMARY_REPOSITORY_PATH/extension"
bun install --frozen-lockfile
bun run build:safari
