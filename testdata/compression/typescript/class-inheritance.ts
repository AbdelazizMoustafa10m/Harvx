import { EventEmitter } from 'events';

interface Loggable {
  log(message: string): void;
}

interface Closeable {
  close(): Promise<void>;
}

abstract class BaseConnection extends EventEmitter {
  protected host: string;
  protected port: number;

  constructor(host: string, port: number) {
    super();
    this.host = host;
    this.port = port;
  }

  abstract connect(): Promise<void>;

  abstract disconnect(): Promise<void>;

  get address(): string {
    return `${this.host}:${this.port}`;
  }
}

class DatabaseConnection extends BaseConnection implements Loggable, Closeable {
  private pool: ConnectionPool;

  constructor(host: string, port: number) {
    super(host, port);
    this.pool = new ConnectionPool();
  }

  async connect(): Promise<void> {
    await this.pool.init(this.address);
    this.emit('connected');
  }

  async disconnect(): Promise<void> {
    await this.pool.drain();
    this.emit('disconnected');
  }

  log(message: string): void {
    console.log(`[DB] ${message}`);
  }

  async close(): Promise<void> {
    await this.disconnect();
  }
}

export { BaseConnection, DatabaseConnection };
