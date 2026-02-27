/**
 * @acme/core - Shared types, constants, and utilities for the Acme Platform.
 */

export type { User, UserRole, CreateUserInput } from './types';
export type { ApiResponse, PaginatedResult, ErrorPayload } from './types';

/** Application-wide constants. */
export const CONSTANTS = {
  MAX_PAGE_SIZE: 100,
  DEFAULT_PAGE_SIZE: 25,
  MAX_USERNAME_LENGTH: 64,
  MIN_PASSWORD_LENGTH: 10,
  SUPPORTED_LOCALES: ['en', 'es', 'fr', 'de', 'ja'] as const,
} as const;

/**
 * Validates an email address using a basic regex pattern.
 * @param email - The email string to validate.
 * @returns true if the email is valid.
 */
export function isValidEmail(email: string): boolean {
  const pattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return pattern.test(email);
}

/**
 * Clamps a number between min and max bounds.
 */
export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

/**
 * Creates a standardized API error payload.
 */
export function createError(code: string, message: string, details?: unknown): ErrorPayload {
  return {
    code,
    message,
    details: details ?? null,
    timestamp: new Date().toISOString(),
  };
}