const add = (a, b) => a + b;

const greet = (name) => {
  return `Hello, ${name}!`;
};

export const handler = async (req) => {
  const body = await req.json();
  return new Response(JSON.stringify(body));
};

export const multiply = (x, y) => x * y;
