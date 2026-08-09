#!/usr/bin/env bash
set -euo pipefail

readonly TEST_DIR="$(mktemp -d)"
readonly APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cleanup() {
    rm -rf "${TEST_DIR}"
}
trap cleanup EXIT

export API_KEY_FILE="${TEST_DIR}/digitalstrom-api-key"
export DIGITALSTROM_MQTT_BIN="${TEST_DIR}/digitalstrom-mqtt"

# shellcheck source=run.sh
source "${APP_DIR}/run.sh"

bashio::log.error() { :; }
bashio::log.info() { :; }
bashio::log.warning() { :; }

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

reset_api_key_state() {
    rm -f "${API_KEY_FILE}" "${API_KEY_STAGING_FILE}" "${API_KEY_FINALIZATION_FILE}"
    API_KEY_BOOTSTRAP_RESUMED="false"
}

test_first_start_removes_password_without_resetting_regeneration() {
    local -a calls=()
    read_optional_password() { printf 'temporary-password'; }
    update_app_option() { calls+=("$*"); }

    finalize_api_key_options "false"

    assert_equal "1" "${#calls[@]}" "first start option update count"
    assert_equal "digitalstrom_password" "${calls[0]}" "first start option update"
}

test_regeneration_resets_flag_before_removing_password() {
    local -a calls=()
    regeneration_option_is_enabled() { return 0; }
    read_optional_password() { printf 'temporary-password'; }
    update_app_option() { calls+=("$*"); }

    finalize_api_key_options "true"

    assert_equal "2" "${#calls[@]}" "regeneration option update count"
    assert_equal "regenerate_api_key ^false" "${calls[0]}" "first regeneration option update"
    assert_equal "digitalstrom_password" "${calls[1]}" "second regeneration option update"
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

    if create_api_key "true"; then
        fail "failed regeneration cleanup should stop bootstrap"
    fi
    assert_equal "new-api-key" "$(<"${API_KEY_FILE}")" "replacement API key"
    assert_equal "true" "$(<"${API_KEY_FINALIZATION_FILE}")" "pending regeneration state"
    assert_equal "1" "${request_count}" "initial API key request count"

    allow_option_updates="true"
    resume_api_key_bootstrap "true"

    assert_equal "1" "${request_count}" "resumed API key request count"
    assert_equal "true" "${API_KEY_BOOTSTRAP_RESUMED}" "resumed bootstrap marker"
    [[ ! -e "${API_KEY_FINALIZATION_FILE}" ]] || fail "pending regeneration state was not removed"
    assert_equal "3" "${#calls[@]}" "regeneration retry option update count"
    assert_equal "regenerate_api_key ^false" "${calls[1]}" "retried regeneration reset"
    assert_equal "digitalstrom_password" "${calls[2]}" "retried password removal"
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

    if create_api_key "false"; then
        fail "failed password removal should stop bootstrap"
    fi
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
    assert_equal "true" "${API_KEY_BOOTSTRAP_RESUMED}" "staged bootstrap marker"
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
        assert_equal "test-password" "$(<"${PASSWORD_FILE}")" "password file contents"
        printf 'new-api-key' > "$1"
        chmod 0600 "$1"
    }
    update_app_option() { calls+=("$*"); }

    create_api_key "false"

    assert_equal "new-api-key" "$(<"${API_KEY_FILE}")" "stored API key"
    assert_equal "digitalstrom_password" "${calls[0]}" "password cleanup option"
    [[ ! -e "${PASSWORD_FILE}" ]] || fail "temporary password file was not removed"
}

test_mqtt_service_enables_tls_scheme() {
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

    read_mqtt_service

    assert_equal "mqtt.local" "${MQTT_HOST}" "MQTT host"
    assert_equal "8883" "${MQTT_PORT}" "MQTT port"
    assert_equal "ssl" "${MQTT_SCHEME}" "MQTT scheme"
}

test_main_resumes_pending_cleanup_before_starting_bridge() {
    local calls_file="${TEST_DIR}/resumed-option-updates"
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
    bashio::config() {
        case "$1" in
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
    bashio::config.true() { [[ "$1" == "regenerate_api_key" ]]; }
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
    assert_file_contains "digitalstrom_password" "${calls_file}" "main password removal"
}

test_main_starts_bridge_with_expected_environment() {
    local output_file="${TEST_DIR}/bridge-environment"
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

    bashio::config() {
        case "$1" in
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
    bashio::config.true() { return 1; }
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

    export DIGITALSTROM_USERNAME="legacy-user"
    export DIGITALSTROM_PASSWORD="legacy-password"
    (main)
    unset DIGITALSTROM_USERNAME DIGITALSTROM_PASSWORD

    assert_file_contains "DIGITALSTROM_HOST=dss.local" "${output_file}" "bridge dSS host"
    assert_file_contains "DIGITALSTROM_PORT=8443" "${output_file}" "bridge dSS port"
    assert_file_contains "DIGITALSTROM_USERNAME=unset" "${output_file}" "bridge omits deprecated dSS username"
    assert_file_contains "DIGITALSTROM_PASSWORD=unset" "${output_file}" "bridge omits deprecated dSS password"
    assert_file_contains "DIGITALSTROM_API_KEY=existing-api-key" "${output_file}" "bridge API key"
    assert_file_contains "MQTT_URL=tcp://mqtt.local:1883" "${output_file}" "bridge MQTT URL"
    assert_file_contains "MQTT_USERNAME=mqtt-user" "${output_file}" "bridge MQTT username"
    assert_file_contains "MQTT_PASSWORD=mqtt-password" "${output_file}" "bridge MQTT password"
    assert_file_contains "HOME_ASSISTANT_DISCOVERY_ENABLED=true" "${output_file}" "bridge discovery mode"
}

(test_first_start_removes_password_without_resetting_regeneration)
(test_regeneration_resets_flag_before_removing_password)
(test_failed_regeneration_reset_keeps_password)
(test_already_applied_option_updates_are_not_repeated)
(test_password_option_read_failure_stops_bootstrap)
(test_failed_regeneration_cleanup_resumes_without_another_key)
(test_failed_password_removal_resumes_without_another_key)
(test_interrupted_staged_key_is_promoted_without_another_request)
(test_failed_staged_key_promotion_does_not_finalize_options)
(test_failed_api_key_request_preserves_existing_key)
(test_api_key_request_uses_configured_port)
(test_api_key_creation_uses_password_file_and_cleans_options)
(test_mqtt_service_enables_tls_scheme)
(test_main_resumes_pending_cleanup_before_starting_bridge)
(test_main_starts_bridge_with_expected_environment)

printf 'HA App runtime tests passed\n'
