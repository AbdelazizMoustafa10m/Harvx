"""Dataclass examples with type hints and default values."""

from __future__ import annotations
from dataclasses import dataclass, field
from typing import Optional, List, Dict

MAX_CONNECTIONS: int = 100
DEFAULT_TIMEOUT: float = 30.0


@dataclass
class DatabaseConfig:
    """Configuration for database connections."""

    host: str = "localhost"
    port: int = 5432
    database: str = "mydb"
    username: str = "admin"
    password: str = ""
    pool_size: int = 10
    timeout: float = DEFAULT_TIMEOUT

    def connection_string(self) -> str:
        """Build the connection string."""
        return f"postgresql://{self.username}:{self.password}@{self.host}:{self.port}/{self.database}"

    def validate(self) -> bool:
        """Validate configuration values."""
        if self.port < 0 or self.port > 65535:
            return False
        if self.pool_size < 1:
            return False
        return True


@dataclass
class ServerConfig:
    """Configuration for the HTTP server."""

    host: str = "0.0.0.0"
    port: int = 8080
    debug: bool = False
    workers: int = 4
    allowed_origins: List[str] = field(default_factory=list)
    extra_headers: Dict[str, str] = field(default_factory=dict)

    def bind_address(self) -> str:
        """Return the bind address string."""
        return f"{self.host}:{self.port}"


@dataclass(frozen=True)
class Coordinate:
    """Immutable 2D coordinate."""

    x: float
    y: float

    def distance_to(self, other: Coordinate) -> float:
        """Calculate Euclidean distance to another coordinate."""
        return ((self.x - other.x) ** 2 + (self.y - other.y) ** 2) ** 0.5


@dataclass
class AppConfig:
    """Top-level application configuration."""

    db: DatabaseConfig = field(default_factory=DatabaseConfig)
    server: ServerConfig = field(default_factory=ServerConfig)
    app_name: str = "myapp"
    version: str = "1.0.0"
    log_level: str = "INFO"
    features: List[str] = field(default_factory=list)

    def is_debug(self) -> bool:
        """Check if running in debug mode."""
        return self.server.debug or self.log_level == "DEBUG"
