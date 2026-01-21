#!/bin/bash
#
# test-unit.sh - Run unit tests with coverage
#
# Usage:
#   ./scripts/test-unit.sh
#
# Environment variables:
#   VERBOSE=1    Enable verbose output
#   TIMEOUT=5m   Test timeout (default: 5m)
#   RACE=1       Enable race detector
#
# Exit codes:
#   0 - All tests passed
#   1 - Tests failed

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
TIMEOUT="${TIMEOUT:-5m}"
VERBOSE="${VERBOSE:-0}"
RACE="${RACE:-0}"

# Change to project root
cd "${PROJECT_ROOT}"

# Ensure coverage directory exists
mkdir -p "${COVERAGE_DIR}"

# Print header
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Running Unit Tests${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Build test arguments
COVERAGE_FILE="${COVERAGE_DIR}/unit.out"
TEST_ARGS="-coverprofile=${COVERAGE_FILE} -covermode=atomic -timeout=${TIMEOUT}"

if [[ "${VERBOSE}" == "1" ]]; then
    TEST_ARGS="${TEST_ARGS} -v"
fi

if [[ "${RACE}" == "1" ]]; then
    echo -e "${YELLOW}Race detector enabled${NC}"
    TEST_ARGS="${TEST_ARGS} -race"
fi

# Get packages to test (exclude e2e)
PACKAGES=$(go list ./... | grep -v '/e2e')

echo -e "${YELLOW}Testing packages:${NC}"
echo "${PACKAGES}" | head -5
PACKAGE_COUNT=$(echo "${PACKAGES}" | wc -l | tr -d ' ')
if [[ "${PACKAGE_COUNT}" -gt 5 ]]; then
    echo "  ... and $((PACKAGE_COUNT - 5)) more"
fi
echo ""

# Run tests
echo -e "${YELLOW}▶ Running tests...${NC}"
echo ""

START_TIME=$(date +%s)

set +e
go test ${TEST_ARGS} ${PACKAGES}
EXIT_CODE=$?
set -e

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""

# Print results
if [[ "${EXIT_CODE}" == "0" ]]; then
    echo -e "${GREEN}✓ Unit tests passed in ${DURATION}s${NC}"

    # Show coverage summary
    if [[ -f "${COVERAGE_FILE}" ]]; then
        echo ""
        echo -e "${BLUE}Coverage Summary:${NC}"
        go tool cover -func="${COVERAGE_FILE}" | tail -1

        # Generate HTML report
        HTML_FILE="${COVERAGE_DIR}/unit.html"
        go tool cover -html="${COVERAGE_FILE}" -o "${HTML_FILE}" 2>/dev/null || true
        echo ""
        echo "Coverage report: ${HTML_FILE}"
    fi
else
    echo -e "${RED}✗ Unit tests failed after ${DURATION}s${NC}"
fi

echo ""
exit "${EXIT_CODE}"
