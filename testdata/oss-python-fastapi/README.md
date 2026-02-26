# TaskAPI

A RESTful task management API built with FastAPI and SQLAlchemy.

## Features

- CRUD operations for users and tasks
- JWT authentication
- SQLAlchemy ORM with async support
- Pydantic models for request/response validation
- Automatic OpenAPI documentation

## Quick Start

```bash
pip install -r requirements.txt
uvicorn main:app --reload
```

Visit [http://localhost:8000/docs](http://localhost:8000/docs) for interactive API docs.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/users` | Create a new user |
| GET | `/api/users/{id}` | Get user by ID |
| GET | `/api/items` | List all items |
| POST | `/api/items` | Create an item |
| PUT | `/api/items/{id}` | Update an item |
| DELETE | `/api/items/{id}` | Delete an item |

## Testing

```bash
pytest tests/ -v
```

## License

MIT