// Parses Minecraft server log lines into typed events.

import type { EventBus } from "./events";

// Minecraft log format: [HH:MM:SS] [Thread/LEVEL]: message
const LOG_PREFIX = /^\[\d{2}:\d{2}:\d{2}\] \[.+?\/INFO\]: /;

// Player join: "PlayerName joined the game"
const PLAYER_JOIN = /^(\w+) joined the game$/;

// Player leave: "PlayerName left the game"
const PLAYER_LEAVE = /^(\w+) left the game$/;

// Chat: "<PlayerName> message"
const CHAT = /^<(\w+)> (.+)$/;

// Death messages — covers most vanilla death messages.
// The player name is at the start, followed by a death message verb.
const DEATH_VERBS = [
  "was shot", "was pummeled", "was pricked", "walked into a cactus",
  "drowned", "experienced kinetic energy", "blew up", "was blown up",
  "was killed by", "hit the ground", "fell", "was squashed", "was squished",
  "was stung", "was obliterated", "suffocated", "starved", "was frozen",
  "was burnt", "was roasted", "went up in flames", "burned",
  "tried to swim in lava", "was struck by lightning", "discovered",
  "was fireballed", "was impaled", "didn't want to live",
  "withered away", "died", "was slain", "was killed",
];
const DEATH_PATTERN = new RegExp(
  `^(\\w+) (${DEATH_VERBS.join("|")})(.*)$`
);

// Advancement: "PlayerName has made the advancement [Advancement Name]"
const ADVANCEMENT = /^(\w+) has made the advancement \[(.+)\]$/;

// Server start: "Done (Xs)! For help, type "help""
const SERVER_START = /^Done \([0-9.]+s\)! For help, type "help"$/;

// Server stop: "Stopping the server" or "Closing Server"
const SERVER_STOP = /^(Stopping the server|Closing Server)$/;

export class LogParser {
  constructor(private events: EventBus) {}

  parseLine(line: string): void {
    // Strip the log prefix
    const match = line.match(LOG_PREFIX);
    if (!match) return;
    const msg = line.slice(match[0].length);

    const now = new Date();

    // Player join
    const joinMatch = msg.match(PLAYER_JOIN);
    if (joinMatch) {
      this.events.emit("playerJoin", { player: joinMatch[1], timestamp: now });
      return;
    }

    // Player leave
    const leaveMatch = msg.match(PLAYER_LEAVE);
    if (leaveMatch) {
      this.events.emit("playerLeave", { player: leaveMatch[1], timestamp: now });
      return;
    }

    // Chat
    const chatMatch = msg.match(CHAT);
    if (chatMatch) {
      this.events.emit("chat", { player: chatMatch[1], message: chatMatch[2], timestamp: now });
      return;
    }

    // Death
    const deathMatch = msg.match(DEATH_PATTERN);
    if (deathMatch) {
      this.events.emit("playerDeath", { player: deathMatch[1], message: msg, timestamp: now });
      return;
    }

    // Advancement
    const advMatch = msg.match(ADVANCEMENT);
    if (advMatch) {
      this.events.emit("playerAdvancement", { player: advMatch[1], advancement: advMatch[2], timestamp: now });
      return;
    }

    // Server start
    if (SERVER_START.test(msg)) {
      this.events.emit("serverStart", { timestamp: now });
      return;
    }

    // Server stop
    if (SERVER_STOP.test(msg)) {
      this.events.emit("serverStop", { timestamp: now });
    }
  }
}
