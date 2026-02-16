import { greet } from "./app";

describe("greet", () => {
  it("should greet by name", () => {
    expect(greet("World")).toBe("Hello, World!");
  });
});
