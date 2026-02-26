"""Database models and Pydantic schemas for TaskAPI."""

from datetime import datetime
from enum import Enum

from pydantic import BaseModel, EmailStr, Field
from sqlalchemy import Column, DateTime, ForeignKey, Integer, String, Text
from sqlalchemy.orm import DeclarativeBase, relationship


class Base(DeclarativeBase):
    """SQLAlchemy declarative base."""

    pass


class ItemStatus(str, Enum):
    """Possible statuses for a task item."""

    pending = "pending"
    in_progress = "in_progress"
    completed = "completed"
    cancelled = "cancelled"


# SQLAlchemy ORM models


class UserDB(Base):
    """User database model."""

    __tablename__ = "users"

    id = Column(Integer, primary_key=True, index=True)
    email = Column(String(255), unique=True, index=True, nullable=False)
    username = Column(String(100), unique=True, index=True, nullable=False)
    hashed_password = Column(String(255), nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    items = relationship("ItemDB", back_populates="owner")


class ItemDB(Base):
    """Task item database model."""

    __tablename__ = "items"

    id = Column(Integer, primary_key=True, index=True)
    title = Column(String(200), nullable=False)
    description = Column(Text, nullable=True)
    status = Column(String(20), default=ItemStatus.pending.value)
    owner_id = Column(Integer, ForeignKey("users.id"))
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    owner = relationship("UserDB", back_populates="items")


# Pydantic request/response schemas


class UserCreate(BaseModel):
    """Schema for creating a new user."""

    email: EmailStr
    username: str = Field(min_length=3, max_length=100)
    password: str = Field(min_length=8, max_length=128)


class UserResponse(BaseModel):
    """Schema for user responses."""

    id: int
    email: str
    username: str
    created_at: datetime

    model_config = {"from_attributes": True}


class ItemCreate(BaseModel):
    """Schema for creating a new item."""

    title: str = Field(min_length=1, max_length=200)
    description: str | None = None


class ItemUpdate(BaseModel):
    """Schema for updating an item."""

    title: str | None = Field(default=None, max_length=200)
    description: str | None = None
    status: ItemStatus | None = None


class ItemResponse(BaseModel):
    """Schema for item responses."""

    id: int
    title: str
    description: str | None
    status: ItemStatus
    owner_id: int
    created_at: datetime
    updated_at: datetime

    model_config = {"from_attributes": True}