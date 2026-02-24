const add = (a: number, b: number): number => a + b;

const greet = (name: string): string => {
  return `Hello, ${name}!`;
};

export const handler = async (req: Request): Promise<Response> => {
  const body = await req.json();
  return new Response(JSON.stringify(body));
};

const identity = <T>(value: T): T => value;

export const multiply = (x: number, y: number): number => x * y;
