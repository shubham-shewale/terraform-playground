#!/bin/bash

# Performance Testing Script
# Tests infrastructure performance under load

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

# Performance test parameters
DURATION=300          # 5 minutes
CONCURRENT_USERS=10   # 10 concurrent users
REQUESTS_PER_USER=50  # 50 requests per user
THINK_TIME=1          # 1 second between requests

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

# Pre-flight checks
preflight_checks() {
    log "Running performance test pre-flight checks..."

    # Check required tools
    local required_tools=("terraform" "aws" "curl" "jq" "bc")
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

    log "Pre-flight checks completed"
}

# Setup performance test environment
setup_performance_test() {
    log "Setting up performance test environment..."

    cd "$PROJECT_DIR"

    # Initialize Terraform
    terraform init -upgrade

    # Create performance test configuration
    cat > performance.tfvars << EOF
environment = "perf-test"
allowed_http_cidrs = ["0.0.0.0/0"]
allowed_ssh_cidrs = ["0.0.0.0/0"]
vpc_cidr = "10.0.0.0/16"
public_subnet_cidr = "10.0.1.0/24"
private_subnet_cidr = "10.0.2.0/24"
instance_type = "t3.medium"  # Larger instance for performance testing
EOF

    # Deploy infrastructure
    terraform plan -var-file=performance.tfvars -out=tfplan
    terraform apply tfplan

    # Get instance details
    PUBLIC_IP=$(terraform output -raw public_instance_public_ip)
    PRIVATE_IP=$(terraform output -raw private_instance_private_ip)
    PUBLIC_INSTANCE_ID=$(terraform output -raw public_instance_id)

    if [ -z "$PUBLIC_IP" ]; then
        error "Failed to get public instance IP"
        exit 1
    fi

    log "Performance test environment setup completed"
    log "Public IP: $PUBLIC_IP"
    log "Private IP: $PRIVATE_IP"
}

# HTTP load test
http_load_test() {
    log "Running HTTP load test..."

    local url="http://$PUBLIC_IP"
    local results_file="/tmp/http_load_test_$(date +%s).txt"

    # Simple load test using curl
    log "Testing with $CONCURRENT_USERS concurrent users for $DURATION seconds..."

    # Create load test script
    cat > /tmp/load_test.sh << EOF
#!/bin/bash
url="$url"
results_file="$results_file"
duration=$DURATION
requests_per_user=$REQUESTS_PER_USER
think_time=$THINK_TIME

start_time=\$(date +%s)
end_time=\$((start_time + duration))

while [ \$(date +%s) -lt \$end_time ]; do
    for i in \$(seq 1 \$requests_per_user); do
        response_time=\$(curl -s -w "%{time_total}" -o /dev/null "\$url")
        echo "\$(date +%s),\$(date +%T),\$response_time" >> "\$results_file"
        sleep \$think_time
    done
done
EOF

    chmod +x /tmp/load_test.sh

    # Run load test in parallel
    for i in $(seq 1 $CONCURRENT_USERS); do
        /tmp/load_test.sh &
    done

    # Wait for all background processes to complete
    wait

    # Analyze results
    analyze_http_results "$results_file"

    log "HTTP load test completed"
}

# Analyze HTTP test results
analyze_http_results() {
    local results_file="$1"

    if [ ! -f "$results_file" ]; then
        error "Results file not found: $results_file"
        return 1
    fi

    log "Analyzing HTTP test results..."

    # Calculate metrics
    local total_requests=$(wc -l < "$results_file")
    local avg_response_time=$(awk -F',' '{sum += $3} END {print sum/NR}' "$results_file")
    local min_response_time=$(awk -F',' 'NR==1{min=$3} {if($3<min)min=$3} END{print min}' "$results_file")
    local max_response_time=$(awk -F',' 'NR==1{max=$3} {if($3>max)max=$3} END{print max}' "$results_file")
    local p95_response_time=$(sort -t',' -k3 -n "$results_file" | awk -F',' 'NR==int(0.95*NR) {print $3}')

    # Calculate requests per second
    local test_duration=$(awk -F',' 'NR==1{first=$1} END{last=$1; print last-first}' "$results_file")
    local rps=$(echo "scale=2; $total_requests / $test_duration" | bc)

    # Display results
    echo "========================================"
    echo "HTTP Load Test Results"
    echo "========================================"
    echo "Total Requests: $total_requests"
    echo "Test Duration: ${test_duration}s"
    echo "Requests/Second: $rps"
    echo "Average Response Time: ${avg_response_time}s"
    echo "Min Response Time: ${min_response_time}s"
    echo "Max Response Time: ${max_response_time}s"
    echo "95th Percentile: ${p95_response_time}s"
    echo "========================================"

    # Performance thresholds
    if (( $(echo "$avg_response_time > 2.0" | bc -l) )); then
        warn "Average response time is high: ${avg_response_time}s"
    fi

    if (( $(echo "$rps < 10" | bc -l) )); then
        warn "Requests per second is low: $rps"
    fi
}

# Network performance test
network_performance_test() {
    log "Running network performance test..."

    # Test latency
    log "Testing network latency..."
    ping -c 10 "$PUBLIC_IP" > /tmp/ping_results.txt

    local avg_latency=$(grep "rtt min/avg/max" /tmp/ping_results.txt | awk -F'/' '{print $5}')
    local packet_loss=$(grep "packet loss" /tmp/ping_results.txt | awk '{print $6}' | tr -d '%')

    echo "========================================"
    echo "Network Performance Results"
    echo "========================================"
    echo "Average Latency: ${avg_latency}ms"
    echo "Packet Loss: ${packet_loss}%"
    echo "========================================"

    # Test bandwidth (simple download test)
    log "Testing network bandwidth..."
    curl -s -w "%{speed_download}" -o /dev/null "http://$PUBLIC_IP/test-file" > /tmp/bandwidth.txt 2>/dev/null || echo "0" > /tmp/bandwidth.txt
    local bandwidth=$(cat /tmp/bandwidth.txt)
    local bandwidth_mbps=$(echo "scale=2; $bandwidth / 1000000" | bc 2>/dev/null || echo "0")

    echo "Download Speed: ${bandwidth_mbps} MB/s"
}

# AWS resource performance monitoring
aws_performance_monitoring() {
    log "Running AWS resource performance monitoring..."

    # Monitor EC2 instance metrics
    log "Monitoring EC2 instance performance..."

    local instance_id="$PUBLIC_INSTANCE_ID"
    local region="${AWS_REGION:-us-east-1}"

    # Get CPU utilization
    local cpu_util=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/EC2 \
        --metric-name CPUUtilization \
        --dimensions Name=InstanceId,Value="$instance_id" \
        --start-time "$(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S)" \
        --end-time "$(date -u +%Y-%m-%dT%H:%M:%S)" \
        --period 300 \
        --statistics Average \
        --region "$region" \
        --query 'Datapoints[0].Average' \
        --output text 2>/dev/null || echo "N/A")

    # Get network traffic
    local network_in=$(aws cloudwatch get-metric-statistics \
        --namespace AWS/EC2 \
        --metric-name NetworkIn \
        --dimensions Name=InstanceId,Value="$instance_id" \
        --start-time "$(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S)" \
        --end-time "$(date -u +%Y-%m-%dT%H:%M:%S)" \
        --period 300 \
        --statistics Sum \
        --region "$region" \
        --query 'Datapoints[0].Sum' \
        --output text 2>/dev/null || echo "N/A")

    echo "========================================"
    echo "AWS Resource Performance"
    echo "========================================"
    echo "CPU Utilization: ${cpu_util}%"
    echo "Network In: ${network_in} bytes"
    echo "========================================"

    # Performance thresholds
    if [[ "$cpu_util" != "N/A" ]] && (( $(echo "$cpu_util > 80" | bc -l 2>/dev/null || echo "0") )); then
        warn "High CPU utilization detected: ${cpu_util}%"
    fi
}

# Generate performance report
generate_performance_report() {
    log "Generating performance test report..."

    local report_dir="$TESTS_DIR/reports"
    mkdir -p "$report_dir"

    local report_file="$report_dir/performance-report-$(date +%Y%m%d-%H%M%S).txt"

    cat > "$report_file" << EOF
Performance Test Report
=======================

Test Date: $(date)
Test Duration: ${DURATION} seconds
Concurrent Users: ${CONCURRENT_USERS}
Requests per User: ${REQUESTS_PER_USER}

Infrastructure Details:
- Public IP: ${PUBLIC_IP}
- Private IP: ${PRIVATE_IP}
- Instance Type: t3.medium
- Region: ${AWS_REGION:-us-east-1}

Test Results Summary:
===================

HTTP Load Test:
- Total Requests: $(wc -l < /tmp/http_load_test_*.txt 2>/dev/null || echo "N/A")
- Average Response Time: $(awk -F',' '{sum += $3} END {if(NR>0) print sum/NR; else print "N/A"}' /tmp/http_load_test_*.txt 2>/dev/null || echo "N/A")
- 95th Percentile: $(sort -t',' -k3 -n /tmp/http_load_test_*.txt 2>/dev/null | awk -F',' 'NR>0 {arr[NR]=$3} END {if(NR>0) print arr[int(NR*0.95)]; else print "N/A"}' || echo "N/A")

Network Performance:
- Average Latency: $(grep "rtt min/avg/max" /tmp/ping_results.txt 2>/dev/null | awk -F'/' '{print $5}' || echo "N/A")
- Packet Loss: $(grep "packet loss" /tmp/ping_results.txt 2>/dev/null | awk '{print $6}' | tr -d '%' || echo "N/A")

Recommendations:
==============

1. Monitor CPU utilization during peak load
2. Consider auto-scaling for high traffic periods
3. Optimize application response times
4. Review network configuration for latency issues
5. Implement caching for improved performance

EOF

    log "Performance report generated: $report_file"
}

# Cleanup performance test environment
cleanup_performance_test() {
    log "Cleaning up performance test environment..."

    cd "$PROJECT_DIR"

    # Destroy infrastructure
    terraform destroy -var-file=performance.tfvars -auto-approve

    # Clean up test files
    rm -f performance.tfvars tfplan
    rm -f /tmp/http_load_test_*.txt
    rm -f /tmp/ping_results.txt
    rm -f /tmp/bandwidth.txt
    rm -f /tmp/load_test.sh

    log "Performance test cleanup completed"
}

# Main execution
main() {
    log "Starting Performance Tests for Basic VPC"

    preflight_checks
    setup_performance_test

    # Run performance tests
    http_load_test
    network_performance_test
    aws_performance_monitoring

    generate_performance_report

    log "🎉 Performance tests completed successfully!"
}

# Trap to ensure cleanup
trap cleanup_performance_test EXIT

# Run main function
main "$@"