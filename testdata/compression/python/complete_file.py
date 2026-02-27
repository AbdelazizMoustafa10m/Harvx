"""Task queue implementation for background job processing."""

import asyncio
import logging
from typing import Any, Callable, Dict, List, Optional
from dataclasses import dataclass, field
from enum import Enum

__all__ = ["TaskQueue", "Task", "TaskStatus", "TaskResult"]

logger = logging.getLogger(__name__)

MAX_RETRIES: int = 3
DEFAULT_TIMEOUT: float = 300.0
QUEUE_SIZE: int = 1000


class TaskStatus(Enum):
    """Possible states for a task."""

    PENDING = "pending"
    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


@dataclass
class TaskResult:
    """Result of a task execution."""

    task_id: str
    status: TaskStatus
    result: Optional[Any] = None
    error: Optional[str] = None
    duration: float = 0.0


@dataclass
class Task:
    """A unit of work to be executed."""

    id: str
    name: str
    func: Callable[..., Any]
    args: tuple = ()
    kwargs: Dict[str, Any] = field(default_factory=dict)
    max_retries: int = MAX_RETRIES
    timeout: float = DEFAULT_TIMEOUT
    priority: int = 0

    def __post_init__(self) -> None:
        if self.max_retries < 0:
            raise ValueError("max_retries must be non-negative")
        if self.timeout <= 0:
            raise ValueError("timeout must be positive")


class TaskQueue:
    """Async task queue for background processing."""

    def __init__(
        self,
        max_workers: int = 4,
        queue_size: int = QUEUE_SIZE,
    ) -> None:
        """Initialize the task queue.

        Args:
            max_workers: Maximum concurrent workers.
            queue_size: Maximum queue capacity.
        """
        self._queue: asyncio.Queue[Task] = asyncio.Queue(maxsize=queue_size)
        self._workers: List[asyncio.Task] = []
        self._results: Dict[str, TaskResult] = {}
        self._max_workers = max_workers
        self._running = False

    async def start(self) -> None:
        """Start the task queue workers."""
        if self._running:
            return
        self._running = True
        for i in range(self._max_workers):
            worker = asyncio.create_task(self._worker(f"worker-{i}"))
            self._workers.append(worker)
        logger.info("Task queue started with %d workers", self._max_workers)

    async def stop(self) -> None:
        """Stop the task queue and wait for pending tasks."""
        self._running = False
        for worker in self._workers:
            worker.cancel()
        await asyncio.gather(*self._workers, return_exceptions=True)
        self._workers.clear()
        logger.info("Task queue stopped")

    async def submit(self, task: Task) -> str:
        """Submit a task for execution.

        Args:
            task: The task to execute.

        Returns:
            The task ID.

        Raises:
            RuntimeError: If the queue is not running.
        """
        if not self._running:
            raise RuntimeError("Task queue is not running")
        await self._queue.put(task)
        logger.debug("Task %s submitted", task.id)
        return task.id

    async def get_result(self, task_id: str) -> Optional[TaskResult]:
        """Get the result of a completed task."""
        return self._results.get(task_id)

    async def _worker(self, name: str) -> None:
        """Worker loop that processes tasks from the queue."""
        while self._running:
            try:
                task = await asyncio.wait_for(
                    self._queue.get(), timeout=1.0
                )
                result = await self._execute(task)
                self._results[task.id] = result
            except asyncio.TimeoutError:
                continue
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error("Worker %s error: %s", name, e)

    async def _execute(self, task: Task) -> TaskResult:
        """Execute a single task with retry logic."""
        import time
        start = time.monotonic()
        last_error = None

        for attempt in range(task.max_retries + 1):
            try:
                result = await asyncio.wait_for(
                    asyncio.to_thread(task.func, *task.args, **task.kwargs),
                    timeout=task.timeout,
                )
                duration = time.monotonic() - start
                return TaskResult(
                    task_id=task.id,
                    status=TaskStatus.COMPLETED,
                    result=result,
                    duration=duration,
                )
            except Exception as e:
                last_error = e
                logger.warning(
                    "Task %s attempt %d failed: %s",
                    task.id, attempt + 1, e,
                )

        duration = time.monotonic() - start
        return TaskResult(
            task_id=task.id,
            status=TaskStatus.FAILED,
            error=str(last_error),
            duration=duration,
        )

    @property
    def pending_count(self) -> int:
        """Number of tasks waiting in the queue."""
        return self._queue.qsize()

    @property
    def is_running(self) -> bool:
        """Whether the task queue is currently running."""
        return self._running
