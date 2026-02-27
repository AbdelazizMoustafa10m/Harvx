from fastapi import APIRouter, Depends, HTTPException, status
from pydantic import BaseModel
from typing import Optional, List

router = APIRouter(prefix="/users", tags=["users"])

class UserCreate(BaseModel):
    name: str
    email: str
    role: str = "viewer"

class UserResponse(BaseModel):
    id: int
    name: str
    email: str
    role: str

    class Config:
        from_attributes = True

MAX_PAGE_SIZE = 100
DEFAULT_PAGE_SIZE = 20

async def get_db():
    """Get database session dependency."""
    db = SessionLocal()
    try:
        yield db
    finally:
        db.close()

@router.get("/", response_model=List[UserResponse])
async def list_users(
    skip: int = 0,
    limit: int = DEFAULT_PAGE_SIZE,
    db: Session = Depends(get_db),
):
    """List users with pagination."""
    users = db.query(User).offset(skip).limit(min(limit, MAX_PAGE_SIZE)).all()
    return users

@router.post("/", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(user: UserCreate, db: Session = Depends(get_db)):
    """Create a new user."""
    db_user = User(**user.model_dump())
    db.add(db_user)
    db.commit()
    db.refresh(db_user)
    return db_user

@router.get("/{user_id}", response_model=UserResponse)
async def get_user(user_id: int, db: Session = Depends(get_db)):
    """Get a user by ID."""
    user = db.query(User).filter(User.id == user_id).first()
    if not user:
        raise HTTPException(status_code=404, detail="User not found")
    return user
