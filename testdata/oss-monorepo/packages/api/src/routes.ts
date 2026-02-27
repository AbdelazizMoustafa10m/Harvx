/**
 * API route definitions for users and jobs.
 */

import { Router, Request, Response } from 'express';
import { z } from 'zod';
import { isValidEmail, clamp, CONSTANTS, createError } from '@acme/core';
import type { User, ApiResponse, PaginatedResult } from '@acme/core';

// ---- User Routes ----

export const userRoutes = Router();

const createUserSchema = z.object({
  email: z.string().email(),
  username: z.string().min(3).max(CONSTANTS.MAX_USERNAME_LENGTH),
  password: z.string().min(CONSTANTS.MIN_PASSWORD_LENGTH),
  displayName: z.string().optional(),
});

const users: Map<string, User> = new Map();
let nextUserId = 1;

userRoutes.get('/', (req: Request, res: Response) => {
  const page = clamp(parseInt(req.query.page as string) || 1, 1, 1000);
  const perPage = clamp(parseInt(req.query.perPage as string) || CONSTANTS.DEFAULT_PAGE_SIZE, 1, CONSTANTS.MAX_PAGE_SIZE);

  const allUsers = Array.from(users.values());
  const start = (page - 1) * perPage;
  const items = allUsers.slice(start, start + perPage);

  const result: PaginatedResult<User> = {
    items,
    total: allUsers.length,
    page,
    perPage,
    hasNext: start + perPage < allUsers.length,
    hasPrev: page > 1,
  };

  const response: ApiResponse<PaginatedResult<User>> = { success: true, data: result };
  res.json(response);
});

userRoutes.post('/', (req: Request, res: Response) => {
  const parsed = createUserSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json(createError('VALIDATION_ERROR', 'Invalid input', parsed.error.issues));
    return;
  }

  const { email, username, displayName } = parsed.data;

  if (!isValidEmail(email)) {
    res.status(400).json(createError('VALIDATION_ERROR', 'Invalid email format'));
    return;
  }

  const id = String(nextUserId++);
  const now = new Date().toISOString();
  const user: User = {
    id,
    email,
    username,
    role: 'viewer',
    displayName: displayName || username,
    createdAt: now,
    updatedAt: now,
  };

  users.set(id, user);
  const response: ApiResponse<User> = { success: true, data: user };
  res.status(201).json(response);
});

// ---- Job Routes ----

export const jobRoutes = Router();

jobRoutes.get('/', (_req: Request, res: Response) => {
  res.json({ success: true, data: { items: [], total: 0 } });
});

jobRoutes.post('/', (req: Request, res: Response) => {
  const { type, payload } = req.body;
  if (!type) {
    res.status(400).json(createError('VALIDATION_ERROR', 'Job type is required'));
    return;
  }

  const job = {
    id: `job_${Date.now()}`,
    type,
    payload: payload || {},
    status: 'queued' as const,
    attempts: 0,
    maxAttempts: 3,
    createdAt: new Date().toISOString(),
  };

  res.status(201).json({ success: true, data: job });
});