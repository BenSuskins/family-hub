"""API client for Family Hub."""

from __future__ import annotations

import asyncio
import logging

import aiohttp

_LOGGER = logging.getLogger(__name__)


class FamilyHubConnectionError(Exception):
    """Error connecting to Family Hub."""


class FamilyHubAuthError(Exception):
    """Authentication error with Family Hub."""


class FamilyHubAPI:
    """API client for Family Hub."""

    def __init__(self, base_url: str, token: str, session: aiohttp.ClientSession) -> None:
        self._base_url = base_url.rstrip("/")
        self._token = token
        self._session = session

    @property
    def _headers(self) -> dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    async def _request(self, path: str, params: dict[str, str] | None = None) -> list[dict]:
        """Make an authenticated GET request."""
        url = f"{self._base_url}{path}"
        try:
            async with self._session.get(
                url, headers=self._headers, params=params, timeout=aiohttp.ClientTimeout(total=10)
            ) as response:
                if response.status == 401:
                    raise FamilyHubAuthError("Invalid API token")
                if response.status == 403:
                    raise FamilyHubAuthError("Forbidden")
                if response.status != 200:
                    raise FamilyHubConnectionError(
                        f"Unexpected status {response.status} from {path}"
                    )
                return await response.json()
        except FamilyHubAuthError:
            raise
        except (aiohttp.ClientError, asyncio.TimeoutError) as err:
            raise FamilyHubConnectionError(f"Error connecting to Family Hub: {err}") from err

    async def get_users(self) -> list[dict]:
        """Get all users."""
        return await self._request("/api/users")

    async def get_chores(self, status: str) -> list[dict]:
        """Get chores filtered by status."""
        return await self._request("/api/chores", params={"status": status})

    async def validate_connection(self) -> bool:
        """Validate the connection and auth token."""
        await self.get_users()
        return True
