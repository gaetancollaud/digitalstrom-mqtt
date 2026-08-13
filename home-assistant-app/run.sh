#!/usr/bin/with-contenv bashio
set -euo pipefail

# API key setup is a two-phase transaction. The staged file holds a newly
# requested key, while the finalization marker records that Supervisor option
# cleanup must complete before another key may be requested.
readonly API_KEY_FILE="${API_KEY_FILE:-/data/digitalstrom-api-key}"
readonly API_KEY_STAGING_FILE="${API_KEY_STAGING_FILE:-${API_KEY_FILE}.staged}"
readonly API_KEY_FINALIZATION_FILE="${API_KEY_FINALIZATION_FILE:-${API_KEY_FILE}.pending-finalization}"
readonly PASSWORD_FILE_PREFIX="${PASSWORD_FILE_PREFIX:-/run/digitalstrom-mqtt-password}"
readonly API_KEY_NAME="digitalstrom-mqtt-home-assistant"
readonly DIGITALSTROM_MQTT_BIN="${DIGITALSTROM_MQTT_BIN:-/digitalstrom-mqtt}"

API_KEY_BOOTSTRAP_RESUMED="false"

cleanup_password_file() {
    if [[ -n "${PASSWORD_FILE:-}" && -f "${PASSWORD_FILE}" ]]; then
        rm -f "${PASSWORD_FILE}"
    fi
}

read_optional_password() {
    local password
    if ! password="$(bashio::config 'digitalstrom_password')"; then
        return 1
    fi
    if [[ "${password}" == "null" ]]; then
        password=""
    fi
    printf '%s' "${password}"
}

update_app_option() {
    bashio::addon.option "$@"
}

regeneration_option_is_enabled() {
    bashio::config.true 'regenerate_api_key'
}

request_api_key() {
    local output_file="$1"

    "${DIGITALSTROM_MQTT_BIN}" \
        -mode=get-api-key \
        -host="${DIGITALSTROM_HOST}" \
        -port="${DIGITALSTROM_PORT}" \
        -username="${DSS_USERNAME}" \
        -password-file="${PASSWORD_FILE}" \
        -api-key-file="${output_file}" \
        -integrationName="${API_KEY_NAME}"
}

finalize_api_key_options() {
    local regenerate_requested="$1"
    local password

    if [[ "${regenerate_requested}" == "true" ]]; then
        if regeneration_option_is_enabled; then
            if ! update_app_option 'regenerate_api_key' '^false'; then
                bashio::log.error "The API key was stored, but Home Assistant could not reset the regeneration option. The password was kept so regeneration can be retried."
                return 1
            fi
        else
            bashio::log.debug "The API key regeneration option is already reset."
        fi
    fi

    if ! password="$(read_optional_password)"; then
        bashio::log.error "The API key was stored, but Home Assistant could not read the temporary password option."
        return 1
    fi
    if [[ -n "${password}" ]]; then
        if ! update_app_option 'digitalstrom_password'; then
            bashio::log.error "The API key was stored, but Home Assistant could not remove the temporary password from the app options. The App will retry without creating another key."
            return 1
        fi
    else
        bashio::log.debug "No temporary digitalSTROM password remains in the App options."
    fi
}

write_api_key_finalization() {
    local regenerate_requested="$1"
    local temporary_file

    if ! temporary_file="$(mktemp "${API_KEY_FINALIZATION_FILE}.XXXXXX")"; then
        bashio::log.error "Could not create the pending API key setup state."
        return 1
    fi
    if ! chmod 0600 "${temporary_file}" \
        || ! printf '%s' "${regenerate_requested}" > "${temporary_file}" \
        || ! mv -f "${temporary_file}" "${API_KEY_FINALIZATION_FILE}"; then
        rm -f "${temporary_file}"
        bashio::log.error "Could not store the pending API key setup state."
        return 1
    fi
}

complete_api_key_finalization() {
    local regenerate_requested

    if [[ ! -s "${API_KEY_FINALIZATION_FILE}" ]]; then
        bashio::log.error "The pending API key setup state is missing or empty."
        return 1
    fi
    regenerate_requested="$(<"${API_KEY_FINALIZATION_FILE}")"
    if [[ "${regenerate_requested}" != "true" && "${regenerate_requested}" != "false" ]]; then
        bashio::log.error "The pending API key setup state is invalid."
        return 1
    fi
    if [[ ! -s "${API_KEY_FILE}" ]]; then
        bashio::log.error "The API key setup cannot be finalized because the stored key is missing."
        return 1
    fi

    finalize_api_key_options "${regenerate_requested}" || return 1
    if ! rm -f "${API_KEY_FINALIZATION_FILE}"; then
        bashio::log.error "Could not remove the completed API key setup state."
        return 1
    fi
    bashio::log.info "The digitalSTROM API key setup is complete; the temporary password option is clear."
}

resume_api_key_bootstrap() {
    local regenerate_requested="$1"
    local staged_key_resumed="false"

    if [[ -s "${API_KEY_STAGING_FILE}" ]]; then
        bashio::log.info "Resuming an interrupted digitalSTROM API key setup."
        staged_key_resumed="true"
        write_api_key_finalization "${regenerate_requested}" || return 1
        if ! mv -f "${API_KEY_STAGING_FILE}" "${API_KEY_FILE}"; then
            bashio::log.error "Could not promote the staged digitalSTROM API key."
            return 1
        fi
    fi

    if [[ -e "${API_KEY_FINALIZATION_FILE}" ]]; then
        if [[ "${staged_key_resumed}" != "true" ]]; then
            bashio::log.info "Finishing an interrupted digitalSTROM API key setup."
        fi
        API_KEY_BOOTSTRAP_RESUMED="true"
        complete_api_key_finalization || return 1
    fi
}

create_api_key() {
    local regenerate_requested="$1"
    local password
    if ! password="$(read_optional_password)"; then
        bashio::log.error "Could not read the temporary digitalSTROM password from the App options."
        return 1
    fi
    if [[ -z "${password}" ]]; then
        bashio::log.error "Enter the digitalSTROM password to create or replace the API key."
        return 1
    fi

    if ! PASSWORD_FILE="$(mktemp "${PASSWORD_FILE_PREFIX}.XXXXXX")"; then
        bashio::log.error "Could not create a temporary password file."
        return 1
    fi
    if ! chmod 0600 "${PASSWORD_FILE}" \
        || ! printf '%s' "${password}" > "${PASSWORD_FILE}"; then
        cleanup_password_file
        bashio::log.error "Could not prepare the temporary password file."
        return 1
    fi

    bashio::log.info "Creating a digitalSTROM API key."
    if ! rm -f "${API_KEY_STAGING_FILE}"; then
        cleanup_password_file || true
        bashio::log.error "Could not remove a stale staged API key."
        return 1
    fi
    if ! request_api_key "${API_KEY_STAGING_FILE}"; then
        cleanup_password_file || true
        bashio::log.error "Could not create a digitalSTROM API key. Check the server address and credentials."
        return 1
    fi
    if ! cleanup_password_file; then
        bashio::log.error "Could not remove the temporary password file."
        return 1
    fi

    # Persist the cleanup intent before replacing the canonical key. A restart
    # can then finish option cleanup without requesting a second API key.
    write_api_key_finalization "${regenerate_requested}" || return 1
    if ! mv -f "${API_KEY_STAGING_FILE}" "${API_KEY_FILE}"; then
        bashio::log.error "Could not store the new digitalSTROM API key."
        return 1
    fi
    complete_api_key_finalization
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

load_configuration() {
    unset DIGITALSTROM_USERNAME DIGITALSTROM_PASSWORD
    export DIGITALSTROM_HOST="$(bashio::config 'digitalstrom_host')"
    export DIGITALSTROM_PORT="$(bashio::config 'digitalstrom_port')"
    DSS_USERNAME="$(bashio::config 'digitalstrom_username')"
    export INVERT_BLINDS_POSITION="$(bashio::config 'invert_blinds_position')"
    export METERINGS_ENABLED="$(bashio::config 'meterings_enabled')"
    export METERINGS_INTERVAL_SECONDS="$(bashio::config 'meterings_interval_seconds')"
    export LOG_LEVEL="$(bashio::config 'log_level')"
    export HOME_ASSISTANT_DISCOVERY_ENABLED="true"
}

main() {
    local regenerate_requested="false"

    load_configuration
    if [[ -z "${DIGITALSTROM_HOST}" ]]; then
        bashio::log.error "Enter the digitalSTROM server address in the app configuration."
        return 1
    fi

    trap cleanup_password_file EXIT

    if regeneration_option_is_enabled; then
        regenerate_requested="true"
    fi
    bashio::log.debug "Using digitalSTROM server ${DIGITALSTROM_HOST}:${DIGITALSTROM_PORT}."
    resume_api_key_bootstrap "${regenerate_requested}" || return 1
    if [[ "${API_KEY_BOOTSTRAP_RESUMED}" != "true" ]] \
        && { [[ ! -s "${API_KEY_FILE}" ]] || [[ "${regenerate_requested}" == "true" ]]; }; then
        create_api_key "${regenerate_requested}" || return 1
    elif [[ "${API_KEY_BOOTSTRAP_RESUMED}" != "true" ]]; then
        bashio::log.debug "Using the stored digitalSTROM API key."
    fi

    read_mqtt_service || return 1

    export DIGITALSTROM_API_KEY="$(<"${API_KEY_FILE}")"
    export MQTT_URL="${MQTT_SCHEME}://${MQTT_HOST}:${MQTT_PORT}"
    export MQTT_USERNAME
    export MQTT_PASSWORD

    bashio::log.info "Starting digitalSTROM MQTT with the Home Assistant MQTT service."
    exec "${DIGITALSTROM_MQTT_BIN}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
