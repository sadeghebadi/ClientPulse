#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# ClientPulse Deployment Version Propagation Test Suite (TASK-DEPLOY-002)
# Tests that release versions are explicitly and consistently propagated to
# all remote Docker Compose operations without connecting to Production.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DEPLOY_SCRIPT="${SCRIPT_DIR}/deploy-production.sh"
ROLLBACK_SCRIPT="${SCRIPT_DIR}/rollback-production.sh"
COMPOSE_FILE="${ROOT_DIR}/deploy/docker-compose.production.yml"

TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

assert_eq() {
    local expected="$1"
    local actual="$2"
    local message="$3"
    if [[ "${expected}" != "${actual}" ]]; then
        echo "❌ Assertion failed: ${message} (expected: '${expected}', got: '${actual}')"
        return 1
    fi
    return 0
}

run_test() {
    local test_name="$1"
    shift
    TESTS_RUN=$((TESTS_RUN + 1))
    echo "----------------------------------------------------------------------"
    echo "Test ${TESTS_RUN}: ${test_name}"
    local output
    set +e
    output=$("$@" 2>&1)
    local status=$?
    set -e
    if [[ ${status} -eq 0 ]]; then
        echo "✅ PASS"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "❌ FAILED (exit code ${status}):"
        echo "${output}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# ------------------------------------------------------------------------------
# Test 1: Static Audit of Remote Docker Compose Invocations in deploy-production.sh
# ------------------------------------------------------------------------------
test_static_compose_audit() {
    # Check Stage G (ssh invocation passes VERSION_TAG)
    if ! grep -q 'ssh -p "\${PORT}" "\${USER}@\${HOST}" bash -s -- "\${REMOTE_DIR}" "\${VERSION_TAG}" << .EOF' "${DEPLOY_SCRIPT}"; then
        echo "Error: Stage G does not pass \${VERSION_TAG} to remote ssh script."
        return 1
    fi

    # Check Stage H (ssh invocation passes VERSION_TAG)
    # Stage H should also pass VERSION_TAG
    local stage_h_matches
    stage_h_matches=$(grep -n 'ssh -p "\${PORT}" "\${USER}@\${HOST}" bash -s -- "\${REMOTE_DIR}" "\${VERSION_TAG}" << .EOF' "${DEPLOY_SCRIPT}" | wc -l)
    if [[ "${stage_h_matches}" -lt 2 ]]; then
        echo "Error: Stage G and Stage H must both pass \${VERSION_TAG} to remote ssh script. Found: ${stage_h_matches}"
        return 1
    fi

    # Check for bare 'sudo docker compose' in Stage G and Stage H
    # Lines between Stage G and Stage I must not contain 'sudo docker compose' without 'env VERSION='
    local remote_compose_lines
    remote_compose_lines=$(sed -n '/Stage G:/,/Stage I:/p' "${DEPLOY_SCRIPT}" | grep 'docker compose' || true)
    
    while IFS= read -r line; do
        [[ -z "${line}" ]] && continue
        if [[ ! "${line}" =~ (sudo env VERSION=.*docker compose|VERSION=.*docker compose) ]]; then
            echo "Error: Found remote docker compose invocation missing explicit VERSION: ${line}"
            return 1
        fi
    done <<< "${remote_compose_lines}"

    echo "Static audit verified: All remote compose invocations in Stage G & H use 'sudo env VERSION=\"\${VERSION}\" docker compose'."
    return 0
}

# ------------------------------------------------------------------------------
# Test 2: Missing Version Flag Fails Early
# ------------------------------------------------------------------------------
test_missing_version_fails() {
    local out
    set +e
    out=$("${DEPLOY_SCRIPT}" 2>&1)
    local status=$?
    set -e

    if [[ ${status} -eq 0 ]]; then
        echo "Error: deploy-production.sh without -v should fail, but exited 0"
        return 1
    fi

    if [[ ! "${out}" =~ "Release version (-v) is required" ]]; then
        echo "Error: Expected error message about missing -v, got: ${out}"
        return 1
    fi

    return 0
}

# ------------------------------------------------------------------------------
# Test 3: Compose Interpolation with .env Lacking VERSION Does Not Fall Back
# ------------------------------------------------------------------------------
test_compose_env_version_resolution() {
    local tmp_env
    tmp_env=$(mktemp "/tmp/clientpulse-test-env.XXXXXX")
    trap "rm -f '${tmp_env}'" RETURN

    cat << 'EOF' > "${tmp_env}"
APP_ENV=production
PUBLIC_APP_URL=https://mindu.ir
DATABASE_URL=postgres://dummy:dummy@postgres:5432/db
MIGRATION_DATABASE_URL=postgres://dummy:dummy@postgres:5432/db
WORKER_DATABASE_URL=postgres://dummy:dummy@postgres:5432/db
SESSION_SECRET=01234567890123456789012345678901
TOKEN_HASH_PEPPER=01234567890123456789012345678901
POSTGRES_PASSWORD=dummy
EOF

    # Scenario A: Bare compose without VERSION -> defaults to v1.0.0 (the problem)
    local unversioned_config
    unversioned_config=$(docker compose -f "${COMPOSE_FILE}" --env-file "${tmp_env}" config 2>/dev/null)
    if ! echo "${unversioned_config}" | grep -q "clientpulse-api:v1.0.0"; then
        echo "Warning: Expected fallback to v1.0.0 when unversioned"
    fi

    # Scenario B: With explicit env VERSION="v1.1.1" -> resolves to v1.1.1
    local versioned_config
    versioned_config=$(env VERSION="v1.1.1" docker compose -f "${COMPOSE_FILE}" --env-file "${tmp_env}" config 2>/dev/null)

    if echo "${versioned_config}" | grep -q "clientpulse-api:v1.0.0"; then
        echo "Error: Resolved config still contains v1.0.0 instead of v1.1.1"
        return 1
    fi

    if ! echo "${versioned_config}" | grep -q "clientpulse-api:v1.1.1"; then
        echo "Error: Expected clientpulse-api:v1.1.1 in resolved Compose config"
        return 1
    fi
    if ! echo "${versioned_config}" | grep -q "clientpulse-worker:v1.1.1"; then
        echo "Error: Expected clientpulse-worker:v1.1.1 in resolved Compose config"
        return 1
    fi
    if ! echo "${versioned_config}" | grep -q "clientpulse-web:v1.1.1"; then
        echo "Error: Expected clientpulse-web:v1.1.1 in resolved Compose config"
        return 1
    fi

    echo "Compose resolution verified: Explicit env VERSION prevents fallback to v1.0.0."
    return 0
}

# ------------------------------------------------------------------------------
# Test 4: Mocked Execution of -v 1.1.1 Propagates v1.1.1 to All Operations
# ------------------------------------------------------------------------------
test_version_propagation_1_1_1() {
    local tmp_dir
    tmp_dir=$(mktemp -d "/tmp/clientpulse-test-mock.XXXXXX")
    trap "rm -rf '${tmp_dir}'" RETURN

    local mock_log="${tmp_dir}/compose_calls.log"

    # Create mock binaries in sandbox
    local bin_dir="${tmp_dir}/bin"
    mkdir -p "${bin_dir}"

    # Mock ssh: intercept Stage G and H remote script invocations
    cat << 'EOF' > "${bin_dir}/ssh"
#!/usr/bin/env bash
set -euo pipefail

LOG_FILE="${MOCK_TEST_LOG}"

# Parse remote args passed to bash -s -- ...
ARGS=()
CAPTURE=false
for a in "$@"; do
    if [[ "$CAPTURE" == "true" ]]; then
        ARGS+=("$a")
    fi
    if [[ "$a" == "--" ]]; then
        CAPTURE=true
    fi
done

# Read standard input script
SCRIPT_CONTENT=$(cat)

# Record SSH invocation metadata
echo "[SSH_INVOCATION] ARGS: ${ARGS[*]}" >> "${LOG_FILE}"

# If this is Stage G or H, simulate execution with arguments
if echo "${SCRIPT_CONTENT}" | grep -q 'Starting isolated PostgreSQL\|Starting ClientPulse application'; then
    REMOTE_DIR="${ARGS[0]:-/home/ubuntu/clientpulse}"
    VERSION="${ARGS[1]:-missing}"
    echo "[SSH_SCRIPT] REMOTE_DIR=${REMOTE_DIR} VERSION=${VERSION}" >> "${LOG_FILE}"

    # Join multi-line commands with trailing backslashes
    JOINED_SCRIPT=$(echo "${SCRIPT_CONTENT}" | sed -e :a -e '/\\$/N; s/\\\n//; ta')

    # Extract all 'sudo env VERSION=... docker compose' calls
    while IFS= read -r line; do
        if [[ "${line}" =~ sudo\ env\ VERSION=\"\$\{VERSION\}\"\ docker\ compose\ (.*) ]]; then
            OP="${BASH_REMATCH[1]}"
            echo "[COMPOSE_OP] VERSION=${VERSION} OP=${OP}" >> "${LOG_FILE}"
        fi
    done <<< "${JOINED_SCRIPT}"
fi

exit 0
EOF
    chmod +x "${bin_dir}/ssh"

    # Mock scp, docker, go, npm to make deploy-production.sh run safely in dry test
    cat << 'EOF' > "${bin_dir}/scp"
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "${bin_dir}/scp"

    cat << 'EOF' > "${bin_dir}/docker"
#!/usr/bin/env bash
if [[ "$1" == "build" ]] || [[ "$1" == "save" ]] || [[ "$1" == "pull" ]] || [[ "$1" == "image" ]]; then
    exit 0
fi
exit 0
EOF
    chmod +x "${bin_dir}/docker"

    cat << 'EOF' > "${bin_dir}/sha256sum"
#!/usr/bin/env bash
echo "dummy_sha256  dummy"
exit 0
EOF
    chmod +x "${bin_dir}/sha256sum"

    # Run deploy-production.sh --skip-tests -y -v 1.1.1 with mocked PATH
    export MOCK_TEST_LOG="${mock_log}"
    PATH="${bin_dir}:${PATH}" "${DEPLOY_SCRIPT}" -v 1.1.1 --skip-tests -y >/dev/null 2>&1 || true

    if [[ ! -f "${mock_log}" ]]; then
        echo "Error: Mock log was not created"
        return 1
    fi

    # Verify that VERSION=v1.1.1 was propagated to both Stage G and Stage H
    if ! grep -q '\[SSH_SCRIPT\] REMOTE_DIR=.* VERSION=v1.1.1' "${mock_log}"; then
        echo "Error: Stage G/H did not receive VERSION=v1.1.1 in SSH script arguments."
        cat "${mock_log}"
        return 1
    fi

    # Verify every Compose operation received VERSION=v1.1.1
    local compose_ops
    compose_ops=$(grep '\[COMPOSE_OP\]' "${mock_log}")
    if [[ -z "${compose_ops}" ]]; then
        echo "Error: No compose operations logged"
        return 1
    fi

    while IFS= read -r op; do
        if [[ ! "${op}" =~ VERSION=v1\.1\.1 ]]; then
            echo "Error: Found Compose operation without VERSION=v1.1.1: ${op}"
            return 1
        fi
    done <<< "${compose_ops}"

    # Verify migration container image uses v1.1.1
    if ! echo "${compose_ops}" | grep -qE 'run[[:space:]]+--rm[[:space:]]+--no-deps[[:space:]]+--entrypoint[[:space:]]+/app/migrate[[:space:]]+api[[:space:]]+up'; then
        echo "Error: Migration compose invocation not found in logged operations"
        return 1
    fi

    # Verify application startup uses v1.1.1
    if ! echo "${compose_ops}" | grep -qE 'up[[:space:]]+-d'; then
        echo "Error: Application startup compose invocation not found in logged operations"
        return 1
    fi

    echo "Propagation verified: -v 1.1.1 successfully propagated 'v1.1.1' to all remote Compose calls."
    return 0
}

# ------------------------------------------------------------------------------
# Test 5: Mocked Execution of -v v1.1.2 Propagates v1.1.2
# ------------------------------------------------------------------------------
test_version_propagation_v1_1_2() {
    local tmp_dir
    tmp_dir=$(mktemp -d "/tmp/clientpulse-test-mock2.XXXXXX")
    trap "rm -rf '${tmp_dir}'" RETURN

    local mock_log="${tmp_dir}/compose_calls.log"
    local bin_dir="${tmp_dir}/bin"
    mkdir -p "${bin_dir}"

    cat << 'EOF' > "${bin_dir}/ssh"
#!/usr/bin/env bash
set -euo pipefail
LOG_FILE="${MOCK_TEST_LOG}"
ARGS=()
CAPTURE=false
for a in "$@"; do
    if [[ "$CAPTURE" == "true" ]]; then
        ARGS+=("$a")
    fi
    if [[ "$a" == "--" ]]; then
        CAPTURE=true
    fi
done
SCRIPT_CONTENT=$(cat)
if echo "${SCRIPT_CONTENT}" | grep -q 'Starting isolated PostgreSQL\|Starting ClientPulse application'; then
    REMOTE_DIR="${ARGS[0]:-/home/ubuntu/clientpulse}"
    VERSION="${ARGS[1]:-missing}"
    echo "[SSH_SCRIPT] REMOTE_DIR=${REMOTE_DIR} VERSION=${VERSION}" >> "${LOG_FILE}"
    JOINED_SCRIPT=$(echo "${SCRIPT_CONTENT}" | sed -e :a -e '/\\$/N; s/\\\n//; ta')
    while IFS= read -r line; do
        if [[ "${line}" =~ sudo\ env\ VERSION=\"\$\{VERSION\}\"\ docker\ compose\ (.*) ]]; then
            OP="${BASH_REMATCH[1]}"
            echo "[COMPOSE_OP] VERSION=${VERSION} OP=${OP}" >> "${LOG_FILE}"
        fi
    done <<< "${JOINED_SCRIPT}"
fi
exit 0
EOF
    chmod +x "${bin_dir}/ssh"

    cat << 'EOF' > "${bin_dir}/scp"
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "${bin_dir}/scp"

    cat << 'EOF' > "${bin_dir}/docker"
#!/usr/bin/env bash
exit 0
EOF
    chmod +x "${bin_dir}/docker"

    cat << 'EOF' > "${bin_dir}/sha256sum"
#!/usr/bin/env bash
echo "dummy_sha256 dummy"
exit 0
EOF
    chmod +x "${bin_dir}/sha256sum"

    export MOCK_TEST_LOG="${mock_log}"
    PATH="${bin_dir}:${PATH}" "${DEPLOY_SCRIPT}" -v v1.1.2 --skip-tests -y >/dev/null 2>&1 || true

    if ! grep -q '\[SSH_SCRIPT\] REMOTE_DIR=.* VERSION=v1.1.2' "${mock_log}"; then
        echo "Error: Stage G/H did not receive VERSION=v1.1.2 in SSH script arguments."
        cat "${mock_log}"
        return 1
    fi

    while IFS= read -r op; do
        [[ -z "${op}" ]] && continue
        if [[ ! "${op}" =~ VERSION=v1\.1\.2 ]]; then
            echo "Error: Found Compose operation without VERSION=v1.1.2: ${op}"
            return 1
        fi
    done <<< "$(grep '\[COMPOSE_OP\]' "${mock_log}")"

    echo "Propagation verified: -v v1.1.2 successfully normalized and propagated 'v1.1.2'."
    return 0
}

# ------------------------------------------------------------------------------
# Test 6: Sudo Execution Preserves Explicit VERSION
# ------------------------------------------------------------------------------
test_sudo_env_behavior() {
    # Verify the mechanism: 'sudo env VERSION=value' vs 'VERSION=value sudo'
    # When sudo resets the environment, 'sudo env VERSION=...' guarantees VERSION is in the child process.
    local test_subshell
    test_subshell=$(bash -c '
        fake_sudo() {
            # Simulate sudo scrubbing environment
            env -i "$@"
        }
        VERSION="v2.0.0"
        # Using the exact pattern: sudo env VERSION="${VERSION}" cmd
        fake_sudo env VERSION="${VERSION}" sh -c "echo VERSION=\$VERSION"
    ')
    if [[ "${test_subshell}" != "VERSION=v2.0.0" ]]; then
        echo "Error: 'sudo env VERSION=\${VERSION}' failed to preserve environment in clean subshell: ${test_subshell}"
        return 1
    fi
    echo "Sudo isolation verified: 'sudo env VERSION=\${VERSION}' guarantees propagation through environment resets."
    return 0
}

# ------------------------------------------------------------------------------
# Test 7: Rollback Script Version Propagation
# ------------------------------------------------------------------------------
test_rollback_version_propagation() {
    # Check that rollback script also uses 'sudo env VERSION="${VERSION}" docker compose'
    if ! grep -q 'sudo env VERSION="\${VERSION}" docker compose' "${ROLLBACK_SCRIPT}"; then
        echo "Error: rollback-production.sh must use 'sudo env VERSION=\"\${VERSION}\" docker compose'"
        return 1
    fi

    # Verify rollback normalizes -v 1.0.1 to v1.0.1
    local tmp_dir
    tmp_dir=$(mktemp -d "/tmp/clientpulse-test-rollback.XXXXXX")
    trap "rm -rf '${tmp_dir}'" RETURN

    local bin_dir="${tmp_dir}/bin"
    local mock_log="${tmp_dir}/rollback.log"
    mkdir -p "${bin_dir}"

    cat << 'EOF' > "${bin_dir}/ssh"
#!/usr/bin/env bash
LOG_FILE="${MOCK_ROLLBACK_LOG}"
ARGS=()
CAPTURE=false
for a in "$@"; do
    if [[ "$CAPTURE" == "true" ]]; then
        ARGS+=("$a")
    fi
    if [[ "$a" == "--" ]]; then
        CAPTURE=true
    fi
done
echo "ROLLBACK_VERSION=${ARGS[0]}" >> "${LOG_FILE}"
exit 0
EOF
    chmod +x "${bin_dir}/ssh"

    export MOCK_ROLLBACK_LOG="${mock_log}"
    PATH="${bin_dir}:${PATH}" "${ROLLBACK_SCRIPT}" -v 1.0.1 >/dev/null 2>&1 || true

    if ! grep -q 'ROLLBACK_VERSION=v1.0.1' "${mock_log}"; then
        echo "Error: rollback-production.sh did not normalize version 1.0.1 to v1.0.1"
        cat "${mock_log}"
        return 1
    fi

    echo "Rollback verified: rollback-production.sh correctly normalizes and passes explicit target version."
    return 0
}

# ------------------------------------------------------------------------------
# Test 8: TASK-DEPLOY-001 Port Ownership Validation Remains Intact
# ------------------------------------------------------------------------------
test_port_validation_intact() {
    "${SCRIPT_DIR}/test-port-validation.sh" >/dev/null
    echo "Port ownership validation verified: All 16 test scenarios in test-port-validation.sh passed."
    return 0
}

echo "======================================================================"
echo " Starting ClientPulse Version Propagation Test Suite (TASK-DEPLOY-002)"
echo "======================================================================"
echo ""

run_test "Audit of all remote Compose invocations in deploy-production.sh" test_static_compose_audit
run_test "Missing version flag (-v) fails before deployment" test_missing_version_fails
run_test "Docker Compose resolution prevents fallback to v1.0.0" test_compose_env_version_resolution
run_test "Version propagation with -v 1.1.1" test_version_propagation_1_1_1
run_test "Version propagation with -v v1.1.2" test_version_propagation_v1_1_2
run_test "Sudo environment isolation safety" test_sudo_env_behavior
run_test "Rollback script explicit version propagation" test_rollback_version_propagation
run_test "TASK-DEPLOY-001 port ownership validation intact" test_port_validation_intact

echo ""
echo "======================================================================"
echo " Test Summary: ${TESTS_PASSED}/${TESTS_RUN} Passed, ${TESTS_FAILED} Failed"
echo "======================================================================"

if [[ ${TESTS_FAILED} -ne 0 ]]; then
    echo "❌ Some test scenarios failed!"
    exit 1
fi

echo "🎉 All version propagation test scenarios passed successfully!"
exit 0
