"""User routes for registration and profile management."""

import logging

from fastapi import APIRouter, HTTPException, status

from app.models import UserCreate, UserResponse
from app.services.user_service import UserService

logger = logging.getLogger(__name__)
router = APIRouter()
user_service = UserService()


@router.post("/", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(user_data: UserCreate) -> UserResponse:
    """Register a new user account."""
    existing = await user_service.get_by_email(user_data.email)
    if existing:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="A user with this email already exists",
        )

    user = await user_service.create(user_data)
    logger.info("User created: %s", user.username)
    return user


@router.get("/{user_id}", response_model=UserResponse)
async def get_user(user_id: int) -> UserResponse:
    """Retrieve a user by their ID."""
    user = await user_service.get_by_id(user_id)
    if not user:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"User {user_id} not found",
        )
    return user


@router.get("/", response_model=list[UserResponse])
async def list_users(skip: int = 0, limit: int = 20) -> list[UserResponse]:
    """List all users with pagination."""
    if limit > 100:
        limit = 100
    return await user_service.list_users(skip=skip, limit=limit)