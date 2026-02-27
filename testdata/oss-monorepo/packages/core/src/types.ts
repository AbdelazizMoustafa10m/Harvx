/**
 * Shared type definitions for the Acme Platform.
 */

/** Available user roles in the system. */
export type UserRole = 'admin' | 'editor' | 'viewer';

/** Represents a user in the system. */
export interface User {
  id: string;
  email: string;
  username: string;
  role: UserRole;
  displayName: string;
  avatarUrl?: string;
  createdAt: string;
  updatedAt: string;
}

/** Input for creating a new user. */
export interface CreateUserInput {
  email: string;
  username: string;
  password: string;
  role?: UserRole;
  displayName?: string;
}

/** Wrapper for all API responses. */
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  meta?: Record<string, unknown>;
}

/** Paginated result set. */
export interface PaginatedResult<T> {
  items: T[];
  total: number;
  page: number;
  perPage: number;
  hasNext: boolean;
  hasPrev: boolean;
}

/** Standardized error payload. */
export interface ErrorPayload {
  code: string;
  message: string;
  details: unknown;
  timestamp: string;
}

/** Job status for the worker service. */
export type JobStatus = 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';

/** Represents a background job. */
export interface Job {
  id: string;
  type: string;
  payload: Record<string, unknown>;
  status: JobStatus;
  attempts: number;
  maxAttempts: number;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  error?: string;
}