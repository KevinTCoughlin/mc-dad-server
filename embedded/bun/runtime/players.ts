// Online player tracker — maintains a list of connected players.

import type { EventBus } from "./events";
import type { PlayerInfo } from "./types";

export class PlayerTracker {
  private playerMap = new Map<string, PlayerInfo>();

  constructor(events: EventBus) {
    events.on("playerJoin", (e) => {
      this.playerMap.set(e.player, { name: e.player, joinedAt: e.timestamp });
    });

    events.on("playerLeave", (e) => {
      this.playerMap.delete(e.player);
    });

    // Clear player list on server stop (stale data)
    events.on("serverStop", () => {
      this.playerMap.clear();
    });
  }

  get online(): PlayerInfo[] {
    return Array.from(this.playerMap.values());
  }

  get count(): number {
    return this.playerMap.size;
  }

  isOnline(name: string): boolean {
    return this.playerMap.has(name);
  }
}
