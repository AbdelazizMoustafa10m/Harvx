import express from 'express';
import { authenticate } from './middleware';

const router = express.Router();

/** Get all users */
function getUsers(req, res) {
  const users = db.findAll();
  res.json(users);
}

/** Create a new user */
async function createUser(req, res) {
  const user = await db.create(req.body);
  res.status(201).json(user);
}

/** Delete a user by ID */
function deleteUser(req, res) {
  db.remove(req.params.id);
  res.status(204).send();
}

router.get('/users', authenticate, getUsers);
router.post('/users', authenticate, createUser);
router.delete('/users/:id', authenticate, deleteUser);

export default router;
