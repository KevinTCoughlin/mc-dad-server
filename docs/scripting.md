# Scripting Sidecar (Experimental)

The `--experimental-bun` flag enables a TypeScript/JavaScript scripting sidecar powered by [Bun](https://bun.sh). It runs alongside the Minecraft server, tails the server log for events, and connects via RCON to send commands.

```bash
mc-dad-server install --experimental-bun
```

## Quick Start

Scripts live in `~/minecraft-server/bun-scripts/scripts/`. Drop a `.ts` or `.js` file there and restart the server.

```typescript
// scripts/welcome.ts
declare const mc: import("../runtime/server").McServer;

mc.on("playerJoin", async (e) => {
  await mc.say(`Welcome, ${e.player}!`);
});
```

An example script is deployed on first install. See `bun-scripts/scripts/example.ts` for a full demo.

## Events

Listen for server events with `mc.on()`:

| Event | Payload | When |
|-------|---------|------|
| `playerJoin` | `{ player, timestamp }` | Player joins |
| `playerLeave` | `{ player, timestamp }` | Player leaves |
| `chat` | `{ player, message, timestamp }` | Chat message sent |
| `playerDeath` | `{ player, message, timestamp }` | Player dies |
| `playerAdvancement` | `{ player, advancement, timestamp }` | Advancement earned |
| `serverStart` | `{ timestamp }` | Server finishes starting |
| `serverStop` | `{ timestamp }` | Server begins stopping |
| `rconReady` | `{ timestamp }` | RCON connection established |

## Commands (RCON)

```typescript
await mc.command("time set day");       // raw RCON command
await mc.say("Hello everyone!");        // broadcast message
await mc.kick("player", "reason");      // kick a player
await mc.tp("player", 0, 64, 0);       // teleport
```

All commands flow through the command filter and rate limiter (see [Security](#security) below).

## Scheduler

```typescript
// Run every 30 minutes
mc.scheduler.every(30 * 60_000, async () => {
  if (mc.players.count > 0) {
    await mc.say("Remember to save your builds!");
  }
});

// Run once after 10 seconds
const task = mc.scheduler.after(10_000, () => {
  console.log("Delayed task ran");
});

// Cancel
task.cancel();
```

## Players

```typescript
mc.players.online;           // PlayerInfo[] — currently connected
mc.players.count;            // number
mc.players.isOnline("Steve"); // boolean
```

## Webhooks

Expose HTTP endpoints for external integrations (CI, Discord bots, dashboards):

```typescript
mc.webhooks.addRoute({
  path: "/api/say",
  method: "POST",
  handler: async (req) => {
    const { message } = await req.json();
    await mc.say(message);
    return Response.json({ ok: true });
  },
});

mc.webhooks.start(9090);
```

The webhook server binds to `127.0.0.1` by default. See [Configuration](#configuration) for overrides.

## NPM Packages

User scripts can import npm packages. Add dependencies to `bun-scripts/package.json`:

```bash
cd ~/minecraft-server/bun-scripts
bun add zod
```

Then import normally in your scripts:

```typescript
import { z } from "zod";
```

For private registries (Azure Artifacts, GitHub Packages, Artifactory), add a `.npmrc` or `bunfig.toml` in the `bun-scripts/` directory.

## Security

The sidecar includes layered security hardening. All features are on by default and configurable via environment variables.

### Command Blocklist

Dangerous RCON commands are blocked by default:

`op`, `deop`, `stop`, `ban`, `ban-ip`, `pardon`, `pardon-ip`, `whitelist`, `save-off`, `save-all`, `save-on`

Blocked attempts are logged: `[mc-scripts] Blocked RCON command: ...`

Scripts can adjust the filter at runtime:

```typescript
mc.commandFilter.block("give");
mc.commandFilter.unblock("save-all");
mc.commandFilter.setBlocklist(["op", "deop", "stop"]);
```

### Rate Limiting

RCON commands are rate-limited with a token bucket (default: 20/s, burst 40). Excess commands are dropped with a warning.

### Webhook Binding

The webhook server binds to `127.0.0.1` by default (not `0.0.0.0`). Privileged ports (< 1024) are rejected. Non-localhost binding logs a warning.

### Script Path Validation

The script loader rejects filenames containing `..`, `/`, or `\` and verifies resolved paths stay within the `scripts/` directory.

### Script Integrity

On first load, a `.manifest.json` is generated with SHA-256 hashes of all scripts. On subsequent loads, modified scripts trigger a warning:

```
[mc-scripts] WARNING: Script modified since last manifest: my-script.ts
```

Regenerate the manifest after intentional changes:

```bash
cd ~/minecraft-server/bun-scripts
bun run runtime/index.ts --rehash
```

### Resource Limits

The Bun process runs in a subshell with `ulimit` constraints:

- **512 MB** virtual memory
- **256** max open file descriptors

These limits only affect the scripting sidecar, not the Java server.

### RCON Password

The RCON password is never written to disk in the sidecar's `.env` file. It is extracted at runtime from `server.properties` and passed as an inline environment variable to the Bun process.

## Configuration

All settings are configurable via environment variables in `bun-scripts/.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `RCON_PORT` | `25575` | RCON port |
| `RCON_HOST` | `127.0.0.1` | RCON host |
| `MC_SERVER_DIR` | Server install dir | Path to Minecraft server |
| `RCON_BLOCKED_COMMANDS` | See [blocklist](#command-blocklist) | Comma-separated blocked commands (empty = allow all) |
| `RCON_RATE_LIMIT` | `20` | Max RCON commands per second |
| `RCON_RATE_BURST` | `40` | Rate limiter burst capacity |
| `WEBHOOK_HOST` | `127.0.0.1` | Webhook bind address |
| `WEBHOOK_PORT` | `9090` | Webhook port (overrides script-provided value) |

Environment variables set in the shell take precedence over `.env` file values (Bun built-in behavior).

## File Layout

```
~/minecraft-server/bun-scripts/
  .env                   # Configuration (auto-generated)
  package.json           # Dependencies
  tsconfig.json          # TypeScript config
  runtime/               # Framework (overwritten on upgrade)
    index.ts             # Sidecar bootstrap
    server.ts            # McServer API
    command-filter.ts    # RCON command blocklist
    rate-limiter.ts      # Token bucket rate limiter
    integrity.ts         # Script hash verification
    webhooks.ts          # HTTP webhook server
    events.ts            # Typed event bus
    rcon.ts              # RCON protocol client
    log-parser.ts        # Minecraft log parser
    players.ts           # Online player tracker
    scheduler.ts         # Task scheduling
    types.ts             # Type definitions
  scripts/               # Your scripts (preserved across upgrades)
    example.ts           # Example script
```
