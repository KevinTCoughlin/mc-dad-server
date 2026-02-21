// Shared type definitions for the mc-dad-server Bun scripting sidecar.

export interface PlayerInfo {
  name: string;
  joinedAt: Date;
}

export interface ChatEvent {
  player: string;
  message: string;
  timestamp: Date;
}

export interface PlayerJoinEvent {
  player: string;
  timestamp: Date;
}

export interface PlayerLeaveEvent {
  player: string;
  timestamp: Date;
}

export interface PlayerDeathEvent {
  player: string;
  message: string;
  timestamp: Date;
}

export interface PlayerAdvancementEvent {
  player: string;
  advancement: string;
  timestamp: Date;
}

export interface ServerStartEvent {
  timestamp: Date;
}

export interface ServerStopEvent {
  timestamp: Date;
}

export interface RconReadyEvent {
  timestamp: Date;
}

export type McEventMap = {
  playerJoin: PlayerJoinEvent;
  playerLeave: PlayerLeaveEvent;
  chat: ChatEvent;
  playerDeath: PlayerDeathEvent;
  playerAdvancement: PlayerAdvancementEvent;
  serverStart: ServerStartEvent;
  serverStop: ServerStopEvent;
  rconReady: RconReadyEvent;
};

export type McEventName = keyof McEventMap;

export interface ScheduledTask {
  id: number;
  cancel: () => void;
}

export interface WebhookRoute {
  path: string;
  method: "GET" | "POST" | "PUT" | "DELETE";
  handler: (req: Request) => Response | Promise<Response>;
}
