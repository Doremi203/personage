import json
import os
import requests
from pathlib import Path
from typing import Any
import logging
from dotenv import load_dotenv
from yc_lockbox import YandexLockboxClient

logger = logging.getLogger(__name__)

SECRET_PREFIX = "secret:"


class Configuration:
    def __init__(self):
        self.config_dir = Path(__file__).parent.parent.parent.parent / "config"
        self._config = self._load_config()

    GET_TOKEN_FROM_VM_METADATA_URL = "http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token"

    def _load_config(self) -> dict[str, Any]:
        config = {}
        env = os.getenv("APP_ENV", "development").lower()
        logger.info(f"Loading configuration for environment: {env}")

        base_config = self.config_dir / "appsettings.json"
        if base_config.exists():
            with open(base_config, 'r') as f:
                config.update(json.load(f))
        else:
            raise FileNotFoundError(f"Base configuration not found: {base_config}")

        env_config = self.config_dir / f"appsettings.{env}.json"
        if env_config.exists():
            with open(env_config, 'r') as f:
                self._deep_update(config, json.load(f))

        Configuration._apply_environment_overrides(config)
        Configuration._resolve_secrets(config)
        return config

    def _deep_update(self, base: dict, updates: dict) -> None:
        for key, value in updates.items():
            if isinstance(value, dict) and key in base and isinstance(base[key], dict):
                self._deep_update(base[key], value)
            else:
                base[key] = value

    @staticmethod
    def _apply_environment_overrides(config_overrides: dict) -> None:
        personage_root = Path(__file__).parent.parent.parent.parent.parent.parent
        personage_secrets = personage_root / 'secrets.env'
        traitex_root = personage_root / 'backend' / 'traitex'
        traitex_secrets = traitex_root / '.env'

        load_dotenv(personage_secrets)
        load_dotenv(traitex_secrets)

        for key, value in os.environ.items():
            if '__' in key:
                parts = key.split('__')
                if len(parts) == 2:
                    section, subkey = parts
                    section = Configuration._snake_to_pascal(section)
                    subkey = Configuration._snake_to_pascal(subkey)
                    if section not in config_overrides:
                        config_overrides[section] = {}
                    current_value = config_overrides[section].get(subkey, None)
                    if current_value is None:
                        config_overrides[section][subkey] = value
                        continue

                    if isinstance(current_value, int):
                        config_overrides[section][subkey] = int(value)
                    elif isinstance(current_value, bool):
                        config_overrides[section][subkey] = value.lower() in ('true', '1', 'yes')
                    else:
                        config_overrides[section][subkey] = value

    @staticmethod
    def _snake_to_pascal(text: str) -> str:
        return ''.join(word.title() for word in text.split('_'))

    def get(self, key: str, default: Any = None) -> Any:
        keys = key.split('.')
        value = self._config
        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default
        return value

    def get_section(self, section: str) -> dict[str, Any]:
        return self._config.get(section, {})

    def __getitem__(self, key: str) -> Any:
        return self.get(key)

    @staticmethod
    def _resolve_secrets(config: dict) -> None:
        """Walk the config dict and resolve any string values with the 'secret:' prefix.

        Secret format: ``secret:{secret_id}:{version_id}:{key}``

        If no values require resolution, Lockbox is never contacted and no IAM
        token is needed — so Development environments work without credentials.
        """
        secret_refs = Configuration._collect_secret_refs(config)
        if not secret_refs:
            logger.info("No secret references found in config, skipping Lockbox")
            return

        iam_token = Configuration._get_iam_token_on_vm()
        if not iam_token:
            logger.warning("VM metadata unavailable, using IAM token from environment")
            iam_token = Configuration._get_iam_token_from_env()

        lockbox = YandexLockboxClient(iam_token)
        payload_cache: dict[str, Any] = {}

        for section_key, field_key, secret_spec in secret_refs:
            parts = secret_spec.split(":")
            if len(parts) != 4 or parts[0] != "secret":
                raise ValueError(
                    f"Invalid secret format for {section_key}.{field_key}: "
                    f"expected 'secret:{{id}}:{{version}}:{{key}}', got '{secret_spec}'"
                )

            secret_id = parts[1]
            version_id = parts[2]
            payload_key = parts[3]

            cache_key = f"{secret_id}:{version_id}"
            if cache_key not in payload_cache:
                payload_cache[cache_key] = lockbox.get_secret_payload(secret_id, version_id)

            payload = payload_cache[cache_key]
            entry = payload.get(payload_key)
            if entry is None:
                raise KeyError(
                    f"Key '{payload_key}' not found in Lockbox secret {secret_id} "
                    f"(version {version_id}) for config {section_key}.{field_key}"
                )

            config[section_key][field_key] = entry.text_value.get_secret_value()
            logger.info(f"Resolved secret for {section_key}.{field_key}")

    @staticmethod
    def _collect_secret_refs(config: dict) -> list[tuple[str, str, str]]:
        """Return a list of (section_key, field_key, secret_spec) for all
        string values that start with the ``secret:`` prefix."""
        refs = []
        for section_key, section_value in config.items():
            if not isinstance(section_value, dict):
                continue
            for field_key, field_value in section_value.items():
                if isinstance(field_value, str) and field_value.startswith(SECRET_PREFIX):
                    refs.append((section_key, field_key, field_value))
        return refs

    @staticmethod
    def _get_iam_token_on_vm() -> str | None:
        try:
            headers = {"Metadata-Flavor": "Google"}
            response = requests.get(Configuration.GET_TOKEN_FROM_VM_METADATA_URL, headers=headers, timeout=2)
            response.raise_for_status()
            return response.json()["access_token"]
        except Exception:
            logger.warning("Unable to get IAM token on VM")
            return None

    @staticmethod
    def _get_iam_token_from_env() -> str:
        yc_token = os.environ.get("YC_TOKEN", None)
        if not yc_token:
            raise RuntimeError("YC_TOKEN environment variable not set")

        return yc_token
