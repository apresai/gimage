'use strict';

// Guards the exact regression that shipped @apresai/gimage-mcp broken across
// several releases: the bin wrapper was gitignored, so `npm pack` excluded it
// and the published tarball had no `gimage-mcp` entrypoint. These tests assert
// the published file set directly, with no dependency install required, so they
// can run as a pre-publish gate in CI.

const { test } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const pkgRoot = path.join(__dirname, '..');
const pkg = require(path.join(pkgRoot, 'package.json'));

// Exactly what `npm publish` would upload, honoring `files` + ignore rules.
function packedFiles() {
  const out = execFileSync('npm', ['pack', '--dry-run', '--json'], {
    cwd: pkgRoot,
    encoding: 'utf8',
  });
  return JSON.parse(out)[0].files.map((f) => f.path);
}

test('bin entrypoint points at a wrapper that exists on disk', () => {
  assert.strictEqual(pkg.bin['gimage-mcp'], './gimage-mcp.js');
  assert.ok(
    fs.existsSync(path.join(pkgRoot, 'gimage-mcp.js')),
    'gimage-mcp.js wrapper must exist'
  );
});

test('published tarball ships the bin wrapper and install script', () => {
  const files = packedFiles();
  assert.ok(
    files.includes('gimage-mcp.js'),
    `published files must include gimage-mcp.js, got: ${files.join(', ')}`
  );
  assert.ok(files.includes('scripts/install.js'), 'must include scripts/install.js');
  assert.ok(files.includes('package.json'), 'must include package.json');
});

test('published tarball does NOT bundle the platform binaries', () => {
  const binaries = packedFiles().filter((f) => /(^|\/)gimage(-deploy)?(\.exe)?$/.test(f));
  assert.deepStrictEqual(
    binaries,
    [],
    `large platform binaries must not be published, found: ${binaries.join(', ')}`
  );
});
