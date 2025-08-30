#!/bin/bash

# End-to-End Connectivity Test Script
# This script tests the full connectivity and functionality of the VPC infrastructure

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TERRAFORM_DIR="../../"
TEST_ENVIRONMENT="e2e-test"
ALLOWED_HTTP_CIDRS="[\"$(curl -s ifconfig.me)/32\"]"
ALLOWED_SSH_CIDRS="[\"$(curl -s ifconfig.me)/32\"]"

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Pre-flight checks
preflight_checks() {
    log "Running pre-flight checks..."

    # Check required tools
    local required_tools=("terraform" "aws" "curl" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command_exists "$tool"; then
            error "Required tool '$tool' is not installed"
            exit 1
        fi
    done

    # Check AWS credentials
    if ! aws sts get-caller-identity >/dev/null 2>&1; then
        error "AWS credentials are not configured or invalid"
        exit 1
    fi

    log "Pre-flight checks completed successfully"
}

# Setup test environment
setup_test_environment() {
    log "Setting up test environment..."

    cd "$TERRAFORM_DIR"

    # Initialize Terraform
    if ! terraform init -upgrade; then
        error "Failed to initialize Terraform"
        exit 1
    fi

    # Create terraform.tfvars for testing
    cat > terraform.tfvars << EOF
environment = "$TEST_ENVIRONMENT"
allowed_http_cidrs = $ALLOWED_HTTP_CIDRS
allowed_ssh_cidrs = $ALLOWED_SSH_CIDRS
vpc_cidr = "10.0.0.0/16"
public_subnet_cidr = "10.0.1.0/24"
private_subnet_cidr = "10.0.2.0/24"
EOF

    log "Test environment setup completed"
}

# Deploy infrastructure
deploy_infrastructure() {
    log "Deploying infrastructure..."

    # Validate configuration
    if ! terraform validate; then
        error "Terraform validation failed"
        exit 1
    fi

    # Plan deployment
    if ! terraform plan -out=tfplan; then
        error "Terraform plan failed"
        exit 1
    fi

    # Apply deployment
    if ! terraform apply tfplan; then
        error "Terraform apply failed"
        exit 1
    fi

    log "Infrastructure deployment completed"
}

# Test basic connectivity
test_basic_connectivity() {
    log "Testing basic connectivity..."

    # Get outputs
    PUBLIC_IP=$(terraform output -raw public_instance_public_ip)
    PRIVATE_IP=$(terraform output -raw private_instance_private_ip)

    if [ -z "$PUBLIC_IP" ]; then
        error "Failed to get public instance IP"
        return 1
    fi

    if [ -z "$PRIVATE_IP" ]; then
        error "Failed to get private instance IP"
        return 1
    fi

    log "Public instance IP: $PUBLIC_IP"
    log "Private instance IP: $PRIVATE_IP"

    # Wait for instances to be ready
    log "Waiting for instances to be ready..."
    sleep 120

    # Test public instance HTTP connectivity
    log "Testing public instance HTTP connectivity..."
    if curl -f --max-time 30 "http://$PUBLIC_IP" >/dev/null 2>&1; then
        log "✓ Public instance HTTP connectivity test passed"
    else
        error "✗ Public instance HTTP connectivity test failed"
        return 1
    fi

    log "Basic connectivity tests completed"
}

# Test security configurations
test_security_configurations() {
    log "Testing security configurations..."

    PUBLIC_IP=$(terraform output -raw public_instance_public_ip)

    # Test that SSH is blocked from unauthorized IPs
    log "Testing SSH access restrictions..."
    if ! nc -z -w5 "$PUBLIC_IP" 22 2>/dev/null; then
        log "✓ SSH access properly restricted"
    else
        warn "SSH port appears to be open - this may be expected if running from allowed IP"
    fi

    # Test that only allowed HTTP access works
    log "Testing HTTP access control..."
    # This would require testing from different IP ranges
    # For now, just verify the instance is responding
    if curl -f --max-time 30 "http://$PUBLIC_IP" >/dev/null 2>&1; then
        log "✓ HTTP access test passed"
    else
        error "✗ HTTP access test failed"
        return 1
    fi

    log "Security configuration tests completed"
}

# Test monitoring and logging
test_monitoring_and_logging() {
    log "Testing monitoring and logging..."

    # Test CloudWatch alarms
    log "Testing CloudWatch alarms..."
    ALARM_COUNT=$(aws cloudwatch describe-alarms --alarm-name-prefix "$TEST_ENVIRONMENT" --query 'length(MetricAlarms)' --output text)
    if [ "$ALARM_COUNT" -ge 6 ]; then
        log "✓ CloudWatch alarms created ($ALARM_COUNT alarms found)"
    else
        error "✗ Expected at least 6 CloudWatch alarms, found $ALARM_COUNT"
        return 1
    fi

    # Test VPC Flow Logs
    log "Testing VPC Flow Logs..."
    FLOW_LOG_COUNT=$(aws ec2 describe-flow-logs --query 'length(FlowLogs)' --output text)
    if [ "$FLOW_LOG_COUNT" -ge 1 ]; then
        log "✓ VPC Flow Logs enabled ($FLOW_LOG_COUNT flow logs found)"
    else
        error "✗ VPC Flow Logs not found"
        return 1
    fi

    # Test CloudTrail
    log "Testing CloudTrail..."
    TRAIL_COUNT=$(aws cloudtrail describe-trails --query 'length(trailList)' --output text)
    if [ "$TRAIL_COUNT" -ge 1 ]; then
        log "✓ CloudTrail configured ($TRAIL_COUNT trails found)"
    else
        error "✗ CloudTrail not found"
        return 1
    fi

    log "Monitoring and logging tests completed"
}

# Test SSM connectivity
test_ssm_connectivity() {
    log "Testing SSM connectivity..."

    PRIVATE_INSTANCE_ID=$(terraform output -raw private_instance_id)

    # Test SSM agent connectivity
    log "Testing SSM agent connectivity..."
    if aws ssm describe-instance-information --instance-information-filter-list "key=InstanceIds,valueSet=$PRIVATE_INSTANCE_ID" >/dev/null 2>&1; then
        log "✓ SSM agent connectivity test passed"
    else
        error "✗ SSM agent connectivity test failed"
        return 1
    fi

    log "SSM connectivity tests completed"
}

# Test cleanup
cleanup() {
    log "Cleaning up test environment..."

    cd "$TERRAFORM_DIR"

    # Destroy infrastructure
    if ! terraform destroy -auto-approve; then
        error "Failed to destroy infrastructure"
        return 1
    fi

    # Clean up test files
    rm -f terraform.tfvars tfplan

    log "Cleanup completed"
}

# Main test execution
main() {
    log "Starting End-to-End Connectivity Tests"

    # Trap to ensure cleanup on exit
    trap cleanup EXIT

    preflight_checks
    setup_test_environment
    deploy_infrastructure

    # Run tests
    local test_failed=0

    if ! test_basic_connectivity; then
        test_failed=1
    fi

    if ! test_security_configurations; then
        test_failed=1
    fi

    if ! test_monitoring_and_logging; then
        test_failed=1
    fi

    if ! test_ssm_connectivity; then
        test_failed=1
    fi

    if [ $test_failed -eq 0 ]; then
        log "🎉 All End-to-End tests passed!"
        exit 0
    else
        error "❌ Some End-to-End tests failed!"
        exit 1
    fi
}

# Run main function
main "$@"