#!/bin/bash
#
# test-all.sh - Run unit tests and E2E tests in parallel with coverage
#
# Usage:
#   ./scripts/test-all.sh
#
# Environment variables:
#   SKIP_E2E=1        Skip E2E tests (useful when Docker is not available)
#   VERBOSE=1         Enable verbose output
#   TIMEOUT=10m       Test timeout (default: 10m)
#
# Exit codes:
#   0 - All tests passed
#   1 - One or more tests failed
#   2 - Setup error

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
TIMEOUT="${TIMEOUT:-10m}"
VERBOSE="${VERBOSE:-0}"

# Change to project root
cd "${PROJECT_ROOT}"

# Ensure coverage directory exists
mkdir -p "${COVERAGE_DIR}"

# Print header
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Running All Tests (Unit + E2E)${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check if Docker is available for E2E tests
DOCKER_AVAILABLE=0
if command -v docker &> /dev/null && docker info &> /dev/null; then
    DOCKER_AVAILABLE=1
fi

# Track results
UNIT_PID=""
E2E_PID=""
UNIT_EXIT=0
E2E_EXIT=0
UNIT_SKIPPED=0
E2E_SKIPPED=0

# Cleanup function
cleanup() {
    # Kill any background processes
    if [[ -n "${UNIT_PID}" ]] && kill -0 "${UNIT_PID}" 2>/dev/null; then
        kill "${UNIT_PID}" 2>/dev/null || true
    fi
    if [[ -n "${E2E_PID}" ]] && kill -0 "${E2E_PID}" 2>/dev/null; then
        kill "${E2E_PID}" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# ============================================================================
# Run Unit Tests
# ============================================================================
echo -e "${YELLOW}▶ Starting unit tests...${NC}"

UNIT_OUTPUT="${COVERAGE_DIR}/unit.log"
UNIT_COVERAGE="${COVERAGE_DIR}/unit.out"

# Build test args
UNIT_ARGS="-coverprofile=${UNIT_COVERAGE} -covermode=atomic -timeout=${TIMEOUT}"
if [[ "${VERBOSE}" == "1" ]]; then
    UNIT_ARGS="${UNIT_ARGS} -v"
fi

# Run unit tests in background (exclude e2e directory)
(
    go test ${UNIT_ARGS} $(go list ./... | grep -v '/e2e') > "${UNIT_OUTPUT}" 2>&1
) &
UNIT_PID=$!

# ============================================================================
# Run E2E Tests
# ============================================================================
E2E_OUTPUT="${COVERAGE_DIR}/e2e.log"
E2E_COVERAGE="${COVERAGE_DIR}/e2e.out"

if [[ "${SKIP_E2E:-0}" == "1" ]]; then
    echo -e "${YELLOW}▶ Skipping E2E tests (SKIP_E2E=1)${NC}"
    E2E_SKIPPED=1
elif [[ "${DOCKER_AVAILABLE}" == "0" ]]; then
    echo -e "${YELLOW}▶ Skipping E2E tests (Docker not available)${NC}"
    E2E_SKIPPED=1
else
    echo -e "${YELLOW}▶ Starting E2E tests...${NC}"

    # Build test args
    E2E_ARGS="-tags=e2e -coverprofile=${E2E_COVERAGE} -covermode=atomic -timeout=${TIMEOUT}"
    if [[ "${VERBOSE}" == "1" ]]; then
        E2E_ARGS="${E2E_ARGS} -v"
    fi

    # Run E2E tests in background
    (
        go test ${E2E_ARGS} ./e2e/tests/... > "${E2E_OUTPUT}" 2>&1
    ) &
    E2E_PID=$!
fi

# ============================================================================
# Wait for Tests to Complete
# ============================================================================
echo ""
echo -e "${BLUE}Waiting for tests to complete...${NC}"
echo ""

# Wait for unit tests
if wait "${UNIT_PID}"; then
    UNIT_EXIT=0
else
    UNIT_EXIT=1
fi

# Wait for E2E tests (if running)
if [[ "${E2E_SKIPPED}" == "0" ]] && [[ -n "${E2E_PID}" ]]; then
    if wait "${E2E_PID}"; then
        E2E_EXIT=0
    else
        E2E_EXIT=1
    fi
fi

# ============================================================================
# Print Results Summary
# ============================================================================
echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Test Results Summary${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Unit test results
if [[ "${UNIT_EXIT}" == "0" ]]; then
    echo -e "${GREEN}✓ Unit Tests: PASSED${NC}"
    if [[ -f "${UNIT_COVERAGE}" ]]; then
        UNIT_COV=$(go tool cover -func="${UNIT_COVERAGE}" 2>/dev/null | grep total | awk '{print $3}' || echo "N/A")
        echo -e "  Coverage: ${UNIT_COV}"
    fi
else
    echo -e "${RED}✗ Unit Tests: FAILED${NC}"
    echo -e "  See ${UNIT_OUTPUT} for details"
fi

# E2E test results
if [[ "${E2E_SKIPPED}" == "1" ]]; then
    echo -e "${YELLOW}○ E2E Tests: SKIPPED${NC}"
elif [[ "${E2E_EXIT}" == "0" ]]; then
    echo -e "${GREEN}✓ E2E Tests: PASSED${NC}"
    if [[ -f "${E2E_COVERAGE}" ]]; then
        E2E_COV=$(go tool cover -func="${E2E_COVERAGE}" 2>/dev/null | grep total | awk '{print $3}' || echo "N/A")
        echo -e "  Coverage: ${E2E_COV}"
    fi
else
    echo -e "${RED}✗ E2E Tests: FAILED${NC}"
    echo -e "  See ${E2E_OUTPUT} for details"
fi

echo ""

# Show failure details if any test failed
if [[ "${UNIT_EXIT}" != "0" ]]; then
    echo -e "${RED}--- Unit Test Failures ---${NC}"
    tail -50 "${UNIT_OUTPUT}"
    echo ""
fi

if [[ "${E2E_SKIPPED}" == "0" ]] && [[ "${E2E_EXIT}" != "0" ]]; then
    echo -e "${RED}--- E2E Test Failures ---${NC}"
    tail -50 "${E2E_OUTPUT}"
    echo ""
fi

# Coverage files location
echo -e "${BLUE}Coverage files:${NC}"
echo "  Unit: ${UNIT_COVERAGE}"
if [[ "${E2E_SKIPPED}" == "0" ]]; then
    echo "  E2E:  ${E2E_COVERAGE}"
fi
echo ""

# Generate HTML coverage report if tests passed
if [[ "${UNIT_EXIT}" == "0" ]] && [[ -f "${UNIT_COVERAGE}" ]]; then
    go tool cover -html="${UNIT_COVERAGE}" -o "${COVERAGE_DIR}/unit.html" 2>/dev/null || true
    echo "  HTML: ${COVERAGE_DIR}/unit.html"
fi

if [[ "${E2E_SKIPPED}" == "0" ]] && [[ "${E2E_EXIT}" == "0" ]] && [[ -f "${E2E_COVERAGE}" ]]; then
    go tool cover -html="${E2E_COVERAGE}" -o "${COVERAGE_DIR}/e2e.html" 2>/dev/null || true
    echo "  HTML: ${COVERAGE_DIR}/e2e.html"
fi

echo ""

# Final exit code
if [[ "${UNIT_EXIT}" != "0" ]] || [[ "${E2E_SKIPPED}" == "0" && "${E2E_EXIT}" != "0" ]]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
