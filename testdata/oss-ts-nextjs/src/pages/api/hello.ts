import type { NextApiRequest, NextApiResponse } from 'next';

interface HelloResponse {
  message: string;
  timestamp: string;
  version: string;
}

interface ErrorResponse {
  error: string;
}

export default function handler(
  req: NextApiRequest,
  res: NextApiResponse<HelloResponse | ErrorResponse>
) {
  if (req.method !== 'GET') {
    res.setHeader('Allow', ['GET']);
    return res.status(405).json({ error: `Method ${req.method} not allowed` });
  }

  const response: HelloResponse = {
    message: 'Hello from DevBlog API!',
    timestamp: new Date().toISOString(),
    version: '1.0.0',
  };

  return res.status(200).json(response);
}