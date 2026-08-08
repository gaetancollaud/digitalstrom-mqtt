#!/usr/bin/with-contenv bashio
set -euo pipefail

readonly API_KEY_FILE="/data/digitalstrom-api-key"
readonly API_KEY_NAME="digitalstrom-mqtt-home-assistant"

cleanup_password_file() {
    if [[ -n "${PASSWORD_FILE:-}" && -f "${PASSWORD_FILE}" ]]; then
        rm -f "${PASSWORD_FILE}"
    fi
}

read_optional_password() {
    local password
    password="$(bashio::config 'digitalstrom_password')"
    if [[ "${password}" == "null" ]]; then
        password=""
    fi
    printf '%s' "${password}"
}

update_app_option() {
    if declare -F bashio::app.option > /dev/null; then
        bashio::app.option "$@"
        return
    fi
    bashio::addon.option "$@"
}

create_api_key() {
    local password
    password="$(read_optional_password)"
    if [[ -z "${password}" ]]; then
        bashio::log.error "Enter the digitalSTROM password to create or replace the API key."
        return 1
    fi

    PASSWORD_FILE="$(mktemp)"
    chmod 0600 "${PASSWORD_FILE}"
    printf '%s' "${password}" > "${PASSWORD_FILE}"

    bashio::log.info "Creating a digitalSTROM API key."
    if ! /digitalstrom-mqtt \
        -mode=get-api-key \
        -host="${DIGITALSTROM_HOST}" \
        -username="${DIGITALSTROM_USERNAME}" \
        -password-file="${PASSWORD_FILE}" \
        -api-key-file="${API_KEY_FILE}" \
        -integrationName="${API_KEY_NAME}"; then
        bashio::log.error "Could not create a digitalSTROM API key. Check the server address and credentials."
        return 1
    fi

    if ! update_app_option 'digitalstrom_password'; then
        bashio::log.warning "The API key was stored, but Home Assistant could not remove the temporary password from the app options."
    fi
    if ! update_app_option 'regenerate_api_key' '^false'; then
        bashio::log.warning "The API key was stored, but Home Assistant could not reset the regeneration option."
    fi
}

read_mqtt_service() {
    if ! MQTT_HOST="$(bashio::services mqtt 'host')" \
        || ! MQTT_PORT="$(bashio::services mqtt 'port')" \
        || ! MQTT_USERNAME="$(bashio::services mqtt 'username')" \
        || ! MQTT_PASSWORD="$(bashio::services mqtt 'password')"; then
        bashio::log.error "The required Home Assistant MQTT service is unavailable. Install and start the Mosquitto Broker app first."
        return 1
    fi

    MQTT_SSL="$(bashio::services mqtt 'ssl' || true)"
    MQTT_SCHEME="tcp"
    case "${MQTT_SSL}" in
        true|True|TRUE|1) MQTT_SCHEME="ssl" ;;
    esac
}

export DIGITALSTROM_HOST="$(bashio::config 'digitalstrom_host')"
export DIGITALSTROM_USERNAME="$(bashio::config 'digitalstrom_username')"
export INVERT_BLINDS_POSITION="$(bashio::config 'invert_blinds_position')"
export METERINGS_ENABLED="$(bashio::config 'meterings_enabled')"
export METERINGS_INTERVAL_SECONDS="$(bashio::config 'meterings_interval_seconds')"
export LOG_LEVEL="$(bashio::config 'log_level')"
export HOME_ASSISTANT_DISCOVERY_ENABLED="true"

if [[ -z "${DIGITALSTROM_HOST}" ]]; then
    bashio::log.error "Enter the digitalSTROM server address in the app configuration."
    exit 1
fi

trap cleanup_password_file EXIT

if [[ ! -s "${API_KEY_FILE}" ]] || bashio::config.true 'regenerate_api_key'; then
    create_api_key || exit 1
fi

if ! read_mqtt_service; then
    exit 1
fi

export DIGITALSTROM_API_KEY="$(<"${API_KEY_FILE}")"
export MQTT_URL="${MQTT_SCHEME}://${MQTT_HOST}:${MQTT_PORT}"
export MQTT_USERNAME
export MQTT_PASSWORD

bashio::log.info "Starting digitalSTROM MQTT with the Home Assistant MQTT service."
exec /digitalstrom-mqtt
