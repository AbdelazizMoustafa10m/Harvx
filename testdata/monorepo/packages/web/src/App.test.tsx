import React from "react";
import { render, screen } from "@testing-library/react";
import { App } from "./App";

describe("App", () => {
  it("renders the title", () => {
    render(<App title="Test App" />);
    expect(screen.getByText("Test App")).toBeInTheDocument();
  });

  it("renders default title", () => {
    render(<App />);
    expect(screen.getByText("Monorepo Web")).toBeInTheDocument();
  });
});