"""Tests for user endpoints."""

import pytest
from fastapi.testclient import TestClient

from main import app

client = TestClient(app)


class TestCreateUser:
    """Tests for POST /api/users."""

    def test_create_user_success(self) -> None:
        response = client.post(
            "/api/users",
            json={
                "email": "alice@example.com",
                "username": "alice",
                "password": "securepassword123",
            },
        )
        assert response.status_code == 201
        data = response.json()
        assert data["email"] == "alice@example.com"
        assert data["username"] == "alice"
        assert "id" in data
        assert "created_at" in data

    def test_create_user_short_password(self) -> None:
        response = client.post(
            "/api/users",
            json={
                "email": "bob@example.com",
                "username": "bob",
                "password": "short",
            },
        )
        assert response.status_code == 422

    def test_create_user_invalid_email(self) -> None:
        response = client.post(
            "/api/users",
            json={
                "email": "not-an-email",
                "username": "charlie",
                "password": "securepassword123",
            },
        )
        assert response.status_code == 422


class TestGetUser:
    """Tests for GET /api/users/{user_id}."""

    def test_get_nonexistent_user(self) -> None:
        response = client.get("/api/users/99999")
        assert response.status_code == 404

    def test_get_user_by_id(self) -> None:
        # First create a user
        create_resp = client.post(
            "/api/users",
            json={
                "email": "dave@example.com",
                "username": "dave",
                "password": "securepassword123",
            },
        )
        user_id = create_resp.json()["id"]

        # Then retrieve
        response = client.get(f"/api/users/{user_id}")
        assert response.status_code == 200
        assert response.json()["username"] == "dave"


class TestHealthCheck:
    """Tests for the health endpoint."""

    def test_health_returns_ok(self) -> None:
        response = client.get("/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"