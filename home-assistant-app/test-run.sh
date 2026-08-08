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

test_first_start_removes_password_without_resetting_regeneration() {
    local -a calls=()
    update_app_option() { calls+=("$*"); }

    finalize_api_key_options "false"

    assert_equal "1" "${#calls[@]}" "first start option update count"
    assert_equal "digitalstrom_password" "${calls[0]}" "first start option update"
}

test_regeneration_resets_flag_before_removing_password() {
    local -a calls=()
    update_app_option() { calls+=("$*"); }

    finalize_api_key_options "true"

    assert_equal "2" "${#calls[@]}" "regeneration option update count"
    assert_equal "regenerate_api_key ^false" "${calls[0]}" "first regeneration option update"
    assert_equal "digitalstrom_password" "${calls[1]}" "second regeneration option update"
}

test_failed_regeneration_reset_keeps_password() {
    local -a calls=()
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

test_api_key_creation_uses_password_file_and_cleans_options() {
    local -a calls=()
    DIGITALSTROM_HOST="dss.local"
    DIGITALSTROM_USERNAME="dssadmin"
    read_optional_password() { printf 'test-password'; }
    request_api_key() {
        assert_equal "test-password" "$(<"${PASSWORD_FILE}")" "password file contents"
        printf 'new-api-key' > "${API_KEY_FILE}"
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

test_main_starts_bridge_with_expected_environment() {
    local output_file="${TEST_DIR}/bridge-environment"
    export TEST_OUTPUT_FILE="${output_file}"

    printf 'existing-api-key' > "${API_KEY_FILE}"
    cat > "${DIGITALSTROM_MQTT_BIN}" <<'SCRIPT'
#!/usr/bin/env bash
printf '%s\n' \
    "DIGITALSTROM_HOST=${DIGITALSTROM_HOST}" \
    "DIGITALSTROM_USERNAME=${DIGITALSTROM_USERNAME}" \
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

    (main)

    assert_file_contains "DIGITALSTROM_HOST=dss.local" "${output_file}" "bridge dSS host"
    assert_file_contains "DIGITALSTROM_USERNAME=dssadmin" "${output_file}" "bridge dSS username"
    assert_file_contains "DIGITALSTROM_API_KEY=existing-api-key" "${output_file}" "bridge API key"
    assert_file_contains "MQTT_URL=tcp://mqtt.local:1883" "${output_file}" "bridge MQTT URL"
    assert_file_contains "MQTT_USERNAME=mqtt-user" "${output_file}" "bridge MQTT username"
    assert_file_contains "MQTT_PASSWORD=mqtt-password" "${output_file}" "bridge MQTT password"
    assert_file_contains "HOME_ASSISTANT_DISCOVERY_ENABLED=true" "${output_file}" "bridge discovery mode"
}

test_first_start_removes_password_without_resetting_regeneration
test_regeneration_resets_flag_before_removing_password
test_failed_regeneration_reset_keeps_password
test_api_key_creation_uses_password_file_and_cleans_options
test_mqtt_service_enables_tls_scheme
test_main_starts_bridge_with_expected_environment

printf 'HA App runtime tests passed\n'
