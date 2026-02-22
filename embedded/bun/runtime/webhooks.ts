// HTTP webhook server using Bun.serve with route matching.

import type { WebhookRoute } from "./types";

export class WebhookServer {
  private routes: WebhookRoute[] = [];
  private server: ReturnType<typeof Bun.serve> | null = null;

  addRoute(route: WebhookRoute): void {
    this.routes.push(route);
  }

  start(port?: number): void {
    if (this.server) {
      console.warn("[mc-scripts] Webhook server already running");
      return;
    }

    // Env vars override script-provided values (admin authority)
    const envPort = process.env.WEBHOOK_PORT;
    const resolvedPort = envPort ? parseInt(envPort, 10) : (port ?? 9090);
    const hostname = process.env.WEBHOOK_HOST ?? "127.0.0.1";

    if (resolvedPort < 1024 || resolvedPort > 65535) {
      console.error(`[mc-scripts] Invalid webhook port ${resolvedPort} (must be 1024-65535)`);
      return;
    }

    if (hostname !== "127.0.0.1" && hostname !== "localhost" && hostname !== "::1") {
      console.warn(`[mc-scripts] WARNING: Webhook binding to non-localhost address: ${hostname}`);
    }

    this.server = Bun.serve({
      port: resolvedPort,
      hostname,
      fetch: async (req) => {
        const url = new URL(req.url);
        const method = req.method.toUpperCase();

        for (const route of this.routes) {
          if (route.path === url.pathname && route.method === method) {
            try {
              return await route.handler(req);
            } catch (err) {
              console.error(`[mc-scripts] Webhook handler error (${route.method} ${route.path}):`, err);
              return new Response("Internal Server Error", { status: 500 });
            }
          }
        }

        return new Response("Not Found", { status: 404 });
      },
    });

    console.log(`[mc-scripts] Webhook server listening on ${hostname}:${resolvedPort}`);
  }

  stop(): void {
    if (this.server) {
      this.server.stop();
      this.server = null;
    }
  }
}
