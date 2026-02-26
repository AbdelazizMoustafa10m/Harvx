"""Application configuration using Pydantic settings."""

from functools import lru_cache

from pydantic import Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""

    environment: str = Field(default="development", alias="APP_ENV")
    debug: bool = Field(default=False, alias="DEBUG")

    # Database
    database_url: str = Field(
        default="sqlite:///./taskapi.db",
        alias="DATABASE_URL",
    )

    # JWT
    secret_key: str = Field(
        default="change-me-in-production",
        alias="SECRET_KEY",
    )
    access_token_expire_minutes: int = Field(default=30)

    # CORS
    allowed_origins: list[str] = Field(
        default=["http://localhost:3000", "http://localhost:8000"],
    )

    # Pagination
    default_page_size: int = Field(default=20)
    max_page_size: int = Field(default=100)

    model_config = {"env_file": ".env", "env_file_encoding": "utf-8"}


@lru_cache
def get_settings() -> Settings:
    """Return cached application settings."""
    return Settings()


settings = get_settings()