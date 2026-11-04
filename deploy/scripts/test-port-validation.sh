#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# ClientPulse Deployment Preflight Port Ownership Validation Test Suite
# Tests all 9 scenarios specified in TASK-DEPLOY-001 safely in a mocked sandbox.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/deploy-production.sh"

if [[ ! -f "${DEPLOY_SCRIPT}" ]]; then
    echo "❌ Error: deploy-production.sh not found at ${DEPLOY_SCRIPT}"
    exit 1
fi

# Extract validation functions directly from deploy-production.sh
EXTRACTED_CODE=$(sed -n '/# BEGIN: PORT_VALIDATION_FUNCTIONS/,/# END: PORT_VALIDATION_FUNCTIONS/p' "${DEPLOY_SCRIPT}")
if [[ -z "${EXTRACTED_CODE}" ]]; then
    echo "❌ FATAL: Could not extract port validation code from ${DEPLOY_SCRIPT}!"
    exit 1
fi

eval "${EXTRACTED_CODE}"

# Create sandbox temporary directories for mock binaries and scenario states
MOCK_BIN_DIR=$(mktemp -d "/tmp/clientpulse-mock-bin.XXXXXX")
MOCK_STATE_DIR=$(mktemp -d "/tmp/clientpulse-mock-state.XXXXXX")

cleanup() {
    rm -rf "${MOCK_BIN_DIR}" "${MOCK_STATE_DIR}"
}
trap cleanup EXIT

# Create mock 'docker' binary
cat << 'EOF' > "${MOCK_BIN_DIR}/docker"
#!/usr/bin/env bash
set -euo pipefail

STATE_DIR="${CLIENTPULSE_TEST_STATE_DIR}"

if [[ "$1" == "inspect" ]]; then
    # Parse container name (last argument)
    CONTAINER="${@: -1}"
    ERR_FILE="${STATE_DIR}/docker_inspect_err_${CONTAINER}"
    OUT_FILE="${STATE_DIR}/docker_inspect_out_${CONTAINER}"

    if [[ -f "${ERR_FILE}" ]]; then
        cat "${ERR_FILE}" >&2
        exit 1
    elif [[ -f "${OUT_FILE}" ]]; then
        cat "${OUT_FILE}"
        exit 0
    else
        echo "Error: No such container: ${CONTAINER}" >&2
        exit 1
    fi
elif [[ "$1" == "ps" ]]; then
    # Check if --filter publish=<port> is specified
    PORT=""
    for arg in "$@"; do
        if [[ "$arg" =~ publish=([0-9]+) ]]; then
            PORT="${BASH_REMATCH[1]}"
            break
        fi
    done

    if [[ -n "${PORT}" ]] && [[ -f "${STATE_DIR}/docker_ps_publish_${PORT}" ]]; then
        cat "${STATE_DIR}/docker_ps_publish_${PORT}"
    fi
    exit 0
fi

exit 0
EOF
chmod +x "${MOCK_BIN_DIR}/docker"

# Create mock 'ss' binary
cat << 'EOF' > "${MOCK_BIN_DIR}/ss"
#!/usr/bin/env bash
set -euo pipefail

STATE_DIR="${CLIENTPULSE_TEST_STATE_DIR}"

# Check for failure simulation
if [[ -f "${STATE_DIR}/ss_fail" ]]; then
    echo "ss: kernel netlink failure" >&2
    exit 1
fi

# Detect queried port from arguments
PORT=""
for arg in "$@"; do
    if [[ "$arg" =~ :([0-9]+) ]]; then
        PORT="${BASH_REMATCH[1]}"
        break
    fi
done

if [[ -n "${PORT}" ]] && [[ -f "${STATE_DIR}/ss_${PORT}" ]]; then
    cat "${STATE_DIR}/ss_${PORT}"
fi
exit 0
EOF
chmod +x "${MOCK_BIN_DIR}/ss"

export PATH="${MOCK_BIN_DIR}:${PATH}"
export CLIENTPULSE_TEST_STATE_DIR="${MOCK_STATE_DIR}"
export DOCKER_CMD="docker"
export SS_CMD="ss"

# Test Runner State
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

reset_mock_state() {
    rm -rf "${MOCK_STATE_DIR}"/*
}

run_scenario() {
    local scenario_num="$1"
    local scenario_desc="$2"
    local expected_result="$3" # "PASS" or "FAIL"

    TESTS_RUN=$((TESTS_RUN + 1))
    echo "----------------------------------------------------------------------"
    echo "Scenario ${scenario_num}: ${scenario_desc}"
    echo "Expected Outcome: ${expected_result}"

    local output=""
    local exit_code=0
    set +e
    output=$(check_clientpulse_ports 2>&1)
    exit_code=$?
    set -e

    local actual_result="FAIL"
    if [[ ${exit_code} -eq 0 ]]; then
        actual_result="PASS"
    fi

    echo "--- Output ---"
    echo "${output}"
    echo "--------------"

    if [[ "${actual_result}" == "${expected_result}" ]]; then
        echo "Result: ✅ PASS (Exit Code: ${exit_code})"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "Result: ❌ FAILED! (Expected: ${expected_result}, Got: ${actual_result}, Exit Code: ${exit_code})"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    echo ""
}

# Setup standard verified ClientPulse service states
setup_valid_web() {
    cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
    cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-web-1"
true|clientpulse|web|3000/tcp->127.0.0.1:3100 
EOF
}

setup_valid_api() {
    cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 4096 127.0.0.1:8090 0.0.0.0:* users:(("docker-proxy",pid=1002,fd=4))
EOF
    cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-api-1"
true|clientpulse|api|8080/tcp->127.0.0.1:8090 
EOF
}

echo "======================================================================"
echo " Starting ClientPulse Preflight Port Validation Test Suite"
echo "======================================================================"
echo ""

# ------------------------------------------------------------------------------
# 1. Both ports free → PASS
# ------------------------------------------------------------------------------
reset_mock_state
run_scenario "1" "Both ports free (First installation)" "PASS"

# ------------------------------------------------------------------------------
# 2. Both ports owned by the exact expected ClientPulse containers → PASS
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_web
setup_valid_api
run_scenario "2" "Both ports owned by expected ClientPulse containers" "PASS"

# ------------------------------------------------------------------------------
# 3. Port 3100 occupied by an unrelated container → FAIL
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
echo "unrelated-frontend-container" > "${MOCK_STATE_DIR}/docker_ps_publish_3100"
run_scenario "3" "Port 3100 occupied by an unrelated container" "FAIL"

# ------------------------------------------------------------------------------
# 4. Port 8090 occupied by an unrelated container → FAIL
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_web
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 4096 127.0.0.1:8090 0.0.0.0:* users:(("docker-proxy",pid=1002,fd=4))
EOF
echo "unrelated-backend-service" > "${MOCK_STATE_DIR}/docker_ps_publish_8090"
run_scenario "4" "Port 8090 occupied by an unrelated container" "FAIL"

# ------------------------------------------------------------------------------
# 5. Port occupied by a host process → FAIL
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 1 127.0.0.1:3100 0.0.0.0:* users:(("python3",pid=49201,fd=3))
EOF
run_scenario "5" "Port 3100 occupied by a host process (python3)" "FAIL"

# ------------------------------------------------------------------------------
# 6. Expected container exists but has incorrect Compose labels → FAIL
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
# Incorrect Compose project and service labels
cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-web-1"
true|otherproject|otherservice|3000/tcp->127.0.0.1:3100 
EOF
run_scenario "6" "Expected container has incorrect Compose labels" "FAIL"

# ------------------------------------------------------------------------------
# 7. Expected container has incorrect host port binding → FAIL
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
# Bound to 0.0.0.0 instead of loopback 127.0.0.1
cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-web-1"
true|clientpulse|web|3000/tcp->0.0.0.0:3100 
EOF
run_scenario "7" "Expected container has incorrect host IP binding (0.0.0.0 instead of 127.0.0.1)" "FAIL"

# ------------------------------------------------------------------------------
# 8. Docker inspection fails → FAIL safely
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
# Docker daemon communication failure
echo "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?" > "${MOCK_STATE_DIR}/docker_inspect_err_clientpulse-web-1"
run_scenario "8" "Docker inspection fails (daemon unreachable)" "FAIL"

# ------------------------------------------------------------------------------
# 9. One expected container is missing while its port is occupied → FAIL
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
# clientpulse-web-1 is completely missing from Docker, port 3100 occupied
run_scenario "9" "One expected container is missing while its port is occupied" "FAIL"

# ------------------------------------------------------------------------------
# Bonus Edge Cases:
# 10. Container exists but is not running (stopped / exited)
# 11. Host socket listening on 0.0.0.0 instead of loopback 127.0.0.1
# 12. Port 8090 occupied by host process
# 13. Port 8090 expected container has incorrect Compose label
# 14. Port 8090 expected container has incorrect host port binding
# 15. Port 8090 Docker inspection fails
# 16. Port 8090 expected container is missing while port is occupied
# ------------------------------------------------------------------------------
reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 127.0.0.1:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-web-1"
false|clientpulse|web|3000/tcp->127.0.0.1:3100 
EOF
run_scenario "10 (Bonus)" "Expected container is stopped (is_running=false) while port is occupied" "FAIL"

reset_mock_state
setup_valid_api
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_3100"
LISTEN 0 4096 0.0.0.0:3100 0.0.0.0:* users:(("docker-proxy",pid=1001,fd=4))
EOF
cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-web-1"
true|clientpulse|web|3000/tcp->127.0.0.1:3100 
EOF
run_scenario "11 (Bonus)" "Host socket listening on 0.0.0.0:3100 instead of 127.0.0.1:3100" "FAIL"

reset_mock_state
setup_valid_web
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 1 127.0.0.1:8090 0.0.0.0:* users:(("nginx",pid=7812,fd=6))
EOF
run_scenario "12 (Bonus)" "Port 8090 occupied by host process (nginx)" "FAIL"

reset_mock_state
setup_valid_web
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 4096 127.0.0.1:8090 0.0.0.0:* users:(("docker-proxy",pid=1002,fd=4))
EOF
cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-api-1"
true|otherproject|api|8080/tcp->127.0.0.1:8090 
EOF
run_scenario "13 (Bonus)" "Port 8090 expected container has incorrect Compose label" "FAIL"

reset_mock_state
setup_valid_web
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 4096 127.0.0.1:8090 0.0.0.0:* users:(("docker-proxy",pid=1002,fd=4))
EOF
cat << 'EOF' > "${MOCK_STATE_DIR}/docker_inspect_out_clientpulse-api-1"
true|clientpulse|api|8080/tcp->127.0.0.1:8091 
EOF
run_scenario "14 (Bonus)" "Port 8090 expected container has incorrect host port binding (8091 instead of 8090)" "FAIL"

reset_mock_state
setup_valid_web
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 4096 127.0.0.1:8090 0.0.0.0:* users:(("docker-proxy",pid=1002,fd=4))
EOF
echo "Docker daemon I/O error" > "${MOCK_STATE_DIR}/docker_inspect_err_clientpulse-api-1"
run_scenario "15 (Bonus)" "Port 8090 Docker inspection fails (daemon error)" "FAIL"

reset_mock_state
setup_valid_web
cat << 'EOF' > "${MOCK_STATE_DIR}/ss_8090"
LISTEN 0 4096 127.0.0.1:8090 0.0.0.0:* users:(("docker-proxy",pid=1002,fd=4))
EOF
run_scenario "16 (Bonus)" "Port 8090 expected container is missing while port is occupied" "FAIL"

echo "======================================================================"
echo " Test Summary: ${TESTS_PASSED}/${TESTS_RUN} Passed, ${TESTS_FAILED} Failed"
echo "======================================================================"

if [[ ${TESTS_FAILED} -ne 0 ]]; then
    echo "❌ Some test scenarios failed!"
    exit 1
fi

echo "🎉 All port ownership validation test scenarios passed successfully!"
exit 0
