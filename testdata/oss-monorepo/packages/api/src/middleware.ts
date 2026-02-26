/**
 * Express middleware for the Acme API.
 */

import { Request, Response, NextFunction } from 'express';

/**
 * Logs incoming requests with method, path, status code, and duration.
 */
export function requestLogger(req: Request, res: Response, next: NextFunction): void {
  const start = Date.now();

  res.on('finish', () => {
    const duration = Date.now() - start;
    console.log(
      `${req.method} ${req.path} ${res.statusCode} ${duration}ms`
    );
  });

  next();
}

/**
 * Validates that the Content-Type header is application/json for POST/PUT requests.
 */
export function requireJson(req: Request, res: Response, next: NextFunction): void {
  if (['POST', 'PUT', 'PATCH'].includes(req.method)) {
    const contentType = req.headers['content-type'];
    if (!contentType || !contentType.includes('application/json')) {
      res.status(415).json({
        code: 'UNSUPPORTED_MEDIA_TYPE',
        message: 'Content-Type must be application/json',
      });
      return;
    }
  }
  next();
}

/**
 * Rate limiter middleware using a simple in-memory counter.
 */
export function rateLimit(maxRequests: number, windowMs: number) {
  const requests = new Map<string, { count: number; resetAt: number }>();

  return (req: Request, res: Response, next: NextFunction): void => {
    const key = req.ip || 'unknown';
    const now = Date.now();
    const entry = requests.get(key);

    if (!entry || now > entry.resetAt) {
      requests.set(key, { count: 1, resetAt: now + windowMs });
      next();
      return;
    }

    if (entry.count >= maxRequests) {
      res.status(429).json({
        code: 'RATE_LIMITED',
        message: 'Too many requests',
        retryAfter: Math.ceil((entry.resetAt - now) / 1000),
      });
      return;
    }

    entry.count++;
    next();
  };
}