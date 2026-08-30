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

APP_OPTIONS="{}"
API_KEY_REQUEST_SATISFIED="false"
BASHIO_LOG_LEVEL=""

cleanup_password_file() {
    if [[ -n "${PASSWORD_FILE:-}" && -f "${PASSWORD_FILE}" ]]; then
        rm -f "${PASSWORD_FILE}"
    fi
}

load_app_options() {
    if ! APP_OPTIONS="$(bashio::addon.config)"; then
        return 1
    fi
    if [[ -z "${APP_OPTIONS}" ]]; then
        APP_OPTIONS="{}"
    fi
}

app_option() {
    local key="$1"
    local default_value="$2"

    bashio::jq "${APP_OPTIONS}" ".${key} // ${default_value}"
}

read_optional_password() {
    app_option 'digitalstrom_password' '""'
}

update_app_option() {
    local key="$1"
    local value="${2:-}"
    local item
    local updated_options
    local payload
    local requested_log_level="${BASHIO_LOG_LEVEL:-${LOG_LEVEL:-INFO}}"
    local update_status=0

    if [[ -n "${value}" ]]; then
        item="\"${value}\""
        if [[ "${value:0:1}" == "^" ]]; then
            item="${value:1}"
        fi
        updated_options="$(bashio::jq "${APP_OPTIONS}" ".${key} = ${item}")" || return 1
    else
        updated_options="$(bashio::jq "${APP_OPTIONS}" "del(.${key})")" || return 1
    fi
    payload="$(bashio::var.json options "^${updated_options}")" || return 1

    # Bashio logs Supervisor API request bodies at debug level. Keep the
    # options payload, which can still contain the temporary password, out of
    # the App log while updating it.
    bashio::log.level info
    if ! bashio::api.supervisor POST '/addons/self/options' "${payload}"; then
        update_status=1
    fi
    bashio::log.level "${requested_log_level}" || true

    if [[ "${update_status}" -ne 0 ]]; then
        return 1
    fi
    APP_OPTIONS="${updated_options}"
    bashio::cache.flush_all || true
}

regeneration_option_is_enabled() {
    local value
    value="$(app_option 'regenerate_api_key' 'false')" || return 1
    case "${value}" in
        true|True|TRUE|1) return 0 ;;
        *) return 1 ;;
    esac
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
        bashio::log.warning "Ignoring an empty pending API key setup state."
        rm -f "${API_KEY_FINALIZATION_FILE}" || true
        return 0
    fi
    regenerate_requested="$(<"${API_KEY_FINALIZATION_FILE}")"
    if [[ "${regenerate_requested}" != "true" && "${regenerate_requested}" != "false" ]]; then
        bashio::log.warning "Ignoring an invalid pending API key setup state."
        rm -f "${API_KEY_FINALIZATION_FILE}" || true
        return 0
    fi
    if [[ ! -s "${API_KEY_FILE}" ]]; then
        bashio::log.warning "Discarding pending API key setup state because the stored key is missing."
        rm -f "${API_KEY_FINALIZATION_FILE}" || true
        return 0
    fi

    if ! finalize_api_key_options "${regenerate_requested}"; then
        bashio::log.warning "The API key is usable, but Home Assistant option cleanup is still pending and will be retried after the next restart."
        return 0
    fi
    if ! rm -f "${API_KEY_FINALIZATION_FILE}"; then
        bashio::log.warning "Could not remove the completed API key setup state; cleanup will be checked again after the next restart."
        return 0
    fi
    bashio::log.info "The digitalSTROM API key setup is complete; the temporary password option is clear."
}

resume_api_key_bootstrap() {
    local regenerate_requested="$1"
    local pending_regeneration="false"
    local staged_key_resumed="false"

    if [[ -e "${API_KEY_STAGING_FILE}" && ! -s "${API_KEY_STAGING_FILE}" ]]; then
        bashio::log.warning "Discarding an empty staged digitalSTROM API key."
        rm -f "${API_KEY_STAGING_FILE}" || true
    fi
    if [[ -s "${API_KEY_STAGING_FILE}" ]]; then
        bashio::log.info "Resuming an interrupted digitalSTROM API key setup."
        staged_key_resumed="true"
        write_api_key_finalization "${regenerate_requested}" || return 1
        if ! mv -f "${API_KEY_STAGING_FILE}" "${API_KEY_FILE}"; then
            bashio::log.error "Could not promote the staged digitalSTROM API key."
            return 1
        fi
        API_KEY_REQUEST_SATISFIED="true"
    fi

    if [[ -e "${API_KEY_FINALIZATION_FILE}" ]]; then
        if [[ -s "${API_KEY_FINALIZATION_FILE}" ]]; then
            pending_regeneration="$(<"${API_KEY_FINALIZATION_FILE}")"
        fi
        if [[ "${pending_regeneration}" == "true" && -s "${API_KEY_FILE}" ]]; then
            API_KEY_REQUEST_SATISFIED="true"
        elif [[ "${pending_regeneration}" == "false" \
            && "${regenerate_requested}" == "true" \
            && "${staged_key_resumed}" != "true" ]]; then
            bashio::log.info "Discarding completed first-start cleanup state before regenerating the API key."
            rm -f "${API_KEY_FINALIZATION_FILE}" || true
            return 0
        fi
        if [[ "${staged_key_resumed}" != "true" ]]; then
            bashio::log.info "Finishing an interrupted digitalSTROM API key setup."
        fi
        complete_api_key_finalization
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
    if [[ ! -s "${API_KEY_STAGING_FILE}" ]]; then
        cleanup_password_file || true
        rm -f "${API_KEY_STAGING_FILE}" || true
        bashio::log.error "digitalSTROM returned an empty API key; the existing key was preserved."
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
    API_KEY_REQUEST_SATISFIED="true"
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
        true|True|TRUE|1)
            bashio::log.error "The Home Assistant MQTT service requires TLS, but this App cannot verify a custom broker certificate. Use the Mosquitto App's default internal non-TLS service."
            return 1
            ;;
    esac
}

load_configuration() {
    unset DIGITALSTROM_USERNAME DIGITALSTROM_PASSWORD
    if ! load_app_options \
        || ! DIGITALSTROM_HOST="$(app_option 'digitalstrom_host' '""')" \
        || ! DIGITALSTROM_PORT="$(app_option 'digitalstrom_port' '8080')" \
        || ! DSS_USERNAME="$(app_option 'digitalstrom_username' '"dssadmin"')" \
        || ! INVERT_BLINDS_POSITION="$(app_option 'invert_blinds_position' 'false')" \
        || ! METERINGS_ENABLED="$(app_option 'meterings_enabled' 'true')" \
        || ! METERINGS_INTERVAL_SECONDS="$(app_option 'meterings_interval_seconds' '10')" \
        || ! LOG_LEVEL="$(app_option 'log_level' '"INFO"')"; then
        bashio::log.error "Home Assistant App configuration could not be read."
        return 1
    fi
    # Bashio traces function arguments, and some helpers receive the complete
    # App options JSON. Keep Bashio at DEBUG while still passing TRACE to the
    # bridge itself so temporary credentials cannot appear in launcher logs.
    BASHIO_LOG_LEVEL="${LOG_LEVEL}"
    if [[ "${BASHIO_LOG_LEVEL}" == "TRACE" ]]; then
        BASHIO_LOG_LEVEL="DEBUG"
    fi
    if ! bashio::log.level "${BASHIO_LOG_LEVEL}"; then
        bashio::log.error "The configured App log level is invalid."
        return 1
    fi
    export DIGITALSTROM_HOST DIGITALSTROM_PORT INVERT_BLINDS_POSITION
    export METERINGS_ENABLED METERINGS_INTERVAL_SECONDS LOG_LEVEL
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
    if [[ "${API_KEY_REQUEST_SATISFIED}" != "true" ]] \
        && { [[ ! -s "${API_KEY_FILE}" ]] || [[ "${regenerate_requested}" == "true" ]]; }; then
        create_api_key "${regenerate_requested}" || return 1
    elif [[ "${API_KEY_REQUEST_SATISFIED}" != "true" ]]; then
        bashio::log.debug "Using the stored digitalSTROM API key."
    fi

    read_mqtt_service || return 1

    if ! DIGITALSTROM_API_KEY="$(<"${API_KEY_FILE}")"; then
        bashio::log.error "The stored digitalSTROM API key could not be read."
        return 1
    fi
    MQTT_URL="${MQTT_SCHEME}://${MQTT_HOST}:${MQTT_PORT}"
    export DIGITALSTROM_API_KEY MQTT_URL
    export MQTT_USERNAME
    export MQTT_PASSWORD

    bashio::log.info "Starting digitalSTROM MQTT with the Home Assistant MQTT service."
    exec "${DIGITALSTROM_MQTT_BIN}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
