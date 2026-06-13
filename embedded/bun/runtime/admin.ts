import type { McServer } from "./server";
import { StatsStore } from "./stats";

interface AdminActionRequest {
  action: "start" | "stop" | "say" | "whitelist";
  message?: string;
  player?: string;
  mode?: "on" | "off" | "list" | "reload" | "add" | "remove";
}

interface SseClient {
  send: (event: string, payload: unknown) => void;
  close: () => void;
}

export class AdminControlPlane {
  private readonly host = process.env.MC_ADMIN_HOST ?? "127.0.0.1";
  private readonly port = parseInt(process.env.MC_ADMIN_PORT ?? "8080", 10);
  private readonly logBuffer: string[] = [];
  private readonly maxLogLines = 400;
  private readonly clients = new Set<SseClient>();
  private serverRunning = false;
  private rconReady = false;
  private started = false;
  private server: Bun.Server | null = null;

  constructor(
    private readonly mc: McServer,
    private readonly stats: StatsStore,
  ) {
    this.wireEvents();
  }

  start(): void {
    if (this.started) {
      return;
    }
    this.started = true;

    this.server = Bun.serve({
      hostname: this.host,
      port: this.port,
      fetch: (req) => this.handleRequest(req),
    });
    console.log(`[mc-admin] Dashboard running at http://${this.host}:${this.port}`);
  }

  recordLog(line: string): void {
    if (!line.trim()) {
      return;
    }
    this.logBuffer.push(line);
    if (this.logBuffer.length > this.maxLogLines) {
      this.logBuffer.shift();
    }
    this.broadcast("log", { line, timestamp: new Date().toISOString() });
  }

  stop(): void {
    for (const client of this.clients) {
      client.close();
    }
    this.clients.clear();
    this.server?.stop();
    this.server = null;
    this.stats.close();
  }

  private wireEvents(): void {
    this.mc.on("serverStart", (e) => {
      this.serverRunning = true;
      this.stats.recordServerStart(e.timestamp);
      this.broadcastStatus();
    });
    this.mc.on("serverStop", (e) => {
      this.serverRunning = false;
      this.stats.recordServerStop(e.timestamp);
      this.broadcastStatus();
    });
    this.mc.on("rconReady", () => {
      this.rconReady = true;
      this.serverRunning = true;
      this.broadcastStatus();
    });
    this.mc.on("playerJoin", (e) => {
      this.stats.recordPlayerJoin(e.player, e.timestamp);
      this.broadcastStatus();
    });
    this.mc.on("playerLeave", (e) => {
      this.stats.recordPlayerLeave(e.player, e.timestamp);
      this.broadcastStatus();
    });
  }

  private async handleRequest(req: Request): Promise<Response> {
    const url = new URL(req.url);

    if (req.method === "GET" && (url.pathname === "/" || url.pathname === "/dashboard")) {
      return new Response(this.dashboardHtml(), {
        headers: { "Content-Type": "text/html; charset=utf-8" },
      });
    }

    if (req.method === "GET" && url.pathname === "/api/status") {
      return this.jsonResponse(this.statusPayload());
    }

    if (req.method === "GET" && url.pathname === "/api/logs/stream") {
      return this.sseResponse();
    }

    if (req.method === "POST" && url.pathname === "/api/rcon") {
      return this.handleAction(req);
    }

    if (req.method === "GET" && url.pathname === "/healthz") {
      return this.jsonResponse({ ok: true });
    }

    return new Response("Not found", { status: 404 });
  }

  private async handleAction(req: Request): Promise<Response> {
    let body: AdminActionRequest;
    try {
      body = (await req.json()) as AdminActionRequest;
    } catch {
      return this.jsonResponse({ ok: false, error: "invalid JSON body" }, 400);
    }

    try {
      switch (body.action) {
        case "start":
          return this.handleStartAction();
        case "stop": {
          const response = await this.mc.command("stop");
          return this.jsonResponse({ ok: true, response });
        }
        case "say": {
          const message = body.message?.trim() ?? "";
          if (!message || message.length > 256 || message.includes("\n") || message.includes("\r")) {
            return this.jsonResponse({ ok: false, error: "invalid message" }, 400);
          }
          const response = await this.mc.say(message);
          return this.jsonResponse({ ok: true, response });
        }
        case "whitelist": {
          const command = this.buildWhitelistCommand(body);
          if (!command) {
            return this.jsonResponse({ ok: false, error: "invalid whitelist action" }, 400);
          }
          const response = await this.mc.command(command);
          return this.jsonResponse({ ok: true, response });
        }
        default:
          return this.jsonResponse({ ok: false, error: "unsupported action" }, 400);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "command failed";
      return this.jsonResponse({ ok: false, error: message }, 500);
    }
  }

  private handleStartAction(): Response {
    const startCmd = (process.env.MC_ADMIN_START_CMD ?? "").trim();
    if (!startCmd) {
      return this.jsonResponse(
        {
          ok: false,
          error: "start command not configured (set MC_ADMIN_START_CMD)",
        },
        501,
      );
    }

    if (this.serverRunning) {
      return this.jsonResponse({ ok: true, response: "Server already running" });
    }

    const parts = startCmd.split(/\s+/).filter((part) => part.length > 0);
    if (parts.length === 0) {
      return this.jsonResponse({ ok: false, error: "invalid start command" }, 400);
    }

    Bun.spawn(parts, {
      cwd: process.env.MC_SERVER_DIR,
      stdout: "ignore",
      stderr: "ignore",
    });
    return this.jsonResponse({ ok: true, response: "Start command launched" });
  }

  private buildWhitelistCommand(body: AdminActionRequest): string {
    const mode = body.mode ?? "list";
    if (mode === "on" || mode === "off" || mode === "list" || mode === "reload") {
      return `whitelist ${mode}`;
    }

    const player = body.player?.trim() ?? "";
    if (!/^[A-Za-z0-9_]{1,32}$/.test(player)) {
      return "";
    }
    if (mode === "remove") {
      return `whitelist remove ${player}`;
    }
    return `whitelist add ${player}`;
  }

  private sseResponse(): Response {
    const encoder = new TextEncoder();
    let client: SseClient | null = null;

    const stream = new ReadableStream<Uint8Array>({
      start: (controller) => {
        const send = (event: string, payload: unknown) => {
          const data = `event: ${event}\ndata: ${JSON.stringify(payload)}\n\n`;
          controller.enqueue(encoder.encode(data));
        };
        const close = () => controller.close();
        client = { send, close };
        this.clients.add(client);

        send("status", this.statusPayload());
        for (const line of this.logBuffer) {
          send("log", { line, timestamp: new Date().toISOString() });
        }
      },
      cancel: () => {
        if (client) {
          this.clients.delete(client);
        }
      },
    });

    return new Response(stream, {
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      },
    });
  }

  private broadcast(event: string, payload: unknown): void {
    for (const client of this.clients) {
      try {
        client.send(event, payload);
      } catch {
        this.clients.delete(client);
      }
    }
  }

  private broadcastStatus(): void {
    this.broadcast("status", this.statusPayload());
  }

  private statusPayload() {
    return {
      serverRunning: this.serverRunning,
      rconConnected: this.rconReady,
      playerCount: this.mc.players.count,
      players: this.mc.players.online.map((p) => ({
        name: p.name,
        joinedAt: p.joinedAt.toISOString(),
      })),
      stats: this.stats.getSummary(),
    };
  }

  private jsonResponse(body: unknown, status = 200): Response {
    return new Response(JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json; charset=utf-8" },
    });
  }

  private dashboardHtml(): string {
    return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>MC Dad Admin</title>
    <style>
      body { font-family: system-ui, sans-serif; margin: 1rem auto; max-width: 960px; padding: 0 1rem; }
      .card { border: 1px solid #ccc; border-radius: 8px; padding: 0.75rem; margin-bottom: 0.75rem; }
      #logs { background: #111; color: #d8ffd8; height: 260px; overflow-y: auto; font-family: monospace; font-size: 12px; padding: 0.5rem; }
      button { margin-right: 0.5rem; margin-top: 0.25rem; }
      input { margin-right: 0.5rem; }
    </style>
  </head>
  <body>
    <h1>MC Dad Admin</h1>
    <div class="card">
      <div id="status">Loading status…</div>
      <div id="stats"></div>
    </div>
    <div class="card">
      <strong>RCON Actions</strong><br />
      <button onclick="sendAction({ action: 'start' })">Start</button>
      <button onclick="sendAction({ action: 'stop' })">Stop</button>
      <br />
      <input id="sayMessage" placeholder="Message to broadcast" />
      <button onclick="sendSay()">Say</button>
      <br />
      <input id="whitelistPlayer" placeholder="Player" />
      <button onclick="sendWhitelist('add')">Whitelist Add</button>
      <button onclick="sendWhitelist('remove')">Whitelist Remove</button>
    </div>
    <div class="card">
      <strong>Live Logs</strong>
      <div id="logs"></div>
    </div>

    <script>
      const statusEl = document.getElementById("status");
      const statsEl = document.getElementById("stats");
      const logsEl = document.getElementById("logs");

      function appendLog(line) {
        const node = document.createElement("div");
        node.textContent = line;
        logsEl.appendChild(node);
        if (logsEl.childElementCount > 600) logsEl.removeChild(logsEl.firstChild);
        logsEl.scrollTop = logsEl.scrollHeight;
      }

      function renderStatus(status) {
        statusEl.textContent = "Running: " + status.serverRunning + " | RCON: " + status.rconConnected + " | Players: " + status.playerCount;
        statsEl.textContent = "Player uptime (s): " + status.stats.totalPlayerSeconds + " | Server uptime (s): " + status.stats.totalServerSeconds +
          " | Player sessions: " + status.stats.playerSessions + " | Server sessions: " + status.stats.serverSessions;
      }

      async function refreshStatus() {
        const res = await fetch("/api/status");
        renderStatus(await res.json());
      }

      async function sendAction(payload) {
        const res = await fetch("/api/rcon", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        const data = await res.json();
        if (!data.ok) alert(data.error || "Action failed");
      }

      function sendSay() {
        const message = document.getElementById("sayMessage").value;
        sendAction({ action: "say", message });
      }

      function sendWhitelist(mode) {
        const player = document.getElementById("whitelistPlayer").value;
        sendAction({ action: "whitelist", mode, player });
      }

      const stream = new EventSource("/api/logs/stream");
      stream.addEventListener("status", (event) => renderStatus(JSON.parse(event.data)));
      stream.addEventListener("log", (event) => appendLog(JSON.parse(event.data).line));
      stream.onerror = () => appendLog("[mc-admin] waiting for stream reconnect...");
      refreshStatus();
      setInterval(refreshStatus, 15000);
    </script>
  </body>
</html>`;
  }
}
