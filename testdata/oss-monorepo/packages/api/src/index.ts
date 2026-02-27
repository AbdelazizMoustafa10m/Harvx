/**
 * @acme/api - Express REST API server for the Acme Platform.
 */

import express from 'express';
import cors from 'cors';
import helmet from 'helmet';
import { createError } from '@acme/core';
import { userRoutes, jobRoutes } from './routes';
import { requestLogger } from './middleware';

const app = express();
const PORT = process.env.PORT || 3001;

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());
app.use(requestLogger);

// Routes
app.use('/api/users', userRoutes);
app.use('/api/jobs', jobRoutes);

// Health check
app.get('/health', (_req, res) => {
  res.json({ status: 'ok', uptime: process.uptime() });
});

// 404 handler
app.use((_req, res) => {
  res.status(404).json(createError('NOT_FOUND', 'Resource not found'));
});

// Error handler
app.use((err: Error, _req: express.Request, res: express.Response, _next: express.NextFunction) => {
  console.error('Unhandled error:', err);
  res.status(500).json(createError('INTERNAL_ERROR', 'An unexpected error occurred'));
});

app.listen(PORT, () => {
  console.log(`API server running on port ${PORT}`);
});