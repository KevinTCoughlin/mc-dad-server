// RCON protocol client using Bun native TCP sockets.
// Implements the Source RCON protocol: https://developer.valvesoftware.com/wiki/Source_RCON_Protocol

import type { Socket } from "bun";

const PACKET_TYPE_AUTH = 3;
const PACKET_TYPE_AUTH_RESPONSE = 2;
const PACKET_TYPE_COMMAND = 2;
const PACKET_TYPE_RESPONSE = 0;

interface PendingRequest {
  resolve: (value: string) => void;
  reject: (reason: Error) => void;
  data: Buffer;
}

export class RconClient {
  private socket: Socket | null = null;
  private requestId = 0;
  private pending = new Map<number, PendingRequest>();
  private authenticated = false;
  private buffer = Buffer.alloc(0);

  constructor(
    private host: string,
    private port: number,
    private password: string,
  ) {}

  async connect(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      const self = this;
      let authResolve = resolve;
      let authReject = reject;

      Bun.connect({
        hostname: self.host,
        port: self.port,
        socket: {
          open(socket) {
            self.socket = socket;
            // Send auth packet
            const id = self.nextId();
            const packet = self.encodePacket(id, PACKET_TYPE_AUTH, self.password);
            self.pending.set(id, {
              resolve: () => {
                self.authenticated = true;
                authResolve();
              },
              reject: authReject,
              data: Buffer.alloc(0),
            });
            socket.write(packet);
          },
          data(socket, data) {
            self.buffer = Buffer.concat([self.buffer, Buffer.from(data)]);
            self.processBuffer();
          },
          error(socket, error) {
            authReject(error);
          },
          close() {
            self.socket = null;
            self.authenticated = false;
            // Reject all pending requests
            for (const [, req] of self.pending) {
              req.reject(new Error("RCON connection closed"));
            }
            self.pending.clear();
          },
        },
      });
    });
  }

  async command(cmd: string): Promise<string> {
    if (!this.socket || !this.authenticated) {
      throw new Error("RCON not connected");
    }

    return new Promise<string>((resolve, reject) => {
      const id = this.nextId();
      const packet = this.encodePacket(id, PACKET_TYPE_COMMAND, cmd);
      this.pending.set(id, { resolve, reject, data: Buffer.alloc(0) });
      this.socket!.write(packet);
    });
  }

  disconnect(): void {
    if (this.socket) {
      this.socket.end();
      this.socket = null;
    }
    this.authenticated = false;
  }

  get isConnected(): boolean {
    return this.authenticated && this.socket !== null;
  }

  private nextId(): number {
    return ++this.requestId;
  }

  private encodePacket(id: number, type: number, body: string): Buffer {
    const bodyBuf = Buffer.from(body, "utf-8");
    // Packet: 4 (size) + 4 (id) + 4 (type) + body + 2 (null terminators)
    const size = 4 + 4 + bodyBuf.length + 2;
    const packet = Buffer.alloc(4 + size);
    packet.writeInt32LE(size, 0);
    packet.writeInt32LE(id, 4);
    packet.writeInt32LE(type, 8);
    bodyBuf.copy(packet, 12);
    // Two null bytes at the end (body terminator + packet terminator)
    packet[12 + bodyBuf.length] = 0;
    packet[13 + bodyBuf.length] = 0;
    return packet;
  }

  private processBuffer(): void {
    while (this.buffer.length >= 4) {
      const size = this.buffer.readInt32LE(0);
      const totalPacketSize = 4 + size;

      if (this.buffer.length < totalPacketSize) break;

      const id = this.buffer.readInt32LE(4);
      const type = this.buffer.readInt32LE(8);
      const body = this.buffer.subarray(12, 12 + size - 10).toString("utf-8");

      this.buffer = this.buffer.subarray(totalPacketSize);

      // Auth response
      if (type === PACKET_TYPE_AUTH_RESPONSE) {
        const req = this.pending.get(id) ?? this.pending.get(id - 1);
        if (req) {
          this.pending.delete(id);
          this.pending.delete(id - 1);
          if (id === -1) {
            req.reject(new Error("RCON authentication failed"));
          } else {
            req.resolve(body);
          }
        }
        continue;
      }

      // Command response
      if (type === PACKET_TYPE_RESPONSE) {
        const req = this.pending.get(id);
        if (req) {
          this.pending.delete(id);
          req.resolve(body);
        }
      }
    }
  }
}
