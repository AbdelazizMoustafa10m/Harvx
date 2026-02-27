"""FastAPI router for user management endpoints."""

from typing import Optional, List
from fastapi import APIRouter, Depends, HTTPException, Query
from pydantic import BaseModel, Field

router = APIRouter(prefix="/users", tags=["users"])


class UserCreate(BaseModel):
    """Schema for creating a new user."""

    username: str = Field(..., min_length=3, max_length=50)
    email: str = Field(..., pattern=r"^[\w\.-]+@[\w\.-]+\.\w+$")
    full_name: Optional[str] = None
    is_active: bool = True


class UserResponse(BaseModel):
    """Schema for user response."""

    id: int
    username: str
    email: str
    full_name: Optional[str] = None
    is_active: bool

    class Config:
        from_attributes = True


class UserUpdate(BaseModel):
    """Schema for updating a user."""

    email: Optional[str] = None
    full_name: Optional[str] = None
    is_active: Optional[bool] = None


async def get_db():
    """Dependency to get database session."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()


@router.get("/", response_model=List[UserResponse])
async def list_users(
    skip: int = Query(0, ge=0),
    limit: int = Query(100, ge=1, le=1000),
    db=Depends(get_db),
) -> List[UserResponse]:
    """List all users with pagination."""
    users = db.query(User).offset(skip).limit(limit).all()
    return users


@router.get("/{user_id}", response_model=UserResponse)
async def get_user(user_id: int, db=Depends(get_db)) -> UserResponse:
    """Get a user by ID."""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    return user


@router.post("/", response_model=UserResponse, status_code=201)
async def create_user(user: UserCreate, db=Depends(get_db)) -> UserResponse:
    """Create a new user."""
    db_user = User(**user.model_dump())
    db.add(db_user)
    db.commit()
    db.refresh(db_user)
    return db_user


@router.put("/{user_id}", response_model=UserResponse)
async def update_user(
    user_id: int,
    user: UserUpdate,
    db=Depends(get_db),
) -> UserResponse:
    """Update an existing user."""
    db_user = db.query(User).filter(User.id == user_id).first()
    if not db_user:
        raise HTTPException(status_code=404, detail="User not found")
    for field, value in user.model_dump(exclude_unset=True).items():
        setattr(db_user, field, value)
    db.commit()
    db.refresh(db_user)
    return db_user


@router.delete("/{user_id}", status_code=204)
async def delete_user(user_id: int, db=Depends(get_db)) -> None:
    """Delete a user by ID."""
    db_user = db.query(User).filter(User.id == user_id).first()
    if not db_user:
        raise HTTPException(status_code=404, detail="User not found")
    db.delete(db_user)
    db.commit()
