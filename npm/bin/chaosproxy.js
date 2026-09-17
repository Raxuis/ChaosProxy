#!/usr/bin/env node
import { spawn } from "node:child_process";

import { ensureBinary } from "../lib/install.js";

let binary;
try {
  binary = await ensureBinary({
    onDownload: (archive) => console.error(`chaosproxy: downloading ${archive}`),
  });
} catch (error) {
  console.error(`chaosproxy: ${error.message}`);
  process.exit(1);
}

const child = spawn(binary, process.argv.slice(2), { stdio: "inherit" });
const forward = (signal) => child.kill(signal);
process.on("SIGINT", forward);
process.on("SIGTERM", forward);

child.on("error", (error) => {
  console.error(`chaosproxy: ${error.message}`);
  process.exit(1);
});
child.on("exit", (code, signal) => {
  process.off("SIGINT", forward);
  process.off("SIGTERM", forward);
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code ?? 1);
});
