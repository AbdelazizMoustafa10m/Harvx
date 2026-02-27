"""Examples of various Python decorators."""

import functools
from abc import ABC, abstractmethod
from typing import Any, Callable


def timer(func: Callable) -> Callable:
    """Decorator that times function execution."""
    @functools.wraps(func)
    def wrapper(*args: Any, **kwargs: Any) -> Any:
        import time
        start = time.time()
        result = func(*args, **kwargs)
        elapsed = time.time() - start
        print(f"{func.__name__} took {elapsed:.2f}s")
        return result
    return wrapper


def retry(max_attempts: int = 3, delay: float = 1.0) -> Callable:
    """Decorator factory for retrying failed operations."""
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            last_error = None
            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    last_error = e
            raise last_error
        return wrapper
    return decorator


class Shape(ABC):
    """Abstract base class for shapes."""

    @abstractmethod
    def area(self) -> float:
        """Calculate the area of the shape."""
        ...

    @abstractmethod
    def perimeter(self) -> float:
        """Calculate the perimeter of the shape."""
        ...

    def describe(self) -> str:
        """Return a description of the shape."""
        return f"{self.__class__.__name__}: area={self.area():.2f}"


class Circle(Shape):
    """A circle shape."""

    def __init__(self, radius: float) -> None:
        self._radius = radius

    @property
    def radius(self) -> float:
        """The radius of the circle."""
        return self._radius

    @radius.setter
    def radius(self, value: float) -> None:
        if value < 0:
            raise ValueError("Radius must be non-negative")
        self._radius = value

    def area(self) -> float:
        """Calculate circle area."""
        import math
        return math.pi * self._radius ** 2

    def perimeter(self) -> float:
        """Calculate circle perimeter."""
        import math
        return 2 * math.pi * self._radius

    @staticmethod
    def from_diameter(diameter: float) -> "Circle":
        """Create a circle from its diameter."""
        return Circle(diameter / 2)

    @classmethod
    def unit_circle(cls) -> "Circle":
        """Create a unit circle with radius 1."""
        return cls(1.0)


@timer
@retry(max_attempts=5)
def fetch_resource(url: str) -> dict:
    """Fetch a resource from a URL with retry logic."""
    import urllib.request
    with urllib.request.urlopen(url) as response:
        return response.read()
