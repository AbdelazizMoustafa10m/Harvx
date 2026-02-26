"""Item routes for CRUD operations on task items."""

import logging

from fastapi import APIRouter, HTTPException, Query, status

from app.models import ItemCreate, ItemResponse, ItemStatus, ItemUpdate

logger = logging.getLogger(__name__)
router = APIRouter()

# In-memory store for demo purposes
_items: dict[int, dict] = {}
_next_id = 1


@router.get("/", response_model=list[ItemResponse])
async def list_items(
    status_filter: ItemStatus | None = Query(default=None, alias="status"),
    skip: int = Query(default=0, ge=0),
    limit: int = Query(default=20, ge=1, le=100),
) -> list[ItemResponse]:
    """List items with optional status filtering and pagination."""
    items = list(_items.values())
    if status_filter:
        items = [i for i in items if i["status"] == status_filter.value]
    return items[skip : skip + limit]


@router.post("/", response_model=ItemResponse, status_code=status.HTTP_201_CREATED)
async def create_item(item_data: ItemCreate) -> ItemResponse:
    """Create a new task item."""
    global _next_id
    from datetime import datetime

    now = datetime.utcnow()
    item = {
        "id": _next_id,
        "title": item_data.title,
        "description": item_data.description,
        "status": ItemStatus.pending.value,
        "owner_id": 1,
        "created_at": now,
        "updated_at": now,
    }
    _items[_next_id] = item
    _next_id += 1

    logger.info("Item created: %s (id=%d)", item["title"], item["id"])
    return ItemResponse(**item)


@router.get("/{item_id}", response_model=ItemResponse)
async def get_item(item_id: int) -> ItemResponse:
    """Retrieve an item by its ID."""
    if item_id not in _items:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Item {item_id} not found",
        )
    return ItemResponse(**_items[item_id])


@router.put("/{item_id}", response_model=ItemResponse)
async def update_item(item_id: int, update: ItemUpdate) -> ItemResponse:
    """Update an existing item."""
    if item_id not in _items:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Item {item_id} not found",
        )

    item = _items[item_id]
    update_data = update.model_dump(exclude_unset=True)
    for field, value in update_data.items():
        if value is not None:
            item[field] = value if not isinstance(value, ItemStatus) else value.value

    from datetime import datetime
    item["updated_at"] = datetime.utcnow()
    logger.info("Item updated: id=%d", item_id)
    return ItemResponse(**item)


@router.delete("/{item_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_item(item_id: int) -> None:
    """Delete an item by its ID."""
    if item_id not in _items:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Item {item_id} not found",
        )
    del _items[item_id]
    logger.info("Item deleted: id=%d", item_id)