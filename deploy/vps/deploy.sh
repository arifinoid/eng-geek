#!/usr/bin/env bash
# Build eng-geek locally and publish a release to the VPS.
#
# The NixOS module (nix/modules/nixos/vps-enggeek.nix in the nixos repo)
# declares Postgres, the systemd units, the nginx vhost and /srv/enggeek.
# This script fills the release directory, flips `current`, runs the
# migration+seed oneshot and restarts the service. It mirrors deploy-adl.sh
# and the nata deploy workflow.
#
# Prerequisites (once per host):
#   1. DNS A record enggeek.arifinoid.web.id -> VPS.
#   2. nixos-rebuild switch --flake ~/Documents/repos/nixos#vps \
#        --target-host root@43.173.26.172
#      (creates the eng-geek user, Postgres role/database, units and the
#       ACME certificate; the service stays down until the first release
#       exists because of its ConditionPathExists.)
#
# Usage:
#   bash deploy/vps/deploy.sh
#
# Environment overrides:
#   VPS_HOST      default root@43.173.26.172
#   KEEP_RELEASES releases kept on the host (default 3)
#   BUN / GO      tool names if they are not on PATH

set -euo pipefail

VPS_HOST="${VPS_HOST:-root@43.173.26.172}"
APP_DIR=/srv/enggeek
API_PORT=8083
PUBLIC_URL="https://enggeek.arifinoid.web.id"
KEEP_RELEASES="${KEEP_RELEASES:-3}"
BUN="${BUN:-bun}"
GO="${GO:-go}"

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

log() { printf '\n==> %s\n' "$*"; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

remote() {
  ssh -o StrictHostKeyChecking=accept-new "$VPS_HOST" \
    "export PATH=/run/current-system/sw/bin:\$PATH; $*"
}

[[ -d "$REPO/apps/api" && -d "$REPO/apps/web" ]] || die "not an eng-geek checkout: $REPO"
cd "$REPO"

# Deploy the commit that is actually checked out. A dirty tree would publish
# uncommitted work under a SHA that does not contain it.
git diff --quiet || die "working tree is dirty; commit or stash before deploying"
SHA="$(git rev-parse --short HEAD)"
[[ -n "$SHA" ]] || die "cannot resolve HEAD"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

log "build web (static export)"
"$BUN" install --frozen-lockfile
"$BUN" run build:web

log "build api (static binaries)"
CGO_ENABLED=0 "$GO" -C apps/api build -o "$WORK/release/server" ./cmd/server
CGO_ENABLED=0 "$GO" -C apps/api build -o "$WORK/release/seed" ./cmd/seed

log "stage release"
mkdir -p "$WORK/release/apps/api"
cp -r apps/web/out "$WORK/release/out"
cp -r apps/api/db/migrations "$WORK/release/apps/api/db/migrations"
cp -r apps/api/content/a1 "$WORK/release/apps/api/content/a1"

ID="${SHA}-$(date -u +%Y%m%d%H%M%S)"
DIR="$APP_DIR/releases/$ID"

log "upload $ID"
remote "mkdir -p '$DIR'"
rsync -az --delete "$WORK/release/" "$VPS_HOST:$DIR/"

log "activate $ID"
ssh -o StrictHostKeyChecking=accept-new "$VPS_HOST" "DIR='$DIR'" bash -s <<REMOTE
set -Eeuo pipefail

# Canonicalize before the flip: a missing target would otherwise resolve to
# /srv/enggeek/current and a rollback would create a symlink loop.
prev="\$(readlink -f $APP_DIR/current 2>/dev/null || true)"
if [ -n "\$prev" ] && [ ! -d "\$prev" ]; then
  prev=""
fi

rollback() {
  if [ -n "\$prev" ] && [ "\$prev" != "$DIR" ]; then
    echo "rolling back to \$prev"
    ln -sfn "\$prev" $APP_DIR/current.new
    mv -Tf $APP_DIR/current.new $APP_DIR/current
    systemctl restart eng-geek || true
  fi
}
trap rollback ERR

ln -sfn '$DIR' $APP_DIR/current.new
mv -Tf $APP_DIR/current.new $APP_DIR/current

# Migrations + content seed run as the eng-geek user (peer auth), so a failed
# migration stops the deploy before the API starts.
systemctl start eng-geek-migrate

if systemctl is-active --quiet eng-geek; then
  systemctl restart eng-geek
else
  systemctl start eng-geek
fi

curl -fsS --retry 15 --retry-delay 2 --retry-connrefused \
  http://127.0.0.1:$API_PORT/healthz >/dev/null

trap - ERR

# current is always the newest release; keep a few for rollback.
ls -1dt $APP_DIR/releases/*/ | tail -n +$((KEEP_RELEASES + 1)) | xargs -r rm -rf
REMOTE

log "verify $PUBLIC_URL"
curl -fsS --retry 20 --retry-delay 5 "$PUBLIC_URL/healthz"

log "done ($ID)"
