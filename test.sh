#!/bin/bash

# Test script for tofusort
set -e

echo "Running tofusort tests..."
echo "========================="

# Build the tool
echo "Building tofusort..."
mise exec -- go build -o tofusort ./cmd/tofusort

# Function to run a single test
run_test() {
    local input_file=$1
    local expected_file=$2
    local test_name=$(basename "$input_file" .tf)
    
    echo "Testing: $test_name"
    
    # Copy input to a temp file
    local temp_file="/tmp/test_${test_name}.tf"
    cp "$input_file" "$temp_file"
    
    # Run tofusort on the temp file using mise
    mise exec -- ./tofusort sort "$temp_file" > /dev/null 2>&1
    
    # Compare with expected output
    if diff -u "$expected_file" "$temp_file" > "/tmp/diff_${test_name}.txt"; then
        echo "✅ $test_name: PASSED"
        rm "/tmp/diff_${test_name}.txt"
    else
        echo "❌ $test_name: FAILED"
        echo "Differences:"
        cat "/tmp/diff_${test_name}.txt"
        echo "---"
        
        # Show actual output for debugging
        echo "Actual output:"
        cat "$temp_file"
        echo "---"
    fi
    
    # Clean up
    rm "$temp_file"
}

# Run all tests
for input_file in tests/input/*.tf; do
    test_name=$(basename "$input_file")
    expected_file="tests/expected/$test_name"
    
    if [[ -f "$expected_file" ]]; then
        run_test "$input_file" "$expected_file"
    else
        echo "⚠️  No expected output for $test_name"
    fi
done

echo "========================="
echo "Tests completed!"
