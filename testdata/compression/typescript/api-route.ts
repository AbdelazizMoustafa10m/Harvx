import { NextRequest, NextResponse } from 'next/server';
import type { User } from '@/types/user';

interface ApiResponse<T> {
  data: T;
  status: number;
  message: string;
}

type RequestHandler = (req: NextRequest) => Promise<NextResponse>;

/** Validates the authorization header */
function validateAuth(req: NextRequest): string | null {
  const header = req.headers.get('authorization');
  if (!header || !header.startsWith('Bearer ')) {
    return null;
  }
  return header.slice(7);
}

/** Handles GET requests for user data */
export async function GET(req: NextRequest): Promise<NextResponse> {
  const token = validateAuth(req);
  if (!token) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }

  const users: User[] = await fetchUsers(token);
  const response: ApiResponse<User[]> = {
    data: users,
    status: 200,
    message: 'OK',
  };
  return NextResponse.json(response);
}

/** Handles POST requests to create a user */
export async function POST(req: NextRequest): Promise<NextResponse> {
  const body = await req.json();
  const user = await createUser(body);
  return NextResponse.json(user, { status: 201 });
}

async function fetchUsers(token: string): Promise<User[]> {
  return [];
}

async function createUser(data: unknown): Promise<User> {
  return data as User;
}

export const config = {
  runtime: 'edge',
};
