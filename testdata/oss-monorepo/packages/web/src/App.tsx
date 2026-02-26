import React, { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import type { User, PaginatedResult } from '@acme/core';
import { Button } from './components/Button';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:3001';

function UserList() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchUsers() {
      try {
        const res = await fetch(`${API_BASE}/api/users`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        const data = json.data as PaginatedResult<User>;
        setUsers(data.items);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    }
    fetchUsers();
  }, []);

  if (loading) return <p>Loading users...</p>;
  if (error) return <p className="error">Error: {error}</p>;

  return (
    <div>
      <h2>Users ({users.length})</h2>
      <ul>
        {users.map((user) => (
          <li key={user.id}>
            <strong>{user.displayName}</strong> ({user.email}) - {user.role}
          </li>
        ))}
      </ul>
    </div>
  );
}

function Home() {
  return (
    <div>
      <h1>Acme Platform</h1>
      <p>Welcome to the Acme Platform dashboard.</p>
      <Button variant="primary" onClick={() => console.log('clicked')}>
        Get Started
      </Button>
    </div>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <nav>
        <Link to="/">Home</Link> | <Link to="/users">Users</Link>
      </nav>
      <main>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/users" element={<UserList />} />
        </Routes>
      </main>
    </BrowserRouter>
  );
}