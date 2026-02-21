// Typed EventEmitter with error isolation per handler.

import type { McEventMap, McEventName } from "./types";

type Handler<T> = (event: T) => void | Promise<void>;

export class EventBus {
  private handlers = new Map<string, Handler<any>[]>();

  on<K extends McEventName>(event: K, handler: Handler<McEventMap[K]>): void {
    const list = this.handlers.get(event) ?? [];
    list.push(handler);
    this.handlers.set(event, list);
  }

  off<K extends McEventName>(event: K, handler: Handler<McEventMap[K]>): void {
    const list = this.handlers.get(event);
    if (!list) return;
    const idx = list.indexOf(handler);
    if (idx >= 0) list.splice(idx, 1);
  }

  async emit<K extends McEventName>(event: K, data: McEventMap[K]): Promise<void> {
    const list = this.handlers.get(event);
    if (!list) return;

    for (const handler of list) {
      try {
        await handler(data);
      } catch (err) {
        console.error(`[mc-scripts] Error in ${event} handler:`, err);
      }
    }
  }
}
