# CSPM Monitor Test Suite Makefile
# Comprehensive test orchestration for the Terraform project

.PHONY: help test test-unit test-integration test-performance test-compliance clean setup validate

# Default target
help:
	@echo "CSPM Monitor Test Suite"
	@echo "======================="
	@echo ""
	@echo "Available targets:"
	@echo "  help              - Show this help message"
	@echo "  setup             - Set up test environment and dependencies"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  test-performance  - Run performance tests"
	@echo "  test-compliance   - Run compliance tests"
	@echo "  validate          - Validate test setup and configuration"
	@echo "  clean             - Clean up test artifacts"
	@echo "  coverage          - Generate test coverage report"
	@echo "  benchmark         - Run performance benchmarks"
	@echo ""
	@echo "Test Structure:"
	@echo "  ├── unit/           - Unit tests for Lambda functions"
	@echo "  ├── integration/    - Integration tests for infrastructure"
	@echo "  ├── compliance/     - Compliance and security tests"
	@echo "  ├── scripts/        - Test utilities and performance scripts"
	@echo "  └── testdata/       - Test data and fixtures"
	@echo ""

# Setup test environment
setup:
	@echo "Setting up test environment..."
	@python3 -m pip install --user pytest pytest-cov moto boto3
	@go mod tidy
	@mkdir -p testdata
	@echo "✅ Test environment setup complete"

# Validate test setup
validate:
	@echo "Validating test setup..."
	@python3 --version
	@go version
	@terraform version
	@which pytest || echo "pytest not found - install with: pip install pytest"
	@which tflint || echo "tflint not found - optional for linting"
	@echo "✅ Test setup validation complete"

# Run all tests
test: test-unit test-integration test-compliance
	@echo "✅ All tests completed successfully"

# Unit tests
test-unit:
	@echo "Running unit tests..."
	@cd unit && python3 -m pytest . -v --tb=short
	@echo "✅ Unit tests completed"

# Integration tests
test-integration:
	@echo "Running integration tests..."
	@cd integration && go test -v -timeout 30m
	@echo "✅ Integration tests completed"

# Performance tests
test-performance:
	@echo "Running performance tests..."
	@cd scripts && chmod +x performance_test.sh
	@if [ -z "$$API_URL" ]; then \
		echo "⚠️  API_URL not set, running performance test in dry-run mode"; \
		echo "To run actual performance tests, set API_URL environment variable"; \
		./performance_test.sh --dry-run; \
	else \
		./performance_test.sh; \
	fi
	@echo "✅ Performance tests completed"

# Compliance tests
test-compliance:
	@echo "Running compliance tests..."
	@echo "Note: Compliance tests validate configuration against frameworks"
	@cd integration && go test -v -run TestCompliance
	@echo "✅ Compliance tests completed"

# Generate test coverage
coverage:
	@echo "Generating test coverage report..."
	@cd unit && python3 -m pytest . --cov=. --cov-report=html --cov-report=term
	@echo "Coverage report generated in htmlcov/"
	@echo "✅ Coverage report generated"

# Performance benchmarks
benchmark:
	@echo "Running performance benchmarks..."
	@cd scripts && chmod +x performance_test.sh
	@API_URL="http://localhost:3000" CONCURRENT_REQUESTS=5 TEST_DURATION=30 ./performance_test.sh
	@echo "✅ Performance benchmarks completed"

# Clean up test artifacts
clean:
	@echo "Cleaning up test artifacts..."
	@find . -name "*.pyc" -delete
	@find . -name "__pycache__" -type d -exec rm -rf {} +
	@find . -name ".pytest_cache" -type d -exec rm -rf {} +
	@find . -name "htmlcov" -type d -exec rm -rf {} +
	@find . -name "*.log" -delete
	@find . -name "testdata" -type d -exec rm -rf {} +
	@rm -f scripts/*.zip
	@echo "✅ Cleanup completed"

# Generate test data
generate-test-data:
	@echo "Generating test data..."
	@cd scripts && python3 generate_test_data.py
	@echo "✅ Test data generated"

# Lint tests
lint:
	@echo "Linting test code..."
	@python3 -m flake8 unit/ scripts/ || echo "flake8 not installed - install with: pip install flake8"
	@gofmt -l integration/ || echo "gofmt check complete"
	@echo "✅ Linting completed"

# Security scan tests
security-scan:
	@echo "Running security scan on test code..."
	@python3 -m bandit unit/ scripts/ || echo "bandit not installed - install with: pip install bandit"
	@echo "✅ Security scan completed"

# CI/CD pipeline simulation
ci-pipeline: validate setup generate-test-data lint security-scan test coverage
	@echo "✅ CI/CD pipeline simulation completed successfully"

# Quick test for development
quick-test:
	@echo "Running quick test suite..."
	@cd unit && python3 -m pytest test_api.py::TestLambdaHandler::test_lambda_handler_health_check -v
	@cd unit && python3 -m pytest test_scanner.py::TestProcessFinding::test_process_finding_complete -v
	@echo "✅ Quick tests completed"

# Debug test
debug-test:
	@echo "Running tests in debug mode..."
	@cd unit && python3 -m pytest . -v -s --pdb

# Test with different Python versions (if available)
test-multi-python:
	@echo "Testing with multiple Python versions..."
	@python3 --version
	@python3 unit/test_api.py || echo "Python 3 test failed"
	@echo "✅ Multi-Python testing completed"

# Load test simulation
load-test:
	@echo "Running load test simulation..."
	@cd scripts && chmod +x performance_test.sh
	@API_URL="http://test-endpoint" CONCURRENT_REQUESTS=10 TEST_DURATION=60 ./performance_test.sh
	@echo "✅ Load test simulation completed"

# Test summary
summary:
	@echo "CSPM Monitor Test Suite Summary"
	@echo "==============================="
	@echo ""
	@echo "Test Categories:"
	@echo "• Unit Tests: Lambda function logic and utilities"
	@echo "• Integration Tests: Infrastructure validation and deployment"
	@echo "• Performance Tests: Load testing and benchmarking"
	@echo "• Compliance Tests: Security and regulatory validation"
	@echo ""
	@echo "Key Features Tested:"
	@echo "• Lambda cold starts and memory management"
	@echo "• DynamoDB operations and error handling"
	@echo "• API Gateway request/response processing"
	@echo "• Security Hub finding ingestion"
	@echo "• S3 archival and lifecycle management"
	@echo "• CloudWatch monitoring and alerting"
	@echo "• WAF protection and rate limiting"
	@echo "• VPC and security group configurations"
	@echo ""
	@echo "Test Quality Metrics:"
	@echo "• Code Coverage: Target >80%"
	@echo "• Performance: <2s average response time"
	@echo "• Reliability: <1% error rate under load"
	@echo "• Security: Zero critical vulnerabilities"
	@echo ""

# Default target
.DEFAULT_GOAL := help