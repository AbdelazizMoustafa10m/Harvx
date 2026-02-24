import { NextRequest, NextResponse } from 'next/server';

interface ApiResponse<T> {
  data: T;
  error?: string;
  timestamp: number;
}

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE';

export async function GET(request: NextRequest): Promise<NextResponse> {
  const params = request.nextUrl.searchParams;
  const id = params.get('id');

  if (!id) {
    return NextResponse.json({ error: 'Missing id' }, { status: 400 });
  }

  const data = await fetchData(id);
  return NextResponse.json({ data, timestamp: Date.now() });
}

export async function POST(request: NextRequest): Promise<NextResponse> {
  const body = await request.json();
  const result = await createItem(body);
  return NextResponse.json({ data: result, timestamp: Date.now() });
}

async function fetchData(id: string): Promise<Record<string, unknown>> {
  // Implementation omitted
  return {};
}

async function createItem(data: unknown): Promise<{ id: string }> {
  // Implementation omitted
  return { id: 'new-id' };
}

const API_VERSION = 'v2';
