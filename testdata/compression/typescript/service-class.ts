import { Injectable } from '@nestjs/common';
import { Database } from './database';
import type { Logger } from './logger';

/** User service handles all user-related operations */
@Injectable()
class UserService extends BaseService {
  private db: Database;
  private logger: Logger;

  constructor(db: Database, logger: Logger) {
    super();
    this.db = db;
    this.logger = logger;
  }

  async getUser(id: string): Promise<User> {
    this.logger.info(`Fetching user ${id}`);
    return this.db.findOne('users', id);
  }

  async updateUser(id: string, data: Partial<User>): Promise<User> {
    this.logger.info(`Updating user ${id}`);
    return this.db.update('users', id, data);
  }

  private validate(data: unknown): boolean {
    return data !== null && typeof data === 'object';
  }

  get serviceName(): string {
    return 'UserService';
  }
}

export { UserService };
