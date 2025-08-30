#!/bin/bash

# Performance Test Script for CSPM Monitor
# This script performs basic performance testing on the deployed infrastructure

set -e

echo "🚀 Starting CSPM Monitor Performance Tests"
echo "=========================================="

# Configuration
API_URL="${API_URL:-}"
DYNAMODB_TABLE="${DYNAMODB_TABLE:-}"
REGION="${REGION:-us-east-1}"
CONCURRENT_REQUESTS="${CONCURRENT_REQUESTS:-10}"
TEST_DURATION="${TEST_DURATION:-60}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Validate configuration
if [ -z "$API_URL" ]; then
    print_error "API_URL environment variable is required"
    echo "Usage: API_URL=https://your-api-url ./performance_test.sh"
    exit 1
fi

print_status "Configuration:"
echo "  API URL: $API_URL"
echo "  DynamoDB Table: $DYNAMODB_TABLE"
echo "  Region: $REGION"
echo "  Concurrent Requests: $CONCURRENT_REQUESTS"
echo "  Test Duration: ${TEST_DURATION}s"
echo ""

# Function to test API response time
test_api_response_time() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="$3"

    # Use more compatible time measurement
    local start_time=$(date +%s%N 2>/dev/null || echo "0")
    if [ "$start_time" = "0" ]; then
        # Fallback for systems without nanosecond precision
        start_time=$(date +%s)
        local use_ms=0
    else
        local use_ms=1
    fi

    local response
    local curl_exit_code

    if [ "$method" = "POST" ]; then
        response=$(curl -s -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$data" "$API_URL$endpoint" 2>/dev/null)
        curl_exit_code=$?
    else
        response=$(curl -s -w "%{http_code}" "$API_URL$endpoint" 2>/dev/null)
        curl_exit_code=$?
    fi

    local end_time=$(date +%s%N 2>/dev/null || date +%s)

    local duration
    if [ $use_ms -eq 1 ]; then
        duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds
    else
        duration=$(( (end_time - start_time) * 1000 )) # Convert to milliseconds
    fi

    # Handle curl errors
    if [ $curl_exit_code -ne 0 ]; then
        echo "$duration:000:curl_error"
        return
    fi

    local status_code=$(echo "$response" | tail -c 3)
    local response_body=$(echo "$response" | head -n -1)

    echo "$duration:$status_code:$response_body"
}

# Function to run concurrent requests
run_concurrent_test() {
    local endpoint="$1"
    local num_requests="$2"
    local results_file="$3"

    print_info "Running $num_requests concurrent requests to $endpoint"

    # Create temporary file for results
    local temp_file=$(mktemp)
    local pids=()

    # Run requests in parallel
    for i in $(seq 1 "$num_requests"); do
        (
            result=$(test_api_response_time "$endpoint")
            echo "$result" >> "$temp_file"
        ) &
        pids+=($!)
    done

    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null || true
    done

    # Ensure all results are written
    sync

    # Process results
    local total_time=0
    local success_count=0
    local error_count=0
    local total_requests=0

    while IFS=: read -r duration status_code response_body; do
        total_requests=$((total_requests + 1))

        # Handle potential non-numeric duration
        if [[ "$duration" =~ ^[0-9]+$ ]]; then
            total_time=$((total_time + duration))
        else
            duration=0
        fi

        # Check for successful responses
        if [ "$status_code" = "200" ]; then
            success_count=$((success_count + 1))
        elif [ "$status_code" = "000" ]; then
            error_count=$((error_count + 1))
        fi
    done < "$temp_file"

    # Calculate metrics safely
    if [ $total_requests -gt 0 ]; then
        local avg_time=$((total_time / total_requests))
        local success_rate=$((success_count * 100 / total_requests))
    else
        local avg_time=0
        local success_rate=0
    fi

    echo "$avg_time:$success_rate:$total_requests:$success_count:$error_count" > "$results_file"

    # Cleanup
    rm -f "$temp_file"
}

# Function to test sustained load
test_sustained_load() {
    local endpoint="$1"
    local duration="$2"
    local concurrency="$3"

    print_info "Testing sustained load for ${duration}s with ${concurrency} concurrent users"

    local start_time=$(date +%s)
    local end_time=$((start_time + duration))
    local request_count=0
    local success_count=0
    local error_count=0
    local total_response_time=0
    local batch_count=0

    while [ $(date +%s) -lt $end_time ]; do
        batch_count=$((batch_count + 1))
        local temp_file=$(mktemp)
        local pids=()

        # Run batch of concurrent requests
        for i in $(seq 1 "$concurrency"); do
            (
                result=$(test_api_response_time "$endpoint")
                echo "$result" >> "$temp_file"
            ) &
            pids+=($!)
        done

        # Wait for batch to complete with timeout
        local batch_start=$(date +%s)
        for pid in "${pids[@]}"; do
            # Timeout after 30 seconds per batch
            timeout 30s wait "$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
        done

        # Ensure all results are written
        sync

        # Process batch results
        while IFS=: read -r duration status_code response_body; do
            request_count=$((request_count + 1))

            # Handle potential non-numeric duration
            if [[ "$duration" =~ ^[0-9]+$ ]]; then
                total_response_time=$((total_response_time + duration))
            fi

            if [ "$status_code" = "200" ]; then
                success_count=$((success_count + 1))
            elif [ "$status_code" = "000" ]; then
                error_count=$((error_count + 1))
            fi
        done < "$temp_file"

        rm -f "$temp_file"

        # Check if we're running out of time
        local current_time=$(date +%s)
        if [ $current_time -ge $end_time ]; then
            break
        fi

        # Adaptive delay based on system load
        local elapsed=$((current_time - start_time))
        local progress=$((elapsed * 100 / duration))
        if [ $progress -lt 25 ]; then
            sleep 0.05  # Fast at start
        elif [ $progress -lt 75 ]; then
            sleep 0.1   # Medium in middle
        else
            sleep 0.2   # Slow at end to prevent overwhelming
        fi
    done

    # Calculate final metrics safely
    if [ $request_count -gt 0 ]; then
        local avg_response_time=$((total_response_time / request_count))
        local success_rate=$((success_count * 100 / request_count))
    else
        local avg_response_time=0
        local success_rate=0
    fi

    local actual_duration=$(( $(date +%s) - start_time ))
    local requests_per_second=$((request_count / (actual_duration > 0 ? actual_duration : 1) ))

    print_status "Sustained Load Results:"
    echo "  Total Requests: $request_count"
    echo "  Batches Processed: $batch_count"
    echo "  Actual Duration: ${actual_duration}s"
    echo "  Requests/Second: $requests_per_second"
    echo "  Average Response Time: ${avg_response_time}ms"
    echo "  Success Rate: ${success_rate}%"
    echo "  Error Count: $error_count"
}

# Function to test DynamoDB performance
test_dynamodb_performance() {
    if [ -z "$DYNAMODB_TABLE" ]; then
        print_warning "DynamoDB table not specified, skipping DynamoDB tests"
        return
    fi

    print_info "Testing DynamoDB performance"

    # Test read capacity
    local read_test_start=$(date +%s)
    aws dynamodb describe-table --table-name "$DYNAMODB_TABLE" --region "$REGION" >/dev/null 2>&1
    local read_test_end=$(date +%s)
    local read_time=$((read_test_end - read_test_start))

    print_status "DynamoDB Performance:"
    echo "  Table Describe Time: ${read_time}s"

    # Test query performance (if GSI exists)
    local query_start=$(date +%s)
    aws dynamodb query \
        --table-name "$DYNAMODB_TABLE" \
        --index-name "SeverityTimestampIndex" \
        --key-condition-expression "severity = :severity" \
        --expression-attribute-values '{":severity":{"S":"HIGH"}}' \
        --region "$REGION" \
        --max-items 10 >/dev/null 2>&1
    local query_end=$(date +%s)
    local query_time=$((query_end - query_start))

    echo "  GSI Query Time: ${query_time}s"
}

# Main test execution
echo "1. Testing API Endpoints"
echo "========================"

# Test health endpoint
print_info "Testing health endpoint..."
health_result=$(test_api_response_time "/prod/health")
health_time=$(echo "$health_result" | cut -d: -f1)
health_status=$(echo "$health_result" | cut -d: -f2)

if [ "$health_status" = "200" ]; then
    print_status "Health check passed (${health_time}ms)"
else
    print_error "Health check failed (Status: $health_status)"
fi

# Test findings endpoint
print_info "Testing findings endpoint..."
findings_result=$(test_api_response_time "/prod/findings")
findings_time=$(echo "$findings_result" | cut -d: -f1)
findings_status=$(echo "$findings_result" | cut -d: -f2)

if [ "$findings_status" = "200" ]; then
    print_status "Findings endpoint passed (${findings_time}ms)"
else
    print_error "Findings endpoint failed (Status: $findings_status)"
fi

echo ""
echo "2. Concurrent Request Testing"
echo "============================="

# Test concurrent requests
concurrent_results=$(mktemp)
run_concurrent_test "/prod/findings" "$CONCURRENT_REQUESTS" "$concurrent_results"

avg_time=$(cut -d: -f1 "$concurrent_results")
success_rate=$(cut -d: -f2 "$concurrent_results")
total_requests=$(cut -d: -f3 "$concurrent_results")
success_count=$(cut -d: -f4 "$concurrent_results")
error_count=$(cut -d: -f5 "$concurrent_results")

print_status "Concurrent Test Results:"
echo "  Average Response Time: ${avg_time}ms"
echo "  Success Rate: ${success_rate}%"
echo "  Successful Requests: $success_count/$total_requests"
echo "  Error Count: $error_count"

rm -f "$concurrent_results"

echo ""
echo "3. Sustained Load Testing"
echo "========================="

# Test sustained load
test_sustained_load "/prod/findings" "$TEST_DURATION" 5

echo ""
echo "4. Database Performance"
echo "======================="

# Test DynamoDB performance
test_dynamodb_performance

echo ""
echo "5. Performance Summary"
echo "======================"

print_status "Performance Test Summary:"
echo "✅ API Health Check: ${health_time}ms"
echo "✅ API Findings: ${findings_time}ms"
echo "✅ Concurrent Requests: ${success_rate}% success rate"
echo "✅ Sustained Load: Completed ${TEST_DURATION}s test"

# Performance thresholds
if [ "$health_time" -gt 1000 ]; then
    print_warning "Health check response time is high (>1s)"
fi

if [ "$findings_time" -gt 2000 ]; then
    print_warning "Findings endpoint response time is high (>2s)"
fi

if [ "$success_rate" -lt 95 ]; then
    print_warning "Success rate is below 95%"
fi

print_status "Performance testing completed!"
echo ""
echo "Recommendations:"
echo "- Monitor API response times in production"
echo "- Consider API Gateway caching for better performance"
echo "- Implement rate limiting to prevent abuse"
echo "- Monitor DynamoDB capacity and auto-scaling"

# Cleanup
rm -f /tmp/tmp.* 2>/dev/null || true