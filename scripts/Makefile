.PHONY: test-unit test-e2e test-all test-race clean help

# ============================================================================
# Test Runner Makefile
#
# Usage from project root:
#   make -C scripts test-all     # Run all tests
#   make -C scripts test-unit    # Run unit tests only
#   make -C scripts test-e2e     # Run E2E tests only
#
# Or use the shell scripts directly:
#   ./scripts/test-all.sh
#   ./scripts/test-unit.sh
#   ./scripts/test-e2e.sh
# ============================================================================

# Default target
all: test-all

# Run unit tests with coverage
test-unit:
	@./test-unit.sh

# Run E2E tests with coverage (requires Docker)
test-e2e:
	@./test-e2e.sh

# Run all tests in parallel with coverage
test-all:
	@./test-all.sh

# Run all tests with race detector
test-race:
	@RACE=1 ./test-all.sh

# Run tests with verbose output
test-verbose:
	@VERBOSE=1 ./test-all.sh

# Run only unit tests with race detector
test-unit-race:
	@RACE=1 ./test-unit.sh

# Run only E2E tests with race detector
test-e2e-race:
	@RACE=1 ./test-e2e.sh

# Run specific E2E test (usage: make test-e2e-run TEST=TestSmoke)
test-e2e-run:
	@./test-e2e.sh $(TEST)

# Clean coverage reports
clean:
	@rm -f ../coverage/*.out ../coverage/*.html ../coverage/*.log
	@echo "Cleaned coverage files"

# Show coverage report in browser (macOS)
coverage-html:
	@if [ -f ../coverage/unit.html ]; then open ../coverage/unit.html; fi
	@if [ -f ../coverage/e2e.html ]; then open ../coverage/e2e.html; fi

# Help
help:
	@echo "Test Runner Commands:"
	@echo ""
	@echo "  make test-all       - Run unit and E2E tests in parallel"
	@echo "  make test-unit      - Run unit tests only"
	@echo "  make test-e2e       - Run E2E tests only (requires Docker)"
	@echo ""
	@echo "  make test-race      - Run all tests with race detector"
	@echo "  make test-verbose   - Run all tests with verbose output"
	@echo ""
	@echo "  make test-e2e-run TEST=TestSmoke  - Run specific E2E test"
	@echo ""
	@echo "  make clean          - Remove coverage files"
	@echo "  make coverage-html  - Open coverage reports in browser"
	@echo ""
	@echo "Environment Variables:"
	@echo ""
	@echo "  VERBOSE=1   - Enable verbose test output"
	@echo "  RACE=1      - Enable race detector"
	@echo "  TIMEOUT=10m - Set test timeout"
	@echo "  SKIP_E2E=1  - Skip E2E tests in test-all"
