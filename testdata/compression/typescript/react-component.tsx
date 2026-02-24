import React, { useState, useEffect } from 'react';
import type { FC } from 'react';

interface UserProps {
  name: string;
  age: number;
  onUpdate: (user: User) => void;
}

type User = {
  id: string;
  name: string;
  age: number;
};

/** A user profile component */
const UserProfile: FC<UserProps> = ({ name, age, onUpdate }) => {
  const [editing, setEditing] = useState(false);
  const [localName, setLocalName] = useState(name);

  useEffect(() => {
    setLocalName(name);
  }, [name]);

  const handleSave = () => {
    onUpdate({ id: '1', name: localName, age });
    setEditing(false);
  };

  return (
    <div className="profile">
      {editing ? (
        <input value={localName} onChange={(e) => setLocalName(e.target.value)} />
      ) : (
        <span>{name}</span>
      )}
      <button onClick={handleSave}>Save</button>
    </div>
  );
};

export default UserProfile;
