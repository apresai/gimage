#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const tar = require('tar');

const GITHUB_REPO = 'apresai/gimage';
// npm sets npm_package_version during the install lifecycle (postinstall). When
// this module is require()'d at runtime by the bin wrapper that env var is unset,
// so fall back to the package's own version.
const VERSION = process.env.npm_package_version || require('../package.json').version;

function getPlatformInfo() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows'
  };

  const archMap = {
    x64: 'amd64',
    arm64: 'arm64'
  };

  const mappedPlatform = platformMap[platform];
  const mappedArch = archMap[arch];

  if (!mappedPlatform || !mappedArch) {
    throw new Error(
      `Unsupported platform: ${platform}-${arch}. ` +
      `Install gimage manually from https://github.com/${GITHUB_REPO}/releases ` +
      `or via Homebrew: brew install apresai/tap/gimage`
    );
  }

  return {
    platform: mappedPlatform,
    arch: mappedArch,
    ext: platform === 'win32' ? '.exe' : ''
  };
}

// binaryPath returns where the platform gimage binary lives (or will live) inside
// the package: <package>/bin/gimage[.exe].
function binaryPath() {
  const { ext } = getPlatformInfo();
  return path.join(__dirname, '..', 'bin', `gimage${ext}`);
}

async function downloadBinary() {
  const { platform, arch, ext } = getPlatformInfo();
  const binaryName = `gimage${ext}`;

  // GoReleaser naming format: gimage_VERSION_Platform_arch.tar.gz or .zip for Windows
  const platformCap = platform.charAt(0).toUpperCase() + platform.slice(1);
  let archName = arch;
  if (arch === 'amd64') archName = 'x86_64';

  const extension = platform === 'windows' ? '.zip' : '.tar.gz';
  const tarballName = `gimage_${VERSION}_${platformCap}_${archName}${extension}`;
  const url = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/${tarballName}`;

  const binDir = path.join(__dirname, '..', 'bin');
  const binaryDest = path.join(binDir, binaryName);

  // Create bin directory if it doesn't exist
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  // Progress goes to stderr so it never corrupts the MCP server's stdout
  // JSON-RPC stream when this runs as a lazy download just before `gimage serve`.
  console.error(`Downloading gimage binary for ${platform}-${arch}...`);
  console.error(`URL: ${url}`);

  function extractFrom(response, resolve, reject) {
    const tarPath = path.join(binDir, tarballName);
    const file = fs.createWriteStream(tarPath);
    response.pipe(file);
    file.on('finish', async () => {
      file.close();
      try {
        await tar.x({ file: tarPath, cwd: binDir });
        fs.unlinkSync(tarPath);
        if (platform !== 'windows') {
          fs.chmodSync(binaryDest, 0o755);
        }
        console.error('✓ gimage binary installed successfully');
        resolve();
      } catch (err) {
        reject(err);
      }
    });
    file.on('error', reject);
  }

  return new Promise((resolve, reject) => {
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect (GitHub release assets redirect to a CDN)
        https.get(response.headers.location, (redirectResponse) => {
          if (redirectResponse.statusCode !== 200) {
            reject(new Error(`Download failed with status ${redirectResponse.statusCode} for ${url}`));
            return;
          }
          extractFrom(redirectResponse, resolve, reject);
        }).on('error', reject);
      } else if (response.statusCode === 200) {
        extractFrom(response, resolve, reject);
      } else {
        reject(new Error(`Download failed with status ${response.statusCode} for ${url}`));
      }
    }).on('error', reject);
  });
}

// ensureBinary is idempotent: it returns the path to the platform binary,
// downloading it only if it is not already present. Safe to call from both the
// postinstall hook and the runtime bin wrapper.
async function ensureBinary() {
  const target = binaryPath();
  if (fs.existsSync(target)) {
    return target;
  }
  await downloadBinary();
  return target;
}

async function main() {
  try {
    await ensureBinary();
    console.log('\n✓ Installation complete!');
    console.log('\nTo use with Claude Desktop, add this to your MCP configuration:');
    console.log('\nmacOS: ~/Library/Application Support/Claude/claude_desktop_config.json');
    console.log('Linux: ~/.config/Claude/claude_desktop_config.json');
    console.log('Windows: %APPDATA%\\Claude\\claude_desktop_config.json');
    console.log('\n{');
    console.log('  "mcpServers": {');
    console.log('    "gimage": {');
    console.log('      "command": "npx",');
    console.log('      "args": ["-y", "@apresai/gimage-mcp"]');
    console.log('    }');
    console.log('  }');
    console.log('}');
    console.log('\nBefore using, configure your API keys:');
    console.log('  gimage auth setup gemini');
    console.log('\nFor more information: https://github.com/apresai/gimage');
  } catch (error) {
    // Non-fatal: the bin wrapper will lazily retry the download (or fall back to
    // a system gimage on PATH) on first run, so don't fail the whole install.
    console.error('Postinstall could not download the gimage binary:', error.message);
    console.error('It will be fetched automatically on first run, or install manually:');
    console.error('1. From releases: https://github.com/' + GITHUB_REPO + '/releases');
    console.error('2. Via Homebrew:  brew install apresai/tap/gimage');
  }
}

module.exports = { ensureBinary, downloadBinary, binaryPath, getPlatformInfo };

if (require.main === module) {
  main();
}
