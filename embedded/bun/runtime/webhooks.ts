// HTTP webhook server using Bun.serve with route matching.

import type { WebhookRoute } from "./types";

export class WebhookServer {
  private routes: WebhookRoute[] = [];
  private server: ReturnType<typeof Bun.serve> | null = null;

  addRoute(route: WebhookRoute): void {
    this.routes.push(route);
  }

  start(port: number): void {
    if (this.server) {
      console.warn("[mc-scripts] Webhook server already running");
      return;
    }

    this.server = Bun.serve({
      port,
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

    console.log(`[mc-scripts] Webhook server listening on port ${port}`);
  }

  stop(): void {
    if (this.server) {
      this.server.stop();
      this.server = null;
    }
  }
}
