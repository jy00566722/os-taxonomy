#!/usr/bin/env bash
set -euo pipefail
app_dir="$(cd "$(dirname "$0")/.." && pwd)"
repo_dir="$(git -C "$app_dir" rev-parse --show-toplevel)"
if [ -n "$(git -C "$repo_dir" status --porcelain)" ]; then
  printf 'Commit project changes before building a release.\n' >&2
  exit 1
fi
revision="$(git -C "$repo_dir" rev-parse HEAD)"
built_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
release="$(date -u +%Y%m%dT%H%M%SZ)-${revision:0:12}"
pack="$app_dir/deploy/.artifacts/$release"
mkdir -p "$pack/bin" "$pack/ops"
node "$app_dir/scripts/build-content.mjs" --check
cd "$app_dir/frontend"
if [ ! -d node_modules ]; then npm ci --no-audit --no-fund; fi
npm run build
cd "$app_dir"
docker buildx build --platform linux/amd64 -f deploy/Dockerfile.build --target artifact --build-arg "REVISION=$revision" --build-arg "BUILT_AT=$built_at" --output "type=local,dest=$pack/bin" .
cp -R content "$pack/content"
cp -R frontend/dist "$pack/web"
cp deploy/grade5-math.service deploy/start.deey.top.http.conf "$pack/ops/"
node --input-type=module - "$pack" "$release" "$revision" "$built_at" <<'NODE'
import fs from 'node:fs'; import path from 'node:path'; import {createHash} from 'node:crypto';
const [pack,release,revision,builtAt]=process.argv.slice(2);
fs.writeFileSync(path.join(pack,'RELEASE.json'),JSON.stringify({release,revision,builtAt,platform:'linux/amd64',static:true},null,2)+'\n');
const files=[];function walk(dir){for(const e of fs.readdirSync(dir,{withFileTypes:true})){const p=path.join(dir,e.name);if(e.isDirectory())walk(p);else files.push(p);}}walk(pack);
fs.writeFileSync(path.join(pack,'SHA256SUMS'),files.sort().map(p=>createHash('sha256').update(fs.readFileSync(p)).digest('hex')+'  '+path.relative(pack,p)).join('\n')+'\n');
NODE
COPYFILE_DISABLE=1 tar -czf "$pack.tgz" -C "$pack" .
shasum -a 256 "$pack.tgz"
printf 'Release artifact: %s.tgz\n' "$pack"
