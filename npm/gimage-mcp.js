#!/usr/bin/env node

const { spawn, spawnSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// The platform gimage binary lives beside this wrapper at <package>/bin/gimage.
function packageBinaryPath() {
  const ext = process.platform === 'win32' ? '.exe' : '';
  return path.join(__dirname, 'bin', `gimage${ext}`);
}

// Resolve a system-installed gimage (e.g. via Homebrew) on PATH to its full
// path, or null if absent. Returning the resolved path (rather than the bare
// name) lets spawn launch it directly — important on Windows, where
// spawn('gimage') without a shell does not append .exe / search PATHEXT.
function systemGimagePath() {
  const probe = process.platform === 'win32' ? 'where' : 'which';
  const result = spawnSync(probe, ['gimage'], { encoding: 'utf8' });
  if (result.status !== 0 || !result.stdout) {
    return null;
  }
  // `where` may list several matches; take the first line.
  const first = result.stdout.split(/\r?\n/)[0].trim();
  return first || null;
}

// Resolve the gimage binary, in order of preference:
//   1. the version-pinned binary bundled into this package (downloaded by the
//      postinstall hook),
//   2. a system gimage on PATH (Homebrew, manual install),
//   3. a lazy download into the package bin/ — this covers the case where npm's
//      allow-scripts gate blocked postinstall so the binary was never fetched.
async function resolveBinary() {
  const pkgBinary = packageBinaryPath();
  if (fs.existsSync(pkgBinary)) {
    return pkgBinary;
  }
  const systemBinary = systemGimagePath();
  if (systemBinary) {
    return systemBinary;
  }
  // install.js logs progress to stderr, so this is safe to run right before
  // launching the MCP server (whose stdout is the JSON-RPC channel).
  const { ensureBinary } = require('./scripts/install.js');
  return ensureBinary();
}

async function main() {
  let binaryPath;
  try {
    binaryPath = await resolveBinary();
  } catch (error) {
    console.error('Failed to locate or download the gimage binary:', error.message);
    console.error('\nInstall gimage manually:');
    console.error('  npm install -g @apresai/gimage-mcp');
    console.error('  or');
    console.error('  brew install apresai/tap/gimage');
    console.error('\nSee: https://github.com/apresai/gimage');
    process.exit(1);
  }

  // Forward any args straight through to gimage. With no args, default to the MCP
  // server so `npx -y @apresai/gimage-mcp` launches it for Claude Desktop, while
  // `gimage-mcp --version`, `gimage-mcp serve`, etc. pass through unchanged.
  const forwarded = process.argv.slice(2);
  const args = forwarded.length > 0 ? forwarded : ['serve'];

  const child = spawn(binaryPath, args, {
    stdio: 'inherit',
    env: process.env
  });

  child.on('error', (error) => {
    console.error('Failed to start gimage:', error.message);
    console.error('\nPlease ensure gimage is installed:');
    console.error('  npm install -g @apresai/gimage-mcp');
    console.error('  or');
    console.error('  brew install apresai/tap/gimage');
    process.exit(1);
  });

  child.on('exit', (code) => {
    process.exit(code || 0);
  });

  // Forward termination signals for graceful shutdown.
  process.on('SIGINT', () => child.kill('SIGINT'));
  process.on('SIGTERM', () => child.kill('SIGTERM'));
}

main();
