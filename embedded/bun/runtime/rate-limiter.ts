// Token bucket rate limiter for RCON commands.

export class RateLimiter {
  private tokens: number;
  private lastRefill: number;
  private maxPerSecond: number;
  private burst: number;

  constructor(maxPerSecond: number, burst: number) {
    this.maxPerSecond = maxPerSecond;
    this.burst = burst;
    this.tokens = burst;
    this.lastRefill = Date.now();
  }

  tryAcquire(): boolean {
    this.refill();
    if (this.tokens >= 1) {
      this.tokens -= 1;
      return true;
    }
    return false;
  }

  private refill(): void {
    const now = Date.now();
    const elapsed = (now - this.lastRefill) / 1000;
    this.tokens = Math.min(this.burst, this.tokens + elapsed * this.maxPerSecond);
    this.lastRefill = now;
  }
}
