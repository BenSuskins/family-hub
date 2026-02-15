"""Sensor platform for Family Hub."""

from __future__ import annotations

from typing import Any

from homeassistant.components.sensor import SensorEntity
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddEntitiesCallback
from homeassistant.helpers.update_coordinator import CoordinatorEntity

from .const import DOMAIN
from .coordinator import FamilyHubCoordinator, FamilyHubData


async def async_setup_entry(
    hass: HomeAssistant,
    entry: ConfigEntry,
    async_add_entities: AddEntitiesCallback,
) -> None:
    """Set up Family Hub sensors from a config entry."""
    coordinator: FamilyHubCoordinator = hass.data[DOMAIN][entry.entry_id]

    entities: list[SensorEntity] = [
        FamilyHubChoresDueTodaySensor(coordinator),
        FamilyHubChoresOverdueSensor(coordinator),
    ]

    # Per-user sensors
    for user_id, user_chores in coordinator.data.chores_by_user.items():
        entities.append(
            FamilyHubUserChoresSensor(coordinator, user_id, user_chores.name)
        )

    async_add_entities(entities)


class FamilyHubChoresDueTodaySensor(CoordinatorEntity[FamilyHubCoordinator], SensorEntity):
    """Sensor showing count of chores due today."""

    _attr_has_entity_name = True
    _attr_name = "Chores Due Today"
    _attr_icon = "mdi:clipboard-check-outline"

    def __init__(self, coordinator: FamilyHubCoordinator) -> None:
        super().__init__(coordinator)
        self._attr_unique_id = f"{DOMAIN}_chores_due_today"

    @property
    def native_value(self) -> int:
        return len(self.coordinator.data.chores_due_today)

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        return {"chores": self.coordinator.data.chores_due_today}


class FamilyHubChoresOverdueSensor(CoordinatorEntity[FamilyHubCoordinator], SensorEntity):
    """Sensor showing count of overdue chores."""

    _attr_has_entity_name = True
    _attr_name = "Chores Overdue"
    _attr_icon = "mdi:clipboard-alert-outline"

    def __init__(self, coordinator: FamilyHubCoordinator) -> None:
        super().__init__(coordinator)
        self._attr_unique_id = f"{DOMAIN}_chores_overdue"

    @property
    def native_value(self) -> int:
        return len(self.coordinator.data.chores_overdue)

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        return {"chores": self.coordinator.data.chores_overdue}


class FamilyHubUserChoresSensor(CoordinatorEntity[FamilyHubCoordinator], SensorEntity):
    """Sensor showing chore count for a specific user."""

    _attr_has_entity_name = True
    _attr_icon = "mdi:account-check-outline"

    def __init__(
        self, coordinator: FamilyHubCoordinator, user_id: str, user_name: str
    ) -> None:
        super().__init__(coordinator)
        self._user_id = user_id
        self._user_name = user_name
        self._attr_unique_id = f"{DOMAIN}_{user_id}_chores"
        self._attr_name = f"{user_name} Chores"

    @property
    def _user_chores(self) -> dict[str, Any] | None:
        return self.coordinator.data.chores_by_user.get(self._user_id)

    @property
    def native_value(self) -> int:
        user_chores = self._user_chores
        if user_chores is None:
            return 0
        return len(user_chores.due_today) + len(user_chores.overdue)

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        user_chores = self._user_chores
        if user_chores is None:
            return {"due_today": [], "overdue": []}
        return {
            "due_today": user_chores.due_today,
            "overdue": user_chores.overdue,
        }
