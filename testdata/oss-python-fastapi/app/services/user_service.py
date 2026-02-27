"""User service for managing user accounts."""

import logging
from datetime import datetime

from passlib.context import CryptContext

from app.models import UserCreate, UserResponse

logger = logging.getLogger(__name__)
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

# In-memory store for demo purposes
_users: dict[int, dict] = {}
_next_id = 1


class UserService:
    """Handles user creation, retrieval, and authentication."""

    @staticmethod
    def hash_password(password: str) -> str:
        """Hash a plaintext password using bcrypt."""
        return pwd_context.hash(password)

    @staticmethod
    def verify_password(plain: str, hashed: str) -> bool:
        """Verify a plaintext password against a hashed one."""
        return pwd_context.verify(plain, hashed)

    async def create(self, user_data: UserCreate) -> UserResponse:
        """Create a new user and return the response."""
        global _next_id

        hashed_pw = self.hash_password(user_data.password)
        now = datetime.utcnow()

        user = {
            "id": _next_id,
            "email": user_data.email,
            "username": user_data.username,
            "hashed_password": hashed_pw,
            "created_at": now,
        }
        _users[_next_id] = user
        _next_id += 1

        logger.info("Created user: %s (id=%d)", user["username"], user["id"])
        return UserResponse(**user)

    async def get_by_id(self, user_id: int) -> UserResponse | None:
        """Retrieve a user by their ID."""
        user = _users.get(user_id)
        if not user:
            return None
        return UserResponse(**user)

    async def get_by_email(self, email: str) -> UserResponse | None:
        """Retrieve a user by their email address."""
        for user in _users.values():
            if user["email"] == email:
                return UserResponse(**user)
        return None

    async def list_users(self, skip: int = 0, limit: int = 20) -> list[UserResponse]:
        """List users with pagination."""
        users = list(_users.values())[skip : skip + limit]
        return [UserResponse(**u) for u in users]