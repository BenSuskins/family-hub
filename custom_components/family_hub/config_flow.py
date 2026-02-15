"""Config flow for Family Hub."""

from __future__ import annotations

import logging
from typing import Any

import voluptuous as vol

from homeassistant.config_entries import ConfigFlow, ConfigFlowResult
from homeassistant.helpers.aiohttp_client import async_get_clientsession

from .api import FamilyHubAPI, FamilyHubAuthError, FamilyHubConnectionError
from .const import DOMAIN

_LOGGER = logging.getLogger(__name__)

STEP_USER_DATA_SCHEMA = vol.Schema(
    {
        vol.Required("url"): str,
        vol.Required("token"): str,
    }
)


class FamilyHubConfigFlow(ConfigFlow, domain=DOMAIN):
    """Config flow for Family Hub integration."""

    VERSION = 1

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> ConfigFlowResult:
        """Handle the initial step."""
        errors: dict[str, str] = {}

        if user_input is not None:
            url = user_input["url"].rstrip("/")
            token = user_input["token"]

            # Prevent duplicate entries for the same URL
            self._async_abort_entries_match({"url": url})

            session = async_get_clientsession(self.hass)
            api = FamilyHubAPI(url, token, session)

            try:
                await api.validate_connection()
            except FamilyHubAuthError:
                errors["base"] = "invalid_auth"
            except FamilyHubConnectionError:
                errors["base"] = "cannot_connect"
            except Exception:
                _LOGGER.exception("Unexpected error during config flow")
                errors["base"] = "unknown"
            else:
                return self.async_create_entry(
                    title="Family Hub",
                    data={"url": url, "token": token},
                )

        return self.async_show_form(
            step_id="user",
            data_schema=STEP_USER_DATA_SCHEMA,
            errors=errors,
        )
