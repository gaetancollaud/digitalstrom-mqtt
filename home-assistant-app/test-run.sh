#!/usr/bin/env bash
set -euo pipefail

TEST_DIR="$(mktemp -d)"
readonly TEST_DIR
APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly APP_DIR

cleanup() {
    rm -rf "${TEST_DIR}"
}
trap cleanup EXIT

export API_KEY_FILE="${TEST_DIR}/digitalstrom-api-key"
export PASSWORD_FILE_PREFIX="${TEST_DIR}/digitalstrom-mqtt-password"
export DIGITALSTROM_MQTT_BIN="${TEST_DIR}/digitalstrom-mqtt"

# shellcheck source=run.sh
source "${APP_DIR}/run.sh"

bashio::log.error() { :; }
bashio::log.debug() { :; }
bashio::log.info() { :; }
bashio::log.warning() { :; }
bashio::log.level() { :; }

fail() {
    printf 'FAIL: %s\n' "$1" >&2
    exit 1
}

assert_equal() {
    local expected="$1"
    local actual="$2"
    local message="$3"
    [[ "${actual}" == "${expected}" ]] || fail "${message}: expected '${expected}', got '${actual}'"
}

assert_file_contains() {
    local expected="$1"
    local file="$2"
    local message="$3"
    grep -Fqx -- "${expected}" "${file}" || fail "${message}: missing '${expected}'"
}

assert_file_not_contains() {
    local unexpected="$1"
    local file="$2"
    local message="$3"
    if grep -Fq -- "${unexpected}" "${file}"; then
        fail "${message}: found '${unexpected}'"
    fi
}

reset_api_key_state() {
    rm -f "${API_KEY_FILE}" "${API_KEY_STAGING_FILE}" "${API_KEY_FINALIZATION_FILE}"
    API_KEY_REQUEST_SATISFIED="false"
}

test_boolean_false_is_not_replaced_by_default() {
    APP_OPTIONS='{"meterings_enabled":false}'
    bashio::jq() {
        assert_equal '{"meterings_enabled":false}' "$1" "boolean option input"
        assert_equal \
            'if .meterings_enabled == null then true else .meterings_enabled end' \
            "$2" \
            "boolean option jq filter"
        printf 'false'
    }

    local value
    value="$(app_option 'meterings_enabled' 'true')"

    assert_equal "false" "${value}" "explicit false option value"
}

test_first_start_removes_password_without_resetting_regeneration() {
    local -a calls=()
    read_optional_password() { printf 'temporary-password'; }
    update_app_option() { calls+=("$*"); }

    finalize_api_key_options "false"

    assert_equal "1" "${#calls[@]}" "first start option update count"
    assert_equal 'digitalstrom_password ^""' "${calls[0]}" "first start option update"
}

test_regeneration_resets_flag_before_removing_password() {
    local -a calls=()
    regeneration_option_is_enabled() { return 0; }
    read_optional_password() { printf 'temporary-password'; }
    update_app_option() { calls+=("$*"); }

    finalize_api_key_options "true"

    assert_equal "2" "${#calls[@]}" "regeneration option update count"
    assert_equal "regenerate_api_key ^false" "${calls[0]}" "first regeneration option update"
    assert_equal 'digitalstrom_password ^""' "${calls[1]}" "second regeneration option update"
}

test_failed_regeneration_reset_keeps_password() {
    local -a calls=()
    regeneration_option_is_enabled() { return 0; }
    read_optional_password() { printf 'temporary-password'; }
    update_app_option() {
        calls+=("$*")
        return 1
    }

    if finalize_api_key_options "true"; then
        fail "failed regeneration reset should fail bootstrap"
    fi

    assert_equal "1" "${#calls[@]}" "failed regeneration option update count"
    assert_equal "regenerate_api_key ^false" "${calls[0]}" "failed regeneration option update"
}

test_already_applied_option_updates_are_not_repeated() {
    regeneration_option_is_enabled() { return 1; }
    read_optional_password() { :; }
    update_app_option() { fail "already applied option update was repeated"; }

    finalize_api_key_options "true"
}

test_password_option_read_failure_stops_bootstrap() {
    regeneration_option_is_enabled() { return 1; }
    read_optional_password() { return 1; }
    update_app_option() { fail "option update ran after password read failure"; }

    if finalize_api_key_options "false"; then
        fail "password option read failure should stop finalization"
    fi
    if create_api_key "false"; then
        fail "password option read failure should stop key creation"
    fi
}

test_option_update_propagates_supervisor_failure() {
    local -a log_levels=()
    local current_log_level="DEBUG"
    APP_OPTIONS='{"digitalstrom_password":"temporary-password"}'
    LOG_LEVEL="TRACE"
    BASHIO_LOG_LEVEL="DEBUG"
    bashio::jq() {
        assert_equal "DEBUG" "${current_log_level}" "option update JSON log level"
        assert_equal '.digitalstrom_password = ""' "$2" "option update jq filter"
        printf '{}'
    }
    bashio::var.json() { printf '{"options":{}}'; }
    bashio::api.supervisor() { return 1; }
    bashio::cache.flush_all() { fail "failed option update flushed the cache"; }
    bashio::log.level() {
        current_log_level="$1"
        log_levels+=("$1")
    }

    if update_app_option 'digitalstrom_password' '^""'; then
        fail "failed Supervisor option update was accepted"
    fi

    assert_equal '{"digitalstrom_password":"temporary-password"}' "${APP_OPTIONS}" "options after failed update"
    assert_equal "info" "${log_levels[0]}" "option update temporary log level"
    assert_equal "DEBUG" "${log_levels[1]}" "option update restored log level"
}

test_failed_regeneration_cleanup_resumes_without_another_key() {
    local allow_option_updates="false"
    local request_count=0
    local -a calls=()
    reset_api_key_state
    printf 'old-api-key' > "${API_KEY_FILE}"
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_PORT="8443"
    DSS_USERNAME="dssadmin"
    read_optional_password() { printf 'test-password'; }
    regeneration_option_is_enabled() { return 0; }
    request_api_key() {
        request_count=$((request_count + 1))
        printf 'new-api-key' > "$1"
        chmod 0600 "$1"
    }
    update_app_option() {
        calls+=("$*")
        [[ "${allow_option_updates}" == "true" ]]
    }

    create_api_key "true"
    assert_equal "new-api-key" "$(<"${API_KEY_FILE}")" "replacement API key"
    assert_equal "true" "$(<"${API_KEY_FINALIZATION_FILE}")" "pending regeneration state"
    assert_equal "1" "${request_count}" "initial API key request count"

    allow_option_updates="true"
    resume_api_key_bootstrap "true"

    assert_equal "1" "${request_count}" "resumed API key request count"
    assert_equal "true" "${API_KEY_REQUEST_SATISFIED}" "resumed regeneration marker"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "pending regeneration state was not removed"
    assert_equal "3" "${#calls[@]}" "regeneration retry option update count"
    assert_equal "regenerate_api_key ^false" "${calls[1]}" "retried regeneration reset"
    assert_equal 'digitalstrom_password ^""' "${calls[2]}" "retried password removal"
}

test_failed_password_removal_resumes_without_another_key() {
    local fail_password_removal="true"
    local request_count=0
    reset_api_key_state
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_PORT="8080"
    DSS_USERNAME="dssadmin"
    read_optional_password() { printf 'test-password'; }
    request_api_key() {
        request_count=$((request_count + 1))
        printf 'new-api-key' > "$1"
        chmod 0600 "$1"
    }
    update_app_option() {
        if [[ "$1" == "digitalstrom_password" && "${fail_password_removal}" == "true" ]]; then
            return 1
        fi
    }

    create_api_key "false"
    assert_equal "false" "$(<"${API_KEY_FINALIZATION_FILE}")" "pending first-start state"

    fail_password_removal="false"
    resume_api_key_bootstrap "false"

    assert_equal "1" "${request_count}" "password cleanup retry API key request count"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "pending password cleanup state was not removed"
}

test_interrupted_staged_key_is_promoted_without_another_request() {
    reset_api_key_state
    printf 'staged-api-key' > "${API_KEY_STAGING_FILE}"
    chmod 0600 "${API_KEY_STAGING_FILE}"
    request_api_key() { fail "staged API key must not be requested again"; }
    read_optional_password() { :; }
    update_app_option() { :; }

    resume_api_key_bootstrap "false"

    assert_equal "staged-api-key" "$(<"${API_KEY_FILE}")" "promoted staged API key"
    assert_equal "true" "${API_KEY_REQUEST_SATISFIED}" "staged bootstrap marker"
    [[ ! -e "${API_KEY_STAGING_FILE}" ]] || fail "staged API key was not promoted"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "staged API key finalization was not completed"
}

test_failed_staged_key_promotion_does_not_finalize_options() {
    local option_update_count=0
    reset_api_key_state
    printf 'old-api-key' > "${API_KEY_FILE}"
    printf 'staged-api-key' > "${API_KEY_STAGING_FILE}"
    chmod 0600 "${API_KEY_FILE}" "${API_KEY_STAGING_FILE}"
    update_app_option() { option_update_count=$((option_update_count + 1)); }
    mv() {
        if [[ "${*: -1}" == "${API_KEY_FILE}" ]]; then
            return 1
        fi
        command mv "$@"
    }

    if resume_api_key_bootstrap "true"; then
        fail "failed staged API key promotion should stop bootstrap"
    fi

    assert_equal "old-api-key" "$(<"${API_KEY_FILE}")" "API key after failed promotion"
    assert_equal "0" "${option_update_count}" "option updates after failed promotion"
    [[ -s "${API_KEY_STAGING_FILE}" ]] || fail "staged API key was lost after failed promotion"
}

test_failed_api_key_request_preserves_existing_key() {
    reset_api_key_state
    printf 'old-api-key' > "${API_KEY_FILE}"
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_PORT="8080"
    DSS_USERNAME="dssadmin"
    read_optional_password() { printf 'test-password'; }
    regeneration_option_is_enabled() { return 0; }
    request_api_key() { return 1; }

    if create_api_key "true"; then
        fail "failed API key request should stop bootstrap"
    fi

    assert_equal "old-api-key" "$(<"${API_KEY_FILE}")" "API key after failed replacement"
    [[ ! -e "${API_KEY_STAGING_FILE}" ]] || fail "failed API key request left a staged key"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "failed API key request left pending finalization"
    [[ ! -e "${PASSWORD_FILE}" ]] || fail "failed API key request left the temporary password file"
}

test_empty_api_key_response_preserves_existing_key() {
    reset_api_key_state
    printf 'old-api-key' > "${API_KEY_FILE}"
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_PORT="8080"
    DSS_USERNAME="dssadmin"
    read_optional_password() { printf 'test-password'; }
    request_api_key() { : > "$1"; }

    if create_api_key "true"; then
        fail "empty API key response should stop bootstrap"
    fi

    assert_equal "old-api-key" "$(<"${API_KEY_FILE}")" "API key after empty replacement"
    [[ ! -e "${API_KEY_STAGING_FILE}" ]] || fail "empty API key response left a staged key"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "empty API key response left pending finalization"
}

test_stale_first_start_marker_does_not_consume_regeneration() {
    reset_api_key_state
    printf 'old-api-key' > "${API_KEY_FILE}"
    printf 'false' > "${API_KEY_FINALIZATION_FILE}"
    update_app_option() { fail "stale first-start marker consumed regeneration options"; }

    resume_api_key_bootstrap "true"

    assert_equal "false" "${API_KEY_REQUEST_SATISFIED}" "stale marker regeneration state"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "stale first-start marker was not removed"
}

test_missing_key_marker_allows_api_key_retry() {
    reset_api_key_state
    printf 'true' > "${API_KEY_FINALIZATION_FILE}"
    update_app_option() { fail "missing key marker attempted option cleanup"; }

    resume_api_key_bootstrap "true"

    assert_equal "false" "${API_KEY_REQUEST_SATISFIED}" "missing key retry state"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "missing key marker was not removed"
}

test_empty_marker_does_not_block_valid_key() {
    reset_api_key_state
    printf 'existing-api-key' > "${API_KEY_FILE}"
    : > "${API_KEY_FINALIZATION_FILE}"

    resume_api_key_bootstrap "false"

    assert_equal "existing-api-key" "$(<"${API_KEY_FILE}")" "API key after empty marker"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "empty marker was not removed"
}

test_api_key_request_uses_configured_port() {
    local arguments_file="${TEST_DIR}/api-key-arguments"
    local output_file="${TEST_DIR}/requested-api-key"
    export TEST_ARGUMENTS_FILE="${arguments_file}"
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_PORT="8443"
    DSS_USERNAME="dssadmin"
    PASSWORD_FILE="${TEST_DIR}/password"
    printf 'password' > "${PASSWORD_FILE}"
    cat > "${DIGITALSTROM_MQTT_BIN}" <<'SCRIPT'
#!/usr/bin/env bash
printf '%s\n' "$@" > "${TEST_ARGUMENTS_FILE}"
SCRIPT
    chmod 0755 "${DIGITALSTROM_MQTT_BIN}"

    request_api_key "${output_file}"

    assert_file_contains "-host=dss.local" "${arguments_file}" "API key dSS host"
    assert_file_contains "-port=8443" "${arguments_file}" "API key dSS port"
    assert_file_contains "-api-key-file=${output_file}" "${arguments_file}" "API key output file"
}

test_api_key_creation_uses_password_file_and_cleans_options() {
    local -a calls=()
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_PORT="8080"
    DSS_USERNAME="dssadmin"
    reset_api_key_state
    read_optional_password() { printf 'test-password'; }
    request_api_key() {
        [[ "${PASSWORD_FILE}" == "${PASSWORD_FILE_PREFIX}."* ]] || fail "password file was not created under the configured runtime prefix"
        assert_equal "test-password" "$(<"${PASSWORD_FILE}")" "password file contents"
        printf 'new-api-key' > "$1"
        chmod 0600 "$1"
    }
    update_app_option() { calls+=("$*"); }

    create_api_key "false"

    assert_equal "new-api-key" "$(<"${API_KEY_FILE}")" "stored API key"
    assert_equal 'digitalstrom_password ^""' "${calls[0]}" "password cleanup option"
    [[ ! -e "${PASSWORD_FILE}" ]] || fail "temporary password file was not removed"
}

test_mqtt_service_rejects_unsupported_tls() {
    local log_file="${TEST_DIR}/mqtt-tls-error-log"
    bashio::services() {
        case "$2" in
            host) printf 'mqtt.local' ;;
            port) printf '8883' ;;
            username) printf 'mqtt-user' ;;
            password) printf 'mqtt-password' ;;
            ssl) printf 'true' ;;
            *) return 1 ;;
        esac
    }
    bashio::log.error() { printf 'ERROR: %s\n' "$*" >> "${log_file}"; }

    if read_mqtt_service; then
        fail "unsupported MQTT TLS service was accepted"
    fi

    assert_file_contains \
        "ERROR: The Home Assistant MQTT service requires TLS, but this App cannot verify a custom broker certificate. Use the Mosquitto App's default internal non-TLS service." \
        "${log_file}" \
        "MQTT TLS error"
}

test_main_resumes_pending_cleanup_before_starting_bridge() {
    local calls_file="${TEST_DIR}/resumed-option-updates"
    local log_file="${TEST_DIR}/resumed-bridge-log"
    local started_file="${TEST_DIR}/resumed-bridge-started"
    reset_api_key_state
    printf 'replacement-api-key' > "${API_KEY_FILE}"
    printf 'true' > "${API_KEY_FINALIZATION_FILE}"
    chmod 0600 "${API_KEY_FILE}" "${API_KEY_FINALIZATION_FILE}"
    export TEST_STARTED_FILE="${started_file}"
    cat > "${DIGITALSTROM_MQTT_BIN}" <<'SCRIPT'
#!/usr/bin/env bash
if [[ "$*" == *'-mode=get-api-key'* ]]; then
    exit 99
fi
printf 'started' > "${TEST_STARTED_FILE}"
SCRIPT
    chmod 0755 "${DIGITALSTROM_MQTT_BIN}"
    update_app_option() { printf '%s\n' "$*" >> "${calls_file}"; }
    load_app_options() { APP_OPTIONS='{}'; }
    app_option() {
        case "$1" in
            mqtt_mode) printf 'Home Assistant MQTT service' ;;
            digitalstrom_host) printf 'dss.local' ;;
            digitalstrom_port) printf '8080' ;;
            digitalstrom_username) printf 'dssadmin' ;;
            digitalstrom_password) printf 'temporary-password' ;;
            invert_blinds_position) printf 'false' ;;
            meterings_enabled) printf 'true' ;;
            meterings_interval_seconds) printf '10' ;;
            log_level) printf 'INFO' ;;
            *) return 1 ;;
        esac
    }
    regeneration_option_is_enabled() { return 0; }
    bashio::log.debug() { printf 'DEBUG: %s\n' "$*" >> "${log_file}"; }
    bashio::log.info() { printf 'INFO: %s\n' "$*" >> "${log_file}"; }
    bashio::services() {
        case "$2" in
            host) printf 'mqtt.local' ;;
            port) printf '1883' ;;
            username) printf 'mqtt-user' ;;
            password) printf 'mqtt-password' ;;
            ssl) printf 'false' ;;
            *) return 1 ;;
        esac
    }

    (main)

    [[ -s "${started_file}" ]] || fail "bridge did not start after resuming API key cleanup"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "main left pending API key cleanup"
    assert_file_contains "regenerate_api_key ^false" "${calls_file}" "main regeneration reset"
    assert_file_contains 'digitalstrom_password ^""' "${calls_file}" "main password removal"
    assert_file_contains "INFO: Finishing an interrupted digitalSTROM API key setup." "${log_file}" "resumed API key setup log"
    assert_file_contains "INFO: The digitalSTROM API key setup is complete; the temporary password option is clear." "${log_file}" "completed API key setup log"
    assert_file_not_contains "replacement-api-key" "${log_file}" "resumed API key log secrecy"
    assert_file_not_contains "temporary-password" "${log_file}" "resumed password log secrecy"
}

test_main_starts_bridge_with_expected_environment() {
    local selected_mode="${1:-Home Assistant MQTT service}"
    local expected_broker='mqtt.local'
    if [[ "${selected_mode}" == 'Manual configuration' ]]; then expected_broker='external.example'; fi
    local output_file="${TEST_DIR}/bridge-environment"
    local log_file="${TEST_DIR}/bridge-log"
    export TEST_OUTPUT_FILE="${output_file}"

    printf 'existing-api-key' > "${API_KEY_FILE}"
    rm -f "${API_KEY_STAGING_FILE}" "${API_KEY_FINALIZATION_FILE}"
    cat > "${DIGITALSTROM_MQTT_BIN}" <<'SCRIPT'
#!/usr/bin/env bash
printf '%s\n' \
    "DIGITALSTROM_HOST=${DIGITALSTROM_HOST}" \
    "DIGITALSTROM_PORT=${DIGITALSTROM_PORT}" \
    "DIGITALSTROM_USERNAME=${DIGITALSTROM_USERNAME-unset}" \
    "DIGITALSTROM_PASSWORD=${DIGITALSTROM_PASSWORD-unset}" \
    "DIGITALSTROM_API_KEY=${DIGITALSTROM_API_KEY}" \
    "MQTT_URL=${MQTT_URL}" \
    "MQTT_USERNAME=${MQTT_USERNAME}" \
    "MQTT_PASSWORD=${MQTT_PASSWORD}" \
    "HOME_ASSISTANT_DISCOVERY_ENABLED=${HOME_ASSISTANT_DISCOVERY_ENABLED}" \
    > "${TEST_OUTPUT_FILE}"
SCRIPT
    chmod 0755 "${DIGITALSTROM_MQTT_BIN}"

    load_app_options() { APP_OPTIONS='{}'; }
    app_option() {
        case "$1" in
            mqtt_mode) printf '%s' "${selected_mode}" ;;
            mqtt_host) printf 'external.example' ;;
            mqtt_port) printf '1883' ;;
            mqtt_username) printf 'mqtt-user' ;;
            mqtt_password) printf 'mqtt-password' ;;
            digitalstrom_host) printf 'dss.local' ;;
            digitalstrom_port) printf '8443' ;;
            digitalstrom_username) printf 'dssadmin' ;;
            invert_blinds_position) printf 'false' ;;
            meterings_enabled) printf 'true' ;;
            meterings_interval_seconds) printf '10' ;;
            log_level) printf 'INFO' ;;
            *) return 1 ;;
        esac
    }
    regeneration_option_is_enabled() { return 1; }
    bashio::log.debug() { printf 'DEBUG: %s\n' "$*" >> "${log_file}"; }
    bashio::log.info() { printf 'INFO: %s\n' "$*" >> "${log_file}"; }
    bashio::services() {
        [[ "${selected_mode}" == 'Home Assistant MQTT service' ]] || fail "manual bridge used Supervisor MQTT"
        case "$2" in
            host) printf 'mqtt.local' ;;
            port) printf '1883' ;;
            username) printf 'mqtt-user' ;;
            password) printf 'mqtt-password' ;;
            ssl) printf 'false' ;;
            *) return 1 ;;
        esac
    }

    export DIGITALSTROM_USERNAME="legacy-user"
    export DIGITALSTROM_PASSWORD="legacy-password"
    (main)
    unset DIGITALSTROM_USERNAME DIGITALSTROM_PASSWORD

    assert_file_contains "DIGITALSTROM_HOST=dss.local" "${output_file}" "bridge dSS host"
    assert_file_contains "DIGITALSTROM_PORT=8443" "${output_file}" "bridge dSS port"
    assert_file_contains "DIGITALSTROM_USERNAME=unset" "${output_file}" "bridge omits deprecated dSS username"
    assert_file_contains "DIGITALSTROM_PASSWORD=unset" "${output_file}" "bridge omits deprecated dSS password"
    assert_file_contains "DIGITALSTROM_API_KEY=existing-api-key" "${output_file}" "bridge API key"
    assert_file_contains "MQTT_URL=tcp://${expected_broker}:1883" "${output_file}" "bridge MQTT URL"
    assert_file_contains "MQTT_USERNAME=mqtt-user" "${output_file}" "bridge MQTT username"
    assert_file_contains "MQTT_PASSWORD=mqtt-password" "${output_file}" "bridge MQTT password"
    assert_file_contains "HOME_ASSISTANT_DISCOVERY_ENABLED=true" "${output_file}" "bridge discovery mode"
    assert_file_contains "DEBUG: Using digitalSTROM server dss.local:8443." "${log_file}" "bridge dSS endpoint log"
    assert_file_contains "DEBUG: Using the stored digitalSTROM API key." "${log_file}" "stored API key log"
    assert_file_contains "INFO: Starting digitalSTROM MQTT with the configured broker." "${log_file}" "bridge startup log"
    assert_file_not_contains "existing-api-key" "${log_file}" "API key log secrecy"
    assert_file_not_contains "legacy-password" "${log_file}" "dSS password log secrecy"
    assert_file_not_contains "mqtt-password" "${log_file}" "MQTT password log secrecy"
}

test_configuration_read_failure_is_reported() {
    local log_file="${TEST_DIR}/configuration-error-log"

    load_app_options() { return 1; }
    bashio::log.error() { printf 'ERROR: %s\n' "$*" >> "${log_file}"; }

    if load_configuration; then
        fail "configuration read failure should stop startup"
    fi

    assert_file_contains \
        "ERROR: Home Assistant App configuration could not be read." \
        "${log_file}" \
        "configuration read error"
}

test_configuration_option_failure_is_reported() {
    local log_file="${TEST_DIR}/configuration-option-error-log"

    load_app_options() { APP_OPTIONS='{}'; }
    app_option() {
        if [[ "$1" == "digitalstrom_port" ]]; then
            return 1
        fi
        printf 'unused'
    }
    bashio::log.error() { printf 'ERROR: %s\n' "$*" >> "${log_file}"; }

    if load_configuration; then
        fail "individual configuration option failure should stop startup"
    fi

    assert_file_contains \
        "ERROR: Home Assistant App configuration could not be read." \
        "${log_file}" \
        "configuration option error"
}

test_configuration_applies_bashio_log_level() {
    local applied_log_level=""

    load_app_options() { APP_OPTIONS='{}'; }
    app_option() {
        case "$1" in
            mqtt_mode) printf 'Home Assistant MQTT service' ;;
            digitalstrom_host) printf 'dss.local' ;;
            digitalstrom_port) printf '8080' ;;
            digitalstrom_username) printf 'dssadmin' ;;
            invert_blinds_position) printf 'false' ;;
            meterings_enabled) printf 'true' ;;
            meterings_interval_seconds) printf '10' ;;
            log_level) printf 'DEBUG' ;;
            *) return 1 ;;
        esac
    }
    bashio::log.level() { applied_log_level="$1"; }

    load_configuration

    assert_equal "DEBUG" "${applied_log_level}" "Bashio log level"
    assert_equal "DEBUG" "${BASHIO_LOG_LEVEL}" "stored Bashio log level"
    assert_equal "DEBUG" "${LOG_LEVEL}" "bridge log level"
}

test_configuration_caps_bashio_trace_at_debug() {
    local applied_log_level=""

    load_app_options() { APP_OPTIONS='{}'; }
    app_option() {
        case "$1" in
            mqtt_mode) printf 'Home Assistant MQTT service' ;;
            digitalstrom_host) printf 'dss.local' ;;
            digitalstrom_port) printf '8080' ;;
            digitalstrom_username) printf 'dssadmin' ;;
            invert_blinds_position) printf 'false' ;;
            meterings_enabled) printf 'true' ;;
            meterings_interval_seconds) printf '10' ;;
            log_level) printf 'TRACE' ;;
            *) return 1 ;;
        esac
    }
    bashio::log.level() { applied_log_level="$1"; }

    load_configuration

    assert_equal "DEBUG" "${applied_log_level}" "capped Bashio log level"
    assert_equal "DEBUG" "${BASHIO_LOG_LEVEL}" "stored capped Bashio log level"
    assert_equal "TRACE" "${LOG_LEVEL}" "bridge trace log level"
}

test_watchdog_pause_failure_never_reaches_login() {
    local calls="${TEST_DIR}/pause-calls"
    local attempts=0
    pause_watchdog() {
        attempts=$((attempts + 1))
        printf 'pause\n' >> "${calls}"
        [[ "${attempts}" -gt 1 ]]
    }
    sleep() { printf 'wait\n' >> "${calls}"; }
    main() { printf 'start\n' >> "${calls}"; return 78; }
    if run_app; then fail "permanent configuration failure was accepted"; fi
    assert_equal $'pause\nwait\npause\nstart\npause' "$(<"${calls}")" "pause before login and after failure"
}

test_transient_failure_retries_but_rejected_login_stops() {
    local calls="${TEST_DIR}/retry-calls"
    local marker="${TEST_DIR}/retry-first-attempt"
    pause_watchdog() { printf 'pause\n' >> "${calls}"; }
    sleep() { printf 'retry %s\n' "$1" >> "${calls}"; }
    main() {
        printf 'start\n' >> "${calls}"
        if [[ ! -f "${marker}" ]]; then touch "${marker}"; return 75; fi
        return 78
    }
    if run_app; then fail "rejected credentials should stop startup"; fi
    assert_equal $'pause\nstart\npause\nretry 15\npause\nstart\npause' "$(<"${calls}")" "transient retry and permanent stop"
}

test_crash_leaves_watchdog_decision_to_supervisor() {
    local calls="${TEST_DIR}/crash-calls"
    pause_watchdog() { printf 'pause\n' >> "${calls}"; }
    main() { printf 'start\n' >> "${calls}"; return 2; }
    sleep() { fail "a crash was retried outside Supervisor"; }
    local status=0
    run_app || status=$?
    assert_equal 2 "${status}" "panic exit status"
    assert_equal $'pause\nstart' "$(<"${calls}")" "crash must preserve the selected watchdog state"
}

test_launcher_forwards_manual_stop() {
    local worker="${TEST_DIR}/stop-worker.sh"
    local ready="${TEST_DIR}/stop-ready"
    local stopped="${TEST_DIR}/stop-done"
    printf '%s\n' \
        '#!/usr/bin/env bash' \
        'trap '\''printf stopped > "$2"; exit 0'\'' TERM' \
        'printf ready > "$1"' \
        'while true; do sleep 0.1; done' > "${worker}"
    pause_watchdog() { :; }
    main() { exec bash "${worker}" "${ready}" "${stopped}"; }
    (run_app) &
    local launcher=$!
    trap 'kill -TERM "${launcher}" 2>/dev/null || true; wait "${launcher}" 2>/dev/null || true' EXIT
    local attempt
    for attempt in {1..200}; do
        [[ -f "${ready}" ]] && break
        sleep 0.02
    done
    [[ -f "${ready}" ]] || fail "bridge did not start for stop test"
    kill -TERM "${launcher}"
    wait "${launcher}"
    [[ -f "${stopped}" ]] || fail "manual stop was not forwarded to the bridge"
    trap - EXIT
}

test_api_key_transient_failure_reaches_launcher() {
    reset_api_key_state
    read_optional_password() { printf 'test-password'; }
    request_api_key() { return 75; }
    local status=0
    create_api_key false || status=$?
    assert_equal 75 "${status}" "transient API-key error status"
    [[ ! -f "${PASSWORD_FILE}" ]] || fail "temporary password was not removed"
}

test_password_cleanup_keeps_a_visible_empty_field() {
    APP_OPTIONS='{"digitalstrom_password":"temporary-password","mqtt_password":"broker-secret"}'
    bashio::jq() {
        assert_equal '.digitalstrom_password = ""' "$2" "clear only the temporary password"
        printf '{"digitalstrom_password":"","mqtt_password":"broker-secret"}'
    }
    bashio::var.json() {
        assert_equal '^{"digitalstrom_password":"","mqtt_password":"broker-secret"}' "$2" "retain broker password"
        printf '{"options":{"digitalstrom_password":"","mqtt_password":"broker-secret"}}'
    }
    bashio::api.supervisor() {
        assert_equal '/addons/self/options' "$2" "self options endpoint"
        assert_equal '{"options":{"digitalstrom_password":"","mqtt_password":"broker-secret"}}' "$3" "cleared password payload"
    }
    bashio::cache.flush_all() { :; }
    update_app_option 'digitalstrom_password' '^""'
    assert_equal '{"digitalstrom_password":"","mqtt_password":"broker-secret"}' "${APP_OPTIONS}" "empty field retained"
}

test_mqtt_modes_and_validation() {
    local selected_mode='Manual configuration'
    local host='broker.example'
    local port='1884'
    local username='external-user'
    local password='external-secret'
    local service_calls="${TEST_DIR}/mqtt-service-calls"
    app_option() {
        case "$1" in
            mqtt_mode) printf '%s' "${selected_mode}" ;;
            mqtt_host) printf '%s' "${host}" ;;
            mqtt_port) printf '%s' "${port}" ;;
            mqtt_username) printf '%s' "${username}" ;;
            mqtt_password) printf '%s' "${password}" ;;
            *) fail "unexpected option" ;;
        esac
    }
    bashio::services() { printf 'called\n' >> "${service_calls}"; return 1; }
    read_mqtt_configuration
    assert_equal 'tcp://broker.example:1884' "${MQTT_URL}" "manual broker URL"
    assert_equal 'external-user' "${MQTT_USERNAME}" "manual username"
    assert_equal 'external-secret' "${MQTT_PASSWORD}" "manual password"
    [[ ! -f "${service_calls}" ]] || fail "manual configuration consulted Supervisor MQTT"

    host='2001:db8::1'
    username=''
    password=''
    read_mqtt_configuration
    assert_equal 'tcp://[2001:db8::1]:1884' "${MQTT_URL}" "IPv6 broker URL"
    assert_equal '' "${MQTT_USERNAME}" "anonymous username must clear old value"
    assert_equal '' "${MQTT_PASSWORD}" "anonymous password must clear old value"

    for host in '' 'tcp://broker.example' 'user:secret@broker.example' 'broker.example/path' 'broker example' 'broker.example:1883'; do
        if read_mqtt_configuration; then fail "invalid broker host accepted"; fi
    done
    host='broker.example'
    for port in '' 0 65536 -1 '1883/secret' 999999999999; do
        if read_mqtt_configuration; then fail "invalid broker port accepted"; fi
    done
    port='1883'
    selected_mode='unknown'
    if read_mqtt_configuration; then fail "unknown MQTT mode accepted"; fi
    selected_mode='Home Assistant MQTT service'
    if read_mqtt_configuration; then fail "missing service silently fell back to manual broker"; fi
    [[ -f "${service_calls}" ]] || fail "service mode did not consult Supervisor"
}

test_service_mode_ignores_manual_credentials() {
    app_option() {
        assert_equal 'mqtt_mode' "$1" "service mode must ignore manual fields"
        assert_equal '"Home Assistant MQTT service"' "$2" "existing installations retain service default"
        printf 'Home Assistant MQTT service'
    }
    bashio::services() {
        case "$2" in
            host) printf 'core-mosquitto' ;;
            port) printf '1883' ;;
            username) printf 'service-user' ;;
            password) printf 'service-secret' ;;
            ssl) printf 'false' ;;
            *) return 1 ;;
        esac
    }
    MQTT_USERNAME='old-manual-user'
    MQTT_PASSWORD='old-manual-password'
    read_mqtt_configuration
    assert_equal 'tcp://core-mosquitto:1883' "${MQTT_URL}" "service URL"
    assert_equal 'service-user' "${MQTT_USERNAME}" "service username"
    assert_equal 'service-secret' "${MQTT_PASSWORD}" "service password"
}

test_invalid_mqtt_configuration_stops_before_dss_login() {
    load_configuration() { DIGITALSTROM_HOST='dss.local'; }
    read_mqtt_configuration() { return 1; }
    resume_api_key_bootstrap() { fail "invalid MQTT settings reached dSS bootstrap"; }
    if main; then fail "invalid MQTT settings accepted"; fi
}

(test_password_cleanup_keeps_a_visible_empty_field)
(test_mqtt_modes_and_validation)
(test_service_mode_ignores_manual_credentials)
(test_invalid_mqtt_configuration_stops_before_dss_login)
(test_watchdog_pause_failure_never_reaches_login)
(test_transient_failure_retries_but_rejected_login_stops)
(test_crash_leaves_watchdog_decision_to_supervisor)
(test_launcher_forwards_manual_stop)
(test_api_key_transient_failure_reaches_launcher)
(test_boolean_false_is_not_replaced_by_default)
(test_first_start_removes_password_without_resetting_regeneration)
(test_regeneration_resets_flag_before_removing_password)
(test_failed_regeneration_reset_keeps_password)
(test_already_applied_option_updates_are_not_repeated)
(test_password_option_read_failure_stops_bootstrap)
(test_option_update_propagates_supervisor_failure)
(test_failed_regeneration_cleanup_resumes_without_another_key)
(test_failed_password_removal_resumes_without_another_key)
(test_interrupted_staged_key_is_promoted_without_another_request)
(test_failed_staged_key_promotion_does_not_finalize_options)
(test_failed_api_key_request_preserves_existing_key)
(test_empty_api_key_response_preserves_existing_key)
(test_stale_first_start_marker_does_not_consume_regeneration)
(test_missing_key_marker_allows_api_key_retry)
(test_empty_marker_does_not_block_valid_key)
(test_api_key_request_uses_configured_port)
(test_api_key_creation_uses_password_file_and_cleans_options)
(test_mqtt_service_rejects_unsupported_tls)
(test_main_resumes_pending_cleanup_before_starting_bridge)
(test_main_starts_bridge_with_expected_environment)
(test_main_starts_bridge_with_expected_environment 'Manual configuration')
(test_configuration_read_failure_is_reported)
(test_configuration_option_failure_is_reported)
(test_configuration_applies_bashio_log_level)
(test_configuration_caps_bashio_trace_at_debug)

printf 'HA App runtime tests passed\n'
