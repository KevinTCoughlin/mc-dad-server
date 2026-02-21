// Scheduled task wrapper — interval and timeout management with cancellation.

import type { ScheduledTask } from "./types";

export class Scheduler {
  private nextId = 0;
  private tasks = new Map<number, ReturnType<typeof setInterval> | ReturnType<typeof setTimeout>>();

  /** Run a function repeatedly at the given interval (ms). */
  every(ms: number, fn: () => void | Promise<void>): ScheduledTask {
    const id = ++this.nextId;
    const handle = setInterval(async () => {
      try {
        await fn();
      } catch (err) {
        console.error(`[mc-scripts] Scheduled task ${id} error:`, err);
      }
    }, ms);
    this.tasks.set(id, handle);
    return { id, cancel: () => this.cancel(id) };
  }

  /** Run a function once after the given delay (ms). */
  after(ms: number, fn: () => void | Promise<void>): ScheduledTask {
    const id = ++this.nextId;
    const handle = setTimeout(async () => {
      try {
        await fn();
      } catch (err) {
        console.error(`[mc-scripts] Scheduled task ${id} error:`, err);
      }
      this.tasks.delete(id);
    }, ms);
    this.tasks.set(id, handle);
    return { id, cancel: () => this.cancel(id) };
  }

  /** Cancel a specific scheduled task. */
  cancel(id: number): void {
    const handle = this.tasks.get(id);
    if (handle !== undefined) {
      clearInterval(handle as any);
      clearTimeout(handle as any);
      this.tasks.delete(id);
    }
  }

  /** Cancel all scheduled tasks. */
  cancelAll(): void {
    for (const [id] of this.tasks) {
      this.cancel(id);
    }
  }
}
