#!/bin/bash

# Validation script for CSPM Monitor Terraform configuration
# This script validates the Terraform configuration and checks for common issues

set -e

echo "Validating CSPM Monitor Terraform configuration..."

# Check if terraform is installed
if ! command -v terraform &> /dev/null; then
    echo "❌ Terraform is not installed or not in PATH"
    exit 1
fi

# Check if required files exist
required_files=("main.tf" "variables.tf" "terraform.tf" "backend.tf")
for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Required file $file is missing"
        exit 1
    fi
done

echo "✅ All required files are present"

# Validate Terraform syntax
echo "Running terraform validate..."
if terraform validate; then
    echo "✅ Terraform validation passed"
else
    echo "❌ Terraform validation failed"
    exit 1
fi

# Check for formatting issues
echo "Checking Terraform formatting..."
if terraform fmt -check -recursive .; then
    echo "✅ Terraform formatting is correct"
else
    echo "⚠️  Terraform formatting issues found. Run 'terraform fmt -recursive .' to fix"
fi

# Check for security issues with tfsec (if available)
if command -v tfsec &> /dev/null; then
    echo "Running tfsec security scan..."
    if tfsec . --exclude-rule AWS095; then
        echo "✅ Security scan passed"
    else
        echo "⚠️  Security issues found. Review tfsec output above"
    fi
else
    echo "ℹ️  tfsec not found, skipping security scan"
fi

# Check for linting issues with tflint (if available)
if command -v tflint &> /dev/null; then
    echo "Running tflint..."
    if tflint --config .tflint.hcl .; then
        echo "✅ TFLint passed"
    else
        echo "⚠️  TFLint issues found. Review output above"
    fi
else
    echo "ℹ️  tflint not found, skipping linting"
fi

# Check Lambda source files
echo "Checking Lambda source files..."
lambda_files=("lambda-src/api.py" "lambda-src/scanner.py" "lambda-src/archiver.py")
for file in "${lambda_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Lambda source file $file is missing"
        exit 1
    fi
done
echo "✅ Lambda source files are present"

# Check website files
echo "Checking website files..."
website_files=("website/index.html" "website/app.js" "website/style.css")
for file in "${website_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Website file $file is missing"
        exit 1
    fi
done
echo "✅ Website files are present"

# Check for common configuration issues
echo "Checking for common configuration issues..."

# Check if backend bucket is configured
if grep -q "your-terraform-state-bucket" backend.tf; then
    echo "⚠️  Backend bucket is not configured. Update backend.tf with your S3 bucket"
fi

# Check for hardcoded values
if grep -r "your-api-gateway-url" website/ 2>/dev/null; then
    echo "⚠️  Found placeholder API URL in website files"
fi

# Run test suite if available
if [ -d "tests" ] && [ -f "tests/Makefile" ]; then
    echo "Running test suite..."
    cd tests
    if make smoke-test; then
        echo "✅ Smoke tests passed"
    else
        echo "⚠️  Some smoke tests failed - check test configuration"
    fi
    cd ..
else
    echo "ℹ️  Test suite not found - run tests manually after setup"
fi

echo ""
echo "Validation completed!"
echo ""
echo "Next steps:"
echo "1. Update backend.tf with your S3 bucket for Terraform state"
echo "2. Run './build.sh' to create Lambda ZIP files"
echo "3. Run 'terraform plan' to review the deployment"
echo "4. Run 'terraform apply' to deploy the infrastructure"
echo "5. Run tests: cd tests && make test"
echo ""
echo "Test Suite Available:"
echo "- Unit tests: pytest tests/unit/"
echo "- Integration tests: go test ./tests/integration/"
echo "- E2E tests: go test ./tests/e2e/"
echo "- Compliance tests: go test ./tests/compliance/"
echo "- Performance tests: ./tests/scripts/performance_test.sh"
echo "5. Run 'make test' in tests/ directory to execute full test suite"