#!/bin/bash
#
# merge-coverage.sh - Merge coverage reports from unit tests, E2E tests, and E2E service
#
# Usage:
#   ./scripts/merge-coverage.sh
#
# Input files (in coverage/):
#   unit.out        - Unit test coverage
#   e2e.out         - E2E test coverage (test code itself)
#   e2e-service/    - E2E service coverage (binary coverage data)
#
# Output files (in coverage/):
#   e2e-service.out - Converted service coverage (text format)
#   combined.out    - Merged coverage profile
#   coverage.html   - HTML coverage report
#
# Exit codes:
#   0 - Success
#   1 - No coverage files found
#   2 - Merge failed

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"

# Change to project root
cd "${PROJECT_ROOT}"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Merging Coverage Reports${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Track available coverage files
COVERAGE_FILES=()
HAS_UNIT=0
HAS_E2E=0
HAS_SERVICE=0

# Check for unit test coverage
if [[ -f "${COVERAGE_DIR}/unit.out" ]]; then
    echo -e "${GREEN}✓ Found unit test coverage${NC}"
    COVERAGE_FILES+=("${COVERAGE_DIR}/unit.out")
    HAS_UNIT=1
else
    echo -e "${YELLOW}○ No unit test coverage (unit.out)${NC}"
fi

# Check for E2E test coverage
if [[ -f "${COVERAGE_DIR}/e2e.out" ]]; then
    echo -e "${GREEN}✓ Found E2E test coverage${NC}"
    COVERAGE_FILES+=("${COVERAGE_DIR}/e2e.out")
    HAS_E2E=1
else
    echo -e "${YELLOW}○ No E2E test coverage (e2e.out)${NC}"
fi

# Check for E2E service coverage (binary format)
SERVICE_COVERAGE_DIR="${COVERAGE_DIR}/e2e-service"
SERVICE_COVERAGE_OUT="${COVERAGE_DIR}/e2e-service.out"

if [[ -d "${SERVICE_COVERAGE_DIR}" ]] && [[ -n "$(ls -A "${SERVICE_COVERAGE_DIR}" 2>/dev/null)" ]]; then
    echo -e "${GREEN}✓ Found E2E service coverage data${NC}"

    # Convert binary coverage data to text format
    echo -e "${YELLOW}  Converting binary coverage data...${NC}"

    if go tool covdata textfmt -i="${SERVICE_COVERAGE_DIR}" -o="${SERVICE_COVERAGE_OUT}" 2>/dev/null; then
        echo -e "${GREEN}  Converted to ${SERVICE_COVERAGE_OUT}${NC}"
        COVERAGE_FILES+=("${SERVICE_COVERAGE_OUT}")
        HAS_SERVICE=1
    else
        echo -e "${RED}  Failed to convert service coverage${NC}"
    fi
else
    echo -e "${YELLOW}○ No E2E service coverage (e2e-service/)${NC}"
fi

echo ""

# Check if we have any coverage files
if [[ ${#COVERAGE_FILES[@]} -eq 0 ]]; then
    echo -e "${RED}No coverage files found!${NC}"
    echo ""
    echo "Run tests first:"
    echo "  ./scripts/test-unit.sh    # Generate unit.out"
    echo "  ./scripts/test-e2e.sh     # Generate e2e.out and e2e-service/"
    exit 1
fi

# Merge coverage files
COMBINED_OUT="${COVERAGE_DIR}/combined.out"
COMBINED_HTML="${COVERAGE_DIR}/coverage.html"

echo -e "${BLUE}Merging ${#COVERAGE_FILES[@]} coverage file(s)...${NC}"

# Method 1: Try gocovmerge if available
if command -v gocovmerge &> /dev/null; then
    echo "Using gocovmerge..."
    gocovmerge "${COVERAGE_FILES[@]}" > "${COMBINED_OUT}"
else
    # Method 2: Manual merge (concatenate with header handling)
    echo "Using manual merge (gocovmerge not installed)..."

    # Start with first file (includes header)
    FIRST_FILE="${COVERAGE_FILES[0]}"
    cp "${FIRST_FILE}" "${COMBINED_OUT}"

    # Append remaining files (skip their headers)
    for ((i=1; i<${#COVERAGE_FILES[@]}; i++)); do
        FILE="${COVERAGE_FILES[$i]}"
        # Skip the "mode:" header line and append the rest
        tail -n +2 "${FILE}" >> "${COMBINED_OUT}"
    done

    echo -e "${YELLOW}Tip: Install gocovmerge for better merging:${NC}"
    echo "  go install github.com/wadey/gocovmerge@latest"
fi

echo ""

# Verify combined file
if [[ ! -f "${COMBINED_OUT}" ]] || [[ ! -s "${COMBINED_OUT}" ]]; then
    echo -e "${RED}Failed to create combined coverage file${NC}"
    exit 2
fi

# Generate HTML report
echo -e "${BLUE}Generating HTML report...${NC}"
go tool cover -html="${COMBINED_OUT}" -o="${COMBINED_HTML}"
echo -e "${GREEN}✓ Created ${COMBINED_HTML}${NC}"
echo ""

# Calculate and display coverage statistics
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}  Coverage Summary${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Show individual coverages
if [[ "${HAS_UNIT}" == "1" ]]; then
    UNIT_COV=$(go tool cover -func="${COVERAGE_DIR}/unit.out" 2>/dev/null | grep "^total:" | awk '{print $3}' || echo "N/A")
    echo -e "Unit tests:    ${UNIT_COV}"
fi

if [[ "${HAS_E2E}" == "1" ]]; then
    E2E_COV=$(go tool cover -func="${COVERAGE_DIR}/e2e.out" 2>/dev/null | grep "^total:" | awk '{print $3}' || echo "N/A")
    echo -e "E2E tests:     ${E2E_COV}"
fi

if [[ "${HAS_SERVICE}" == "1" ]]; then
    SVC_COV=$(go tool cover -func="${SERVICE_COVERAGE_OUT}" 2>/dev/null | grep "^total:" | awk '{print $3}' || echo "N/A")
    echo -e "E2E service:   ${SVC_COV}"
fi

echo ""

# Show combined coverage
TOTAL_COV=$(go tool cover -func="${COMBINED_OUT}" 2>/dev/null | grep "^total:" | awk '{print $3}' || echo "N/A")
echo -e "${GREEN}Combined:      ${TOTAL_COV}${NC}"
echo ""

# Show package breakdown (top 10 lowest coverage)
echo -e "${BLUE}Lowest Coverage Packages:${NC}"
go tool cover -func="${COMBINED_OUT}" 2>/dev/null | \
    grep -v "^total:" | \
    sort -t'%' -k2 -n | \
    head -10 | \
    awk '{printf "  %-50s %s\n", $1, $3}'
echo ""

# Output file locations
echo -e "${BLUE}Output files:${NC}"
echo "  Combined profile: ${COMBINED_OUT}"
echo "  HTML report:      ${COMBINED_HTML}"
echo ""

# Success message with instructions
echo -e "${GREEN}Coverage merge complete!${NC}"
echo ""
echo "View the report:"
echo "  open ${COMBINED_HTML}"
echo ""

exit 0
