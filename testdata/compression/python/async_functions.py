"""Async functions, context managers, and *args/**kwargs patterns."""

import asyncio
from typing import Any, AsyncIterator, Optional
from contextlib import asynccontextmanager


async def fetch_url(url: str, timeout: float = 30.0) -> bytes:
    """Fetch content from a URL."""
    async with asyncio.timeout(timeout):
        reader, writer = await asyncio.open_connection(url, 443, ssl=True)
        writer.write(b"GET / HTTP/1.1\r\n\r\n")
        data = await reader.read(4096)
        writer.close()
        return data


async def process_batch(
    items: list[str],
    *args: Any,
    concurrency: int = 10,
    **kwargs: Any,
) -> list[dict]:
    """Process a batch of items concurrently.

    Args:
        items: Items to process.
        *args: Additional positional arguments.
        concurrency: Maximum concurrent tasks.
        **kwargs: Additional keyword arguments.

    Returns:
        List of processing results.
    """
    semaphore = asyncio.Semaphore(concurrency)
    results = []

    async def _process_one(item: str) -> dict:
        async with semaphore:
            return {"item": item, "status": "done"}

    tasks = [_process_one(item) for item in items]
    results = await asyncio.gather(*tasks)
    return list(results)


@asynccontextmanager
async def managed_connection(
    host: str,
    port: int,
    **kwargs: Any,
) -> AsyncIterator[Any]:
    """Async context manager for managed connections."""
    conn = await create_connection(host, port, **kwargs)
    try:
        yield conn
    finally:
        await conn.close()


async def stream_data(
    source: str,
    chunk_size: int = 1024,
) -> AsyncIterator[bytes]:
    """Stream data from a source in chunks."""
    reader = await open_source(source)
    try:
        while True:
            chunk = await reader.read(chunk_size)
            if not chunk:
                break
            yield chunk
    finally:
        await reader.close()


def sync_wrapper(*args: Any, **kwargs: Any) -> Any:
    """Synchronous wrapper for running async functions."""
    loop = asyncio.new_event_loop()
    try:
        return loop.run_until_complete(
            _run_async(*args, **kwargs)
        )
    finally:
        loop.close()


async def gather_with_errors(
    *coros: Any,
    return_exceptions: bool = False,
) -> list[Any]:
    """Gather coroutines with optional error handling."""
    results = await asyncio.gather(
        *coros, return_exceptions=return_exceptions
    )
    return list(results)


class AsyncProcessor:
    """Async processor with lifecycle management."""

    def __init__(self, name: str, workers: int = 4) -> None:
        """Initialize the async processor."""
        self._name = name
        self._workers = workers
        self._running = False

    async def __aenter__(self) -> "AsyncProcessor":
        await self.start()
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.stop()

    async def start(self) -> None:
        """Start the processor."""
        self._running = True

    async def stop(self) -> None:
        """Stop the processor."""
        self._running = False

    async def process(self, data: Any, **kwargs: Any) -> Optional[Any]:
        """Process a single item."""
        if not self._running:
            raise RuntimeError("Processor is not running")
        return await self._do_process(data, **kwargs)

    async def _do_process(self, data: Any, **kwargs: Any) -> Any:
        """Internal processing logic."""
        await asyncio.sleep(0.01)
        return {"processed": data}
