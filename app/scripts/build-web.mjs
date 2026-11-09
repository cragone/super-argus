// Builds the web client and writes it into internal/webui/dist so `go build` embeds it.
// vite.config.ts points build.outDir at that path directly (with emptyOutDir), so this
// is just a thin wrapper that fails loudly if the build didn't actually produce output.
import { execSync } from 'node:child_process';
import { existsSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const appDir = resolve(here, '..');
const dest = resolve(appDir, '..', 'internal', 'webui', 'dist');

execSync('npx vite build', { cwd: appDir, stdio: 'inherit' });

if (!existsSync(join(dest, 'index.html'))) throw new Error('build produced no index.html');

// vite's emptyOutDir wipes .gitkeep along with everything else; restore it so a fresh
// checkout before any build still has the directory `//go:embed all:dist` needs.
writeFileSync(join(dest, '.gitkeep'), '');

console.log(`web UI written to ${dest}`);
