// RCON command blocklist — prevents scripts from running dangerous commands.

const DEFAULT_BLOCKED = [
  "op", "deop", "stop", "ban", "ban-ip",
  "pardon", "pardon-ip", "whitelist",
  "save-off", "save-all", "save-on",
];

export class CommandFilter {
  private blocked: Set<string>;

  constructor(blocklist?: string[]) {
    const env = process.env.RCON_BLOCKED_COMMANDS;
    if (env !== undefined) {
      // Explicit env var: empty string means allow all, otherwise comma-separated
      this.blocked = new Set(
        env === "" ? [] : env.split(",").map((c) => c.trim().toLowerCase()),
      );
    } else if (blocklist) {
      this.blocked = new Set(blocklist.map((c) => c.toLowerCase()));
    } else {
      this.blocked = new Set(DEFAULT_BLOCKED);
    }
  }

  isAllowed(cmd: string): boolean {
    const root = cmd.trim().split(/\s+/)[0]?.replace(/^\//, "").toLowerCase();
    if (!root) return true;
    if (this.blocked.has(root)) {
      console.warn(`[mc-scripts] Blocked RCON command: ${cmd}`);
      return false;
    }
    return true;
  }

  block(command: string): void {
    this.blocked.add(command.toLowerCase());
  }

  unblock(command: string): void {
    this.blocked.delete(command.toLowerCase());
  }

  setBlocklist(commands: string[]): void {
    this.blocked = new Set(commands.map((c) => c.toLowerCase()));
  }

  get blockedCommands(): string[] {
    return Array.from(this.blocked);
  }
}
