import { execFile } from "node:child_process";
import { createHash } from "node:crypto";
import { access, chmod, mkdir, mkdtemp, readFile, rename, rm, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { join } from "node:path";
import { promisify } from "node:util";

const repository = "Raxuis/ChaosProxy";
const platforms = { darwin: "darwin", linux: "linux", win32: "windows" };
const architectures = { x64: "amd64", arm64: "arm64" };
const execFileAsync = promisify(execFile);

export function releaseAsset(version, platform = process.platform, arch = process.arch) {
  const os = platforms[platform];
  const cpu = architectures[arch];
  if (!os || !cpu) {
    throw new Error(
      `no prebuilt binary for ${platform}/${arch}; install with go install github.com/Raxuis/chaosproxy/cmd/chaosproxy@v${version}`,
    );
  }
  const extension = os === "windows" ? "zip" : "tar.gz";
  return {
    archive: `chaosproxy_${version}_${os}_${cpu}.${extension}`,
    binary: os === "windows" ? "chaosproxy.exe" : "chaosproxy",
  };
}

export function parseChecksums(text) {
  const checksums = new Map();
  for (const line of text.split("\n")) {
    const match = line.trim().match(/^([a-f0-9]{64})\s+\*?(\S+)$/);
    if (match) {
      checksums.set(match[2], match[1]);
    }
  }
  return checksums;
}

export function sha256(buffer) {
  return createHash("sha256").update(buffer).digest("hex");
}

export function verifyChecksum(buffer, archive, checksums) {
  const expected = checksums.get(archive);
  if (!expected) {
    throw new Error(`checksums.txt has no entry for ${archive}`);
  }
  const actual = sha256(buffer);
  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${archive}: expected ${expected}, got ${actual}`);
  }
}

export async function ensureBinary(options = {}) {
  const version = options.version ?? (await packageVersion());
  const baseURL =
    options.baseURL ??
    process.env.CHAOSPROXY_DOWNLOAD_BASE_URL ??
    `https://github.com/${repository}/releases/download/v${version}`;
  const cacheDir = options.cacheDir ?? process.env.CHAOSPROXY_CACHE_DIR ?? defaultCacheDir();
  const { archive, binary } = releaseAsset(version);
  const installDir = join(cacheDir, version);
  const binaryPath = join(installDir, binary);
  if (await exists(binaryPath)) {
    return binaryPath;
  }

  options.onDownload?.(archive);
  const [checksums, archiveBuffer] = await Promise.all([
    download(`${baseURL}/checksums.txt`).then((buffer) => parseChecksums(buffer.toString("utf8"))),
    download(`${baseURL}/${archive}`),
  ]);
  verifyChecksum(archiveBuffer, archive, checksums);

  await mkdir(cacheDir, { recursive: true });
  const staging = await mkdtemp(join(cacheDir, `.${version}-`));
  try {
    const archivePath = join(staging, archive);
    await writeFile(archivePath, archiveBuffer);
    await execFileAsync("tar", ["-xf", archivePath, "-C", staging]);
    await rm(archivePath);
    await chmod(join(staging, binary), 0o755);
    await rename(staging, installDir);
  } catch (error) {
    await rm(staging, { recursive: true, force: true });
    if (await exists(binaryPath)) {
      return binaryPath;
    }
    throw error;
  }
  return binaryPath;
}

async function packageVersion() {
  const manifest = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf8"));
  return manifest.version;
}

async function download(url) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`download of ${url} failed with ${response.status}`);
  }
  return Buffer.from(await response.arrayBuffer());
}

async function exists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

function defaultCacheDir() {
  if (process.env.XDG_CACHE_HOME) {
    return join(process.env.XDG_CACHE_HOME, "chaosproxy");
  }
  if (process.platform === "win32") {
    return join(process.env.LOCALAPPDATA ?? homedir(), "chaosproxy");
  }
  if (process.platform === "darwin") {
    return join(homedir(), "Library", "Caches", "chaosproxy");
  }
  return join(homedir(), ".cache", "chaosproxy");
}
