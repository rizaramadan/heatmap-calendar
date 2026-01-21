#!/bin/bash
#
# test-e2e.sh - Run E2E tests with coverage
#
# Usage:
#   ./scripts/test-e2e.sh
#   ./scripts/test-e2e.sh TestSmoke    # Run specific test
#
# Environment variables:
#   VERBOSE=1         Enable verbose output
#   TIMEOUT=10m       Test timeout (default: 10m)
#   RACE=1            Enable race detector
#   SKIP_DOCKER_CHECK Skip Docker availability check
#
# Exit codes:
#   0 - All tests passed
#   1 - Tests failed
#   2 - Docker not available (skipped)

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
RACE="${RACE:-0}"
TEST_PATTERN="${1:-}"

# Change to project root
cd "${PROJECT_ROOT}"

# Ensure coverage directory exists
mkdir -p "${COVERAGE_DIR}"

# Print header
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Running E2E Tests${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check Docker availability
if [[ "${SKIP_DOCKER_CHECK:-0}" != "1" ]]; then
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Docker is not installed${NC}"
        echo "E2E tests require Docker for testcontainers."
        echo ""
        echo "Options:"
        echo "  1. Install Docker: https://docs.docker.com/get-docker/"
        echo "  2. Use docker-compose fallback: cd e2e && make test-external-db"
        echo ""
        exit 2
    fi

    if ! docker info &> /dev/null; then
        echo -e "${RED}Docker daemon is not running${NC}"
        echo "Please start Docker and try again."
        echo ""
        exit 2
    fi

    echo -e "${GREEN}Docker is available${NC}"
    echo ""
fi

# Build test arguments
COVERAGE_FILE="${COVERAGE_DIR}/e2e.out"
TEST_ARGS="-tags=e2e -coverprofile=${COVERAGE_FILE} -covermode=atomic -timeout=${TIMEOUT}"

if [[ "${VERBOSE}" == "1" ]]; then
    TEST_ARGS="${TEST_ARGS} -v"
fi

if [[ "${RACE}" == "1" ]]; then
    echo -e "${YELLOW}Race detector enabled${NC}"
    TEST_ARGS="${TEST_ARGS} -race"
fi

if [[ -n "${TEST_PATTERN}" ]]; then
    echo -e "${YELLOW}Running tests matching: ${TEST_PATTERN}${NC}"
    TEST_ARGS="${TEST_ARGS} -run=${TEST_PATTERN}"
fi

echo -e "${YELLOW}Test configuration:${NC}"
echo "  Timeout: ${TIMEOUT}"
echo "  Coverage: ${COVERAGE_FILE}"
echo ""

# Run tests
echo -e "${YELLOW}▶ Running E2E tests...${NC}"
echo -e "${YELLOW}  (This may take a while - starting containers...)${NC}"
echo ""

START_TIME=$(date +%s)

set +e
go test ${TEST_ARGS} ./e2e/tests/...
EXIT_CODE=$?
set -e

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""

# Print results
if [[ "${EXIT_CODE}" == "0" ]]; then
    echo -e "${GREEN}✓ E2E tests passed in ${DURATION}s${NC}"

    # Show coverage summary
    if [[ -f "${COVERAGE_FILE}" ]]; then
        echo ""
        echo -e "${BLUE}Coverage Summary:${NC}"
        go tool cover -func="${COVERAGE_FILE}" | tail -1

        # Generate HTML report
        HTML_FILE="${COVERAGE_DIR}/e2e.html"
        go tool cover -html="${COVERAGE_FILE}" -o "${HTML_FILE}" 2>/dev/null || true
        echo ""
        echo "Coverage report: ${HTML_FILE}"
    fi
else
    echo -e "${RED}✗ E2E tests failed after ${DURATION}s${NC}"
fi

echo ""
exit "${EXIT_CODE}"
