// McServer — central API object exposed as the global `mc` object.

import { EventBus } from "./events";
import { RconClient } from "./rcon";
import { LogParser } from "./log-parser";
import { PlayerTracker } from "./players";
import { Scheduler } from "./scheduler";
import { WebhookServer } from "./webhooks";
import { CommandFilter } from "./command-filter";
import { RateLimiter } from "./rate-limiter";
import type { McEventMap, McEventName } from "./types";

export class McServer {
  readonly events = new EventBus();
  readonly players: PlayerTracker;
  readonly scheduler = new Scheduler();
  readonly webhooks = new WebhookServer();
  readonly logParser: LogParser;
  readonly commandFilter = new CommandFilter();

  private rcon: RconClient;
  private rconHost: string;
  private rconPort: number;
  private rconPassword: string;
  private rateLimiter: RateLimiter;

  constructor(rconHost: string, rconPort: number, rconPassword: string) {
    this.rconHost = rconHost;
    this.rconPort = rconPort;
    this.rconPassword = rconPassword;
    this.rcon = new RconClient(rconHost, rconPort, rconPassword);
    this.players = new PlayerTracker(this.events);
    this.logParser = new LogParser(this.events);

    const rateLimit = parseInt(process.env.RCON_RATE_LIMIT ?? "20", 10);
    const rateBurst = parseInt(process.env.RCON_RATE_BURST ?? "40", 10);
    this.rateLimiter = new RateLimiter(rateLimit, rateBurst);

    console.log(`[mc-scripts] Command filter active, blocking: ${this.commandFilter.blockedCommands.join(", ") || "(none)"}`);
  }

  // --- Event API ---

  on<K extends McEventName>(event: K, handler: (e: McEventMap[K]) => void | Promise<void>): void {
    this.events.on(event, handler);
  }

  // --- RCON API ---

  /** Connect to the Minecraft RCON server with retries. */
  async connectRcon(maxRetries = 30, delayMs = 2000): Promise<boolean> {
    for (let i = 1; i <= maxRetries; i++) {
      try {
        await this.rcon.connect();
        console.log("[mc-scripts] RCON connected");
        this.events.emit("rconReady", { timestamp: new Date() });
        return true;
      } catch (err) {
        if (i < maxRetries) {
          console.log(`[mc-scripts] RCON connection attempt ${i}/${maxRetries} failed, retrying in ${delayMs / 1000}s...`);
          await Bun.sleep(delayMs);
        }
      }
    }
    console.warn("[mc-scripts] RCON connection failed after all retries — running without RCON");
    return false;
  }

  /** Send a raw RCON command. */
  async command(cmd: string): Promise<string> {
    if (!this.rcon.isConnected) {
      console.warn(`[mc-scripts] RCON not connected, cannot run: ${cmd}`);
      return "";
    }
    if (!this.commandFilter.isAllowed(cmd)) {
      return "";
    }
    if (!this.rateLimiter.tryAcquire()) {
      console.warn(`[mc-scripts] Rate limited, dropping command: ${cmd}`);
      return "";
    }
    return this.rcon.command(cmd);
  }

  /** Broadcast a message to all players. */
  async say(message: string): Promise<string> {
    return this.command(`say ${message}`);
  }

  /** Kick a player with an optional reason. */
  async kick(player: string, reason = ""): Promise<string> {
    const cmd = reason ? `kick ${player} ${reason}` : `kick ${player}`;
    return this.command(cmd);
  }

  /** Teleport a player to coordinates. */
  async tp(player: string, x: number, y: number, z: number): Promise<string> {
    return this.command(`tp ${player} ${x} ${y} ${z}`);
  }

  // --- Lifecycle ---

  /** Graceful shutdown. */
  shutdown(): void {
    console.log("[mc-scripts] Shutting down...");
    this.scheduler.cancelAll();
    this.webhooks.stop();
    this.rcon.disconnect();
  }
}
