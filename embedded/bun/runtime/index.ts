// Sidecar boot: loads .env, creates the global `mc` object, loads user scripts,
// tails the server log for events, and connects RCON.

import { McServer } from "./server";
import { readdir } from "node:fs/promises";
import { join, resolve } from "node:path";
import { watch } from "node:fs";

// Load .env
const envPath = join(import.meta.dir, "..", ".env");
const envFile = Bun.file(envPath);
if (await envFile.exists()) {
  const text = await envFile.text();
  for (const line of text.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const eq = trimmed.indexOf("=");
    if (eq < 0) continue;
    const key = trimmed.slice(0, eq).trim();
    const value = trimmed.slice(eq + 1).trim();
    if (!process.env[key]) {
      process.env[key] = value;
    }
  }
}

const RCON_PASSWORD = process.env.RCON_PASSWORD ?? "";
const RCON_PORT = parseInt(process.env.RCON_PORT ?? "25575", 10);
const RCON_HOST = process.env.RCON_HOST ?? "127.0.0.1";
const MC_SERVER_DIR = process.env.MC_SERVER_DIR ?? resolve(import.meta.dir, "../..");

// Create the mc server instance
const mc = new McServer(RCON_HOST, RCON_PORT, RCON_PASSWORD);

// Expose as global
(globalThis as any).mc = mc;

// --- Load user scripts ---
const scriptsDir = join(import.meta.dir, "..", "scripts");
async function loadScripts() {
  let files: string[];
  try {
    const entries = await readdir(scriptsDir);
    files = entries.filter((f) => f.endsWith(".ts") || f.endsWith(".js")).sort();
  } catch {
    console.log("[mc-scripts] No scripts directory found");
    return;
  }

  for (const file of files) {
    const path = join(scriptsDir, file);
    try {
      await import(path);
      console.log(`[mc-scripts] Loaded script: ${file}`);
    } catch (err) {
      console.error(`[mc-scripts] Failed to load script ${file}:`, err);
    }
  }
}

await loadScripts();

// --- Tail server log from EOF ---
const logPath = join(MC_SERVER_DIR, "logs", "latest.log");

async function tailLog() {
  const file = Bun.file(logPath);
  if (!(await file.exists())) {
    console.log("[mc-scripts] Waiting for server log...");
    // Wait for the log file to appear
    await new Promise<void>((resolve) => {
      const logsDir = join(MC_SERVER_DIR, "logs");
      const interval = setInterval(async () => {
        if (await Bun.file(logPath).exists()) {
          clearInterval(interval);
          resolve();
        }
      }, 1000);
    });
  }

  // Read current size to start tailing from EOF
  const stat = await file.stat();
  let offset = stat?.size ?? 0;
  let partial = "";

  // Poll for new data
  setInterval(async () => {
    try {
      const currentStat = await Bun.file(logPath).stat();
      const currentSize = currentStat?.size ?? 0;

      if (currentSize < offset) {
        // Log was rotated/truncated, restart from beginning
        offset = 0;
        partial = "";
      }

      if (currentSize > offset) {
        const f = Bun.file(logPath);
        const chunk = await f.slice(offset, currentSize).text();
        offset = currentSize;

        const text = partial + chunk;
        const lines = text.split("\n");
        // Last element may be a partial line
        partial = lines.pop() ?? "";

        for (const line of lines) {
          if (line.trim()) {
            mc.logParser.parseLine(line);
          }
        }
      }
    } catch {
      // Log file may be temporarily unavailable during rotation
    }
  }, 250);
}

tailLog();

// --- Connect RCON (with retries) ---
mc.connectRcon();

// --- Graceful shutdown ---
process.on("SIGTERM", () => {
  mc.shutdown();
  process.exit(0);
});
process.on("SIGINT", () => {
  mc.shutdown();
  process.exit(0);
});

console.log("[mc-scripts] Scripting sidecar ready");
