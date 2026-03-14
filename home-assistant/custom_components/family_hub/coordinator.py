"""Data coordinator for Family Hub."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import date
import logging
from typing import Any

from homeassistant.core import HomeAssistant
from homeassistant.exceptions import ConfigEntryAuthFailed
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed

from .api import FamilyHubAPI, FamilyHubAuthError, FamilyHubConnectionError
from .const import DEFAULT_SCAN_INTERVAL, DOMAIN

_LOGGER = logging.getLogger(__name__)


@dataclass
class UserChores:
    """Chore summary for a single user."""

    name: str
    due_today: list[dict] = field(default_factory=list)
    overdue: list[dict] = field(default_factory=list)


@dataclass
class FamilyHubData:
    """Data returned by the coordinator."""

    users: dict[str, dict] = field(default_factory=dict)
    chores_due_today: list[dict] = field(default_factory=list)
    chores_overdue: list[dict] = field(default_factory=list)
    chores_by_user: dict[str, UserChores] = field(default_factory=dict)


def _is_due_today(chore: dict) -> bool:
    """Check if a chore's DueDate is today."""
    due_date = chore.get("DueDate")
    if not due_date:
        return False
    try:
        # Go marshals time.Time as RFC3339: "2025-02-15T00:00:00Z"
        return date.fromisoformat(due_date[:10]) == date.today()
    except (ValueError, TypeError):
        return False


def _chore_summary(chore: dict, users: dict[str, dict]) -> dict[str, Any]:
    """Build a slim chore summary dict for sensor attributes."""
    assignee_id = chore.get("AssignedToUserID")
    assignee_name = users.get(assignee_id, {}).get("Name", "Unassigned") if assignee_id else "Unassigned"
    return {
        "name": chore.get("Name", ""),
        "assignee": assignee_name,
        "due_date": chore.get("DueDate"),
        "due_time": chore.get("DueTime"),
    }


class FamilyHubCoordinator(DataUpdateCoordinator[FamilyHubData]):
    """Coordinator to poll Family Hub API."""

    def __init__(self, hass: HomeAssistant, api: FamilyHubAPI) -> None:
        super().__init__(
            hass,
            _LOGGER,
            name=DOMAIN,
            update_interval=DEFAULT_SCAN_INTERVAL,
        )
        self.api = api

    async def _async_update_data(self) -> FamilyHubData:
        try:
            raw_users = await self.api.get_users()
            pending_chores = await self.api.get_chores("pending")
            overdue_chores = await self.api.get_chores("overdue")
        except FamilyHubAuthError as err:
            raise ConfigEntryAuthFailed from err
        except FamilyHubConnectionError as err:
            raise UpdateFailed(str(err)) from err

        users = {user["ID"]: user for user in raw_users}

        due_today = [c for c in pending_chores if _is_due_today(c)]

        # Build per-user breakdown
        chores_by_user: dict[str, UserChores] = {}
        for user_id, user in users.items():
            chores_by_user[user_id] = UserChores(name=user.get("Name", "Unknown"))

        for chore in due_today:
            assignee = chore.get("AssignedToUserID")
            if assignee and assignee in chores_by_user:
                chores_by_user[assignee].due_today.append(
                    _chore_summary(chore, users)
                )

        for chore in overdue_chores:
            assignee = chore.get("AssignedToUserID")
            if assignee and assignee in chores_by_user:
                chores_by_user[assignee].overdue.append(
                    _chore_summary(chore, users)
                )

        return FamilyHubData(
            users=users,
            chores_due_today=[_chore_summary(c, users) for c in due_today],
            chores_overdue=[_chore_summary(c, users) for c in overdue_chores],
            chores_by_user=chores_by_user,
        )
