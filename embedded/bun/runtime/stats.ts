import { Database } from "bun:sqlite";

interface ActivePlayerSession {
  id: number;
  joinedAtMs: number;
}

export interface StatsSummary {
  totalPlayerSeconds: number;
  totalServerSeconds: number;
  playerSessions: number;
  serverSessions: number;
}

export class StatsStore {
  private db: Database;
  private activePlayers = new Map<string, ActivePlayerSession>();
  private activeServerSession: { id: number; startedAtMs: number } | null = null;

  constructor(dbPath: string) {
    this.db = new Database(dbPath, { create: true });
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS player_sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        player TEXT NOT NULL,
        joined_at TEXT NOT NULL,
        left_at TEXT,
        duration_seconds INTEGER
      );
      CREATE TABLE IF NOT EXISTS server_sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        started_at TEXT NOT NULL,
        stopped_at TEXT,
        duration_seconds INTEGER
      );
      CREATE INDEX IF NOT EXISTS idx_player_sessions_player ON player_sessions(player);
    `);
  }

  recordPlayerJoin(player: string, at: Date): void {
    if (this.activePlayers.has(player)) {
      return;
    }
    const result = this.db
      .query("INSERT INTO player_sessions (player, joined_at) VALUES (?, ?)")
      .run(player, at.toISOString());
    this.activePlayers.set(player, {
      id: Number(result.lastInsertRowid),
      joinedAtMs: at.getTime(),
    });
  }

  recordPlayerLeave(player: string, at: Date): void {
    const active = this.activePlayers.get(player);
    if (!active) {
      return;
    }
    const durationSeconds = Math.max(0, Math.floor((at.getTime() - active.joinedAtMs) / 1000));
    this.db
      .query("UPDATE player_sessions SET left_at = ?, duration_seconds = ? WHERE id = ?")
      .run(at.toISOString(), durationSeconds, active.id);
    this.activePlayers.delete(player);
  }

  recordServerStart(at: Date): void {
    if (this.activeServerSession !== null) {
      return;
    }
    const result = this.db
      .query("INSERT INTO server_sessions (started_at) VALUES (?)")
      .run(at.toISOString());
    this.activeServerSession = {
      id: Number(result.lastInsertRowid),
      startedAtMs: at.getTime(),
    };
  }

  recordServerStop(at: Date): void {
    if (this.activeServerSession === null) {
      return;
    }
    const durationSeconds = Math.max(0, Math.floor((at.getTime() - this.activeServerSession.startedAtMs) / 1000));
    this.db
      .query("UPDATE server_sessions SET stopped_at = ?, duration_seconds = ? WHERE id = ?")
      .run(at.toISOString(), durationSeconds, this.activeServerSession.id);
    this.activeServerSession = null;
  }

  getSummary(now = new Date()): StatsSummary {
    const playerRow = this.db
      .query(
        "SELECT COALESCE(SUM(duration_seconds), 0) AS total_seconds, COUNT(*) AS sessions FROM player_sessions",
      )
      .get() as { total_seconds: number; sessions: number } | null;
    const serverRow = this.db
      .query(
        "SELECT COALESCE(SUM(duration_seconds), 0) AS total_seconds, COUNT(*) AS sessions FROM server_sessions",
      )
      .get() as { total_seconds: number; sessions: number } | null;

    let totalPlayerSeconds = playerRow?.total_seconds ?? 0;
    for (const active of this.activePlayers.values()) {
      totalPlayerSeconds += Math.max(0, Math.floor((now.getTime() - active.joinedAtMs) / 1000));
    }

    let totalServerSeconds = serverRow?.total_seconds ?? 0;
    if (this.activeServerSession !== null) {
      totalServerSeconds += Math.max(0, Math.floor((now.getTime() - this.activeServerSession.startedAtMs) / 1000));
    }

    return {
      totalPlayerSeconds,
      totalServerSeconds,
      playerSessions: playerRow?.sessions ?? 0,
      serverSessions: serverRow?.sessions ?? 0,
    };
  }

  close(): void {
    this.db.close();
  }
}
