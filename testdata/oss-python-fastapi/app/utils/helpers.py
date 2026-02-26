"""Common helper functions for the TaskAPI application."""

import re
from datetime import datetime, timedelta

from jose import jwt

from app.config import settings

ALGORITHM = "HS256"


def create_access_token(
    data: dict,
    expires_delta: timedelta | None = None,
) -> str:
    """Create a JWT access token with the given claims.

    Args:
        data: Claims to encode in the token.
        expires_delta: Optional custom expiration time.

    Returns:
        Encoded JWT string.
    """
    to_encode = data.copy()
    expire = datetime.utcnow() + (
        expires_delta or timedelta(minutes=settings.access_token_expire_minutes)
    )
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, settings.secret_key, algorithm=ALGORITHM)


def decode_access_token(token: str) -> dict:
    """Decode and validate a JWT access token.

    Args:
        token: The JWT string to decode.

    Returns:
        Decoded claims dictionary.

    Raises:
        jose.JWTError: If the token is invalid or expired.
    """
    return jwt.decode(token, settings.secret_key, algorithms=[ALGORITHM])


def sanitize_string(value: str) -> str:
    """Remove potentially dangerous characters from a string.

    Strips HTML tags and limits whitespace.
    """
    # Remove HTML tags
    cleaned = re.sub(r"<[^>]+>", "", value)
    # Normalize whitespace
    cleaned = re.sub(r"\s+", " ", cleaned).strip()
    return cleaned


def paginate(items: list, page: int = 1, per_page: int = 20) -> dict:
    """Apply pagination to a list of items.

    Returns:
        Dictionary with 'items', 'total', 'page', 'per_page', and 'has_more'.
    """
    total = len(items)
    start = (page - 1) * per_page
    end = start + per_page

    return {
        "items": items[start:end],
        "total": total,
        "page": page,
        "per_page": per_page,
        "has_more": end < total,
    }