#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -ne 1 ]; then printf 'Usage: bash deploy/upload.sh /absolute/path/release.tgz\n' >&2; exit 1; fi
artifact="$1"
release="$(basename "$artifact" .tgz)"
if [[ ! "$release" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{12}$ ]]; then printf 'Invalid release name\n' >&2; exit 1; fi
ops_dir="$(cd "$(dirname "$0")" && pwd)"
key="${DEPLOY_KEY:-$HOME/.ssh/id_rsa}"
target="${DEPLOY_TARGET:-root@start.deey.top}"
port="${DEPLOY_PORT:-22}"
checksum="$(shasum -a 256 "$artifact" | awk '{print $1}')"
remote_archive="/var/tmp/grade5-math-$release.tgz"
scp -i "$key" -o IdentitiesOnly=yes -P "$port" "$artifact" "$target:$remote_archive"
ssh -i "$key" -o IdentitiesOnly=yes -p "$port" "$target" "bash -s -- '$remote_archive' '$checksum' '$release'" < "$ops_dir/activate.sh"
