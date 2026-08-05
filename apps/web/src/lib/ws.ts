import type { ClientMessage, ServerMessage, SignalMessage } from './game';

export interface Socket {
  readyState: number;
  send(data: string): void;
  close(): void;
  onopen: (() => void) | null;
  onmessage: ((event: { data: string }) => void) | null;
  onclose: (() => void) | null;
}

const OPEN = 1;

class NativeSocket implements Socket {
  private readonly ws: WebSocket;

  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onclose: (() => void) | null = null;

  constructor(url: string) {
    this.ws = new WebSocket(url);
    this.ws.onopen = () => this.onopen?.();
    this.ws.onmessage = (event) => this.onmessage?.({ data: String(event.data) });
    this.ws.onclose = () => this.onclose?.();
  }

  get readyState(): number {
    return this.ws.readyState;
  }

  send(data: string): void {
    this.ws.send(data);
  }

  close(): void {
    this.ws.close();
  }
}

export function wsUrl(roomId: string, playerId: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
  return `${protocol}://${window.location.host}/api/v1/rooms/${roomId}/ws?playerId=${playerId}`;
}

export interface GameClientOptions {
  createSocket?: (url: string) => Socket;
  reconnectMs?: number;
  onOpen?: () => void;
  onClose?: () => void;
  onSignal?: (message: SignalMessage) => void;
}

export class GameClient {
  private socket: Socket | null = null;
  private timer: number | null = null;
  private readonly createSocket: (url: string) => Socket;
  private readonly reconnectMs: number;

  constructor(
    private readonly url: string,
    private readonly onMessage: (message: ServerMessage) => void,
    private readonly options: GameClientOptions = {},
  ) {
    this.createSocket = options.createSocket ?? ((url) => new NativeSocket(url));
    this.reconnectMs = options.reconnectMs ?? 2000;
  }

  connect(): void {
    const socket = this.createSocket(this.url);
    this.socket = socket;
    socket.onopen = () => this.options.onOpen?.();
    socket.onmessage = (event) => {
      const message = JSON.parse(event.data) as ServerMessage;
      if (message.type === 'signal') {
        this.options.onSignal?.(message);
      } else {
        this.onMessage(message);
      }
    };
    socket.onclose = () => {
      this.options.onClose?.();
      this.scheduleReconnect();
    };
  }

  send(message: ClientMessage): void {
    if (this.socket && this.socket.readyState === OPEN) {
      this.socket.send(JSON.stringify(message));
    }
  }

  close(): void {
    if (this.timer !== null) {
      window.clearTimeout(this.timer);
      this.timer = null;
    }
    this.socket?.close();
    this.socket = null;
  }

  private scheduleReconnect(): void {
    if (this.socket === null) {
      return;
    }
    this.timer = window.setTimeout(() => this.connect(), this.reconnectMs);
  }
}
