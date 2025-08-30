#!/bin/bash

# Build script for CSPM Monitor Lambda functions
# This script creates the necessary ZIP files for Lambda deployment

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to cleanup on error
cleanup() {
    print_error "Build failed! Cleaning up..."
    rm -f lambda-src/*.zip 2>/dev/null || true
    exit 1
}

# Trap errors
trap cleanup ERR

echo "Building CSPM Monitor Lambda functions..."

# Create lambda-src directory if it doesn't exist
if [ ! -d "lambda-src" ]; then
    print_status "Creating lambda-src directory..."
    mkdir -p lambda-src
fi

# Clean up any existing ZIP files
print_status "Cleaning up existing ZIP files..."
rm -f lambda-src/*.zip

# Function to validate Python syntax
validate_python() {
    local file="$1"
    if command -v python3 &> /dev/null; then
        if python3 -m py_compile "$file"; then
            print_status "Python syntax validation passed for $file"
        else
            print_error "Python syntax validation failed for $file"
            return 1
        fi
    else
        print_warning "Python3 not found, skipping syntax validation for $file"
    fi
}

# Function to create ZIP with error handling
create_zip() {
    local zip_name="$1"
    local source_file="$2"

    if [ ! -f "$source_file" ]; then
        print_error "$source_file not found"
        return 1
    fi

    # Validate Python syntax first
    if ! validate_python "$source_file"; then
        return 1
    fi

    if zip "$zip_name" "$source_file"; then
        print_status "Created $zip_name successfully"
        return 0
    else
        print_error "Failed to create $zip_name"
        return 1
    fi
}

# Configuration variables (can be overridden by environment)
LAMBDA_SRC_DIR="${LAMBDA_SRC_DIR:-lambda-src}"
SCANNER_FUNCTION="${SCANNER_FUNCTION:-scanner}"
API_FUNCTION="${API_FUNCTION:-api}"
ARCHIVER_FUNCTION="${ARCHIVER_FUNCTION:-archiver}"

# Create scanner Lambda ZIP
print_status "Creating ${SCANNER_FUNCTION} Lambda ZIP..."
cd "$LAMBDA_SRC_DIR"
if ! create_zip "${SCANNER_FUNCTION}.zip" "${SCANNER_FUNCTION}.py"; then
    print_error "Failed to create ${SCANNER_FUNCTION}.zip"
    exit 1
fi

# Create API Lambda ZIP
print_status "Creating ${API_FUNCTION} Lambda ZIP..."
if ! create_zip "${API_FUNCTION}.zip" "${API_FUNCTION}.py"; then
    print_error "Failed to create ${API_FUNCTION}.zip"
    exit 1
fi

# Create Archiver Lambda ZIP
print_status "Creating ${ARCHIVER_FUNCTION} Lambda ZIP..."
if ! create_zip "${ARCHIVER_FUNCTION}.zip" "${ARCHIVER_FUNCTION}.py"; then
    print_error "Failed to create ${ARCHIVER_FUNCTION}.zip"
    exit 1
fi

cd ..

print_status "Build completed successfully!"
echo "Generated files:"
echo "  - ${LAMBDA_SRC_DIR}/${SCANNER_FUNCTION}.zip"
echo "  - ${LAMBDA_SRC_DIR}/${API_FUNCTION}.zip"
echo "  - ${LAMBDA_SRC_DIR}/${ARCHIVER_FUNCTION}.zip"

# Verify ZIP files with detailed information
print_status "Verifying ZIP files..."
all_files_exist=true
total_size=0

for zip_file in "${SCANNER_FUNCTION}.zip" "${API_FUNCTION}.zip" "${ARCHIVER_FUNCTION}.zip"; do
    if [ -f "${LAMBDA_SRC_DIR}/$zip_file" ]; then
        size=$(stat -f%z "${LAMBDA_SRC_DIR}/$zip_file" 2>/dev/null || stat -c%s "${LAMBDA_SRC_DIR}/$zip_file" 2>/dev/null || echo "0")
        total_size=$((total_size + size))
        print_status "$zip_file: $(numfmt --to=iec-i --suffix=B $size 2>/dev/null || echo "${size} bytes")"
    else
        print_error "$zip_file is missing"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = true ]; then
    print_status "All ZIP files created successfully (Total: $(numfmt --to=iec-i --suffix=B $total_size 2>/dev/null || echo "${total_size} bytes"))"
    echo ""
    print_status "Build summary:"
    ls -la "${LAMBDA_SRC_DIR}"/*.zip
else
    print_error "Some ZIP files are missing"
    exit 1
fi