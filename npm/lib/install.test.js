import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";

import { ensureBinary, parseChecksums, releaseAsset, sha256, verifyChecksum } from "./install.js";

test("releaseAsset maps Node platforms to release archives", () => {
  assert.deepEqual(releaseAsset("1.2.3", "darwin", "arm64"), {
    archive: "chaosproxy_1.2.3_darwin_arm64.tar.gz",
    binary: "chaosproxy",
  });
  assert.deepEqual(releaseAsset("1.2.3", "win32", "x64"), {
    archive: "chaosproxy_1.2.3_windows_amd64.zip",
    binary: "chaosproxy.exe",
  });
  assert.throws(() => releaseAsset("1.2.3", "aix", "ppc64"), /no prebuilt binary for aix\/ppc64/);
});

test("checksums are parsed and enforced", () => {
  const archive = "chaosproxy_1.2.3_linux_amd64.tar.gz";
  const contents = Buffer.from("archive");
  const checksums = parseChecksums(`${sha256(contents)}  ${archive}\n\nnot a checksum line\n`);

  verifyChecksum(contents, archive, checksums);
  assert.throws(() => verifyChecksum(Buffer.from("tampered"), archive, checksums), /checksum mismatch/);
  assert.throws(() => verifyChecksum(contents, "missing.tar.gz", checksums), /no entry for missing\.tar\.gz/);
});

test("ensureBinary downloads, verifies, extracts, and caches the binary", { skip: process.platform === "win32" }, async (t) => {
  const release = await serveRelease(t, { tamper: false });

  const installed = await ensureBinary({ version: "1.2.3", baseURL: release.baseURL, cacheDir: release.cacheDir });
  assert.equal(execFileSync(installed, { encoding: "utf8" }).trim(), "chaosproxy 1.2.3");
  const downloads = release.downloads();

  assert.equal(await ensureBinary({ version: "1.2.3", baseURL: release.baseURL, cacheDir: release.cacheDir }), installed);
  assert.equal(release.downloads(), downloads);
});

test("ensureBinary refuses a tampered archive", { skip: process.platform === "win32" }, async (t) => {
  const release = await serveRelease(t, { tamper: true });

  await assert.rejects(
    ensureBinary({ version: "1.2.3", baseURL: release.baseURL, cacheDir: release.cacheDir }),
    /checksum mismatch/,
  );
  assert.equal(existsSync(join(release.cacheDir, "1.2.3")), false);
});

async function serveRelease(t, { tamper }) {
  const workspace = mkdtempSync(join(tmpdir(), "chaosproxy-npm-"));
  const { archive, binary } = releaseAsset("1.2.3");
  writeFileSync(join(workspace, binary), "#!/bin/sh\necho chaosproxy 1.2.3\n");
  execFileSync("tar", ["-czf", join(workspace, archive), "-C", workspace, binary]);
  const archiveContents = readFileSync(join(workspace, archive));
  const checksums = `${sha256(tamper ? Buffer.from("other") : archiveContents)}  ${archive}\n`;

  let downloads = 0;
  const server = createServer((request, response) => {
    downloads++;
    if (request.url === "/checksums.txt") {
      response.end(checksums);
    } else if (request.url === `/${archive}`) {
      response.end(archiveContents);
    } else {
      response.statusCode = 404;
      response.end();
    }
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  t.after(() => server.close());

  return {
    baseURL: `http://127.0.0.1:${server.address().port}`,
    cacheDir: join(workspace, "cache"),
    downloads: () => downloads,
  };
}
