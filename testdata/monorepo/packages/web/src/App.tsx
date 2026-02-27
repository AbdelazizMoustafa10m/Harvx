import React from "react";

interface AppProps {
  title?: string;
}

export function App({ title = "Monorepo Web" }: AppProps): JSX.Element {
  return (
    <div className="app">
      <h1>{title}</h1>
      <p>Welcome to the monorepo web application.</p>
    </div>
  );
}

export default App;