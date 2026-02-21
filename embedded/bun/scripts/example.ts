// Example mc-dad-server script — demonstrates the scripting API.
// Drop .ts files in this directory and restart the server to load them.

declare const mc: import("../runtime/server").McServer;

// Welcome players
mc.on("playerJoin", async (e) => {
  await mc.say(`Welcome to the server, ${e.player}!`);
});

// Goodbye message
mc.on("playerLeave", async (e) => {
  await mc.say(`${e.player} has left the game. See you next time!`);
});

// Respond to !players command in chat
mc.on("chat", async (e) => {
  if (e.message === "!players") {
    const players = mc.players.online;
    if (players.length === 0) {
      await mc.say("No players online (how are you seeing this?)");
    } else {
      const names = players.map((p) => p.name).join(", ");
      await mc.say(`Online (${players.length}): ${names}`);
    }
  }
});

// Celebrate advancements
mc.on("playerAdvancement", async (e) => {
  await mc.say(`Great job ${e.player}! You earned: ${e.advancement}`);
});

// Log deaths (no chat spam)
mc.on("playerDeath", (e) => {
  console.log(`[death] ${e.message}`);
});

// Periodic server message every 30 minutes
mc.scheduler.every(30 * 60_000, async () => {
  if (mc.players.count > 0) {
    await mc.say("Remember to save your builds! Type !players to see who's online.");
  }
});

// Webhook: POST /api/say — broadcast a message from outside
mc.webhooks.addRoute({
  path: "/api/say",
  method: "POST",
  handler: async (req) => {
    try {
      const body = (await req.json()) as { message?: string };
      if (!body.message) {
        return new Response(JSON.stringify({ error: "missing message" }), {
          status: 400,
          headers: { "Content-Type": "application/json" },
        });
      }
      await mc.say(body.message);
      return new Response(JSON.stringify({ ok: true }), {
        headers: { "Content-Type": "application/json" },
      });
    } catch {
      return new Response(JSON.stringify({ error: "invalid JSON" }), {
        status: 400,
        headers: { "Content-Type": "application/json" },
      });
    }
  },
});

// Webhook: GET /api/players — player list
mc.webhooks.addRoute({
  path: "/api/players",
  method: "GET",
  handler: () => {
    return new Response(
      JSON.stringify({
        count: mc.players.count,
        players: mc.players.online,
      }),
      { headers: { "Content-Type": "application/json" } },
    );
  },
});

// Start webhook server on port 9090
mc.webhooks.start(9090);
