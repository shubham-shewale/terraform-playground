#!/bin/bash

# Test Runner Script
# Executes all test types in the correct order

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_DIR="$(dirname "$TESTS_DIR")"

# Default values
TEST_TYPE="all"
VERBOSE=false
CLEANUP=true
TIMEOUT="60m"
PARALLEL=true

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Help function
show_help() {
    cat << EOF
Test Runner Script for Basic VPC

USAGE:
    $0 [OPTIONS] [TEST_TYPE]

TEST_TYPES:
    unit         Run unit tests only
    integration  Run integration tests only
    e2e          Run end-to-end tests only
    compliance   Run compliance tests only
    all          Run all tests (default)

OPTIONS:
    -h, --help           Show this help message
    -v, --verbose        Enable verbose output
    --no-cleanup         Skip cleanup after tests
    --timeout DURATION   Set test timeout (default: 60m)
    --no-parallel        Disable parallel test execution
    --dry-run           Show what would be executed without running

EXAMPLES:
    $0 unit                    # Run unit tests
    $0 integration -v          # Run integration tests with verbose output
    $0 all --timeout 90m       # Run all tests with 90 minute timeout
    $0 e2e --no-cleanup        # Run E2E tests without cleanup

ENVIRONMENT VARIABLES:
    AWS_REGION          AWS region for tests
    AWS_PROFILE         AWS profile to use
    TEST_ENVIRONMENT    Test environment name
    TF_LOG              Terraform log level

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --no-cleanup)
                CLEANUP=false
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --no-parallel)
                PARALLEL=false
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            unit|integration|e2e|compliance|all)
                TEST_TYPE="$1"
                shift
                ;;
            *)
                error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Pre-flight checks
preflight_checks() {
    log "Running pre-flight checks..."

    # Check required tools
    local required_tools=("go" "terraform" "aws")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            error "Required tool '$tool' is not installed"
            exit 1
        fi
    done

    # Check AWS credentials
    if ! aws sts get-caller-identity >/dev/null 2>&1; then
        error "AWS credentials are not configured"
        exit 1
    fi

    # Check Go version
    local go_version=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
    if [[ "$(printf '%s\n' "$go_version" "1.21" | sort -V | head -n1)" != "1.21" ]]; then
        error "Go version 1.21 or higher is required"
        exit 1
    fi

    log "Pre-flight checks completed"
}

# Setup test environment
setup_environment() {
    log "Setting up test environment..."

    cd "$TESTS_DIR"

    # Install Go dependencies
    if [ "$DRY_RUN" != "true" ]; then
        go mod download
        go mod tidy
    fi

    # Set environment variables
    export TEST_ENVIRONMENT="${TEST_ENVIRONMENT:-test}"
    export AWS_REGION="${AWS_REGION:-us-east-1}"
    export TF_LOG="${TF_LOG:-info}"

    if [ "$VERBOSE" = "true" ]; then
        export TERRATEST_LOG=debug
    fi

    log "Test environment setup completed"
}

# Run unit tests
run_unit_tests() {
    log "Running unit tests..."

    local cmd="go test -v ./unit/... -timeout $TIMEOUT"
    if [ "$PARALLEL" = "false" ]; then
        cmd="$cmd -p 1"
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo "Would run: $cmd"
        return 0
    fi

    cd "$TESTS_DIR"
    if ! eval "$cmd"; then
        error "Unit tests failed"
        return 1
    fi

    log "Unit tests completed successfully"
}

# Run integration tests
run_integration_tests() {
    log "Running integration tests..."

    local cmd="go test -v ./integration/... -timeout $TIMEOUT"
    if [ "$PARALLEL" = "false" ]; then
        cmd="$cmd -p 1"
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo "Would run: $cmd"
        return 0
    fi

    cd "$TESTS_DIR"
    if ! eval "$cmd"; then
        error "Integration tests failed"
        return 1
    fi

    log "Integration tests completed successfully"
}

# Run end-to-end tests
run_e2e_tests() {
    log "Running end-to-end tests..."

    local script="$TESTS_DIR/e2e/connectivity_test.sh"

    if [ ! -f "$script" ]; then
        error "E2E test script not found: $script"
        return 1
    fi

    if [ "$DRY_RUN" = "true" ]; then
        echo "Would run: $script"
        return 0
    fi

    chmod +x "$script"
    if ! "$script"; then
        error "E2E tests failed"
        return 1
    fi

    log "E2E tests completed successfully"
}

# Run compliance tests
run_compliance_tests() {
    log "Running compliance tests..."

    if ! command -v inspec >/dev/null 2>&1; then
        warn "InSpec not found, skipping compliance tests"
        return 0
    fi

    local cmd="inspec exec $TESTS_DIR/compliance/ --reporter cli"

    if [ "$DRY_RUN" = "true" ]; then
        echo "Would run: $cmd"
        return 0
    fi

    cd "$TESTS_DIR/compliance"
    if ! eval "$cmd"; then
        error "Compliance tests failed"
        return 1
    fi

    log "Compliance tests completed successfully"
}

# Generate test report
generate_report() {
    log "Generating test report..."

    local report_dir="$TESTS_DIR/reports"
    mkdir -p "$report_dir"

    # Generate coverage report
    if [ "$DRY_RUN" != "true" ]; then
        cd "$TESTS_DIR"
        go test -v ./... -coverprofile="$report_dir/coverage.out" >/dev/null 2>&1
        go tool cover -html="$report_dir/coverage.out" -o "$report_dir/coverage.html"
    fi

    # Generate summary
    cat > "$report_dir/test-summary.txt" << EOF
Test Execution Summary
======================

Execution Date: $(date)
Test Type: $TEST_TYPE
Environment: ${TEST_ENVIRONMENT:-test}
AWS Region: ${AWS_REGION:-us-east-1}
Timeout: $TIMEOUT
Parallel: $PARALLEL
Cleanup: $CLEANUP

Test Results:
- Unit Tests: $([ "$TEST_TYPE" = "unit" ] || [ "$TEST_TYPE" = "all" ] && echo "Executed" || echo "Skipped")
- Integration Tests: $([ "$TEST_TYPE" = "integration" ] || [ "$TEST_TYPE" = "all" ] && echo "Executed" || echo "Skipped")
- E2E Tests: $([ "$TEST_TYPE" = "e2e" ] || [ "$TEST_TYPE" = "all" ] && echo "Executed" || echo "Skipped")
- Compliance Tests: $([ "$TEST_TYPE" = "compliance" ] || [ "$TEST_TYPE" = "all" ] && echo "Executed" || echo "Skipped")

Reports Generated:
- Coverage Report: $report_dir/coverage.html
- Test Summary: $report_dir/test-summary.txt
EOF

    log "Test report generated: $report_dir/test-summary.txt"
}

# Cleanup function
cleanup() {
    if [ "$CLEANUP" = "false" ]; then
        warn "Cleanup disabled, manual cleanup may be required"
        return 0
    fi

    log "Cleaning up test environment..."

    # Clean up test files
    cd "$PROJECT_DIR"
    rm -f terraform.tfvars tfplan

    # Clean up Go build cache
    go clean -cache -testcache -modcache

    log "Cleanup completed"
}

# Main execution
main() {
    parse_args "$@"

    log "Starting Basic VPC Test Suite"
    log "Test Type: $TEST_TYPE"
    log "Verbose: $VERBOSE"
    log "Timeout: $TIMEOUT"
    log "Parallel: $PARALLEL"
    log "Cleanup: $CLEANUP"

    preflight_checks
    setup_environment

    local test_failed=0

    # Run tests based on type
    case $TEST_TYPE in
        unit)
            run_unit_tests || test_failed=1
            ;;
        integration)
            run_integration_tests || test_failed=1
            ;;
        e2e)
            run_e2e_tests || test_failed=1
            ;;
        compliance)
            run_compliance_tests || test_failed=1
            ;;
        all)
            run_unit_tests || test_failed=1
            run_integration_tests || test_failed=1
            run_compliance_tests || test_failed=1
            run_e2e_tests || test_failed=1
            ;;
        *)
            error "Invalid test type: $TEST_TYPE"
            exit 1
            ;;
    esac

    generate_report

    if [ $test_failed -eq 0 ]; then
        log "🎉 All tests completed successfully!"
        exit 0
    else
        error "❌ Some tests failed!"
        exit 1
    fi
}

# Trap to ensure cleanup
trap cleanup EXIT

# Run main function
main "$@"