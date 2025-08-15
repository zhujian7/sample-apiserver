#!/bin/bash

# Test runner script for MyTest API Server

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

print_usage() {
    echo "Usage: $0 [unit|integration|all|coverage]"
    echo ""
    echo "Commands:"
    echo "  unit        - Run unit tests only"
    echo "  integration - Run integration tests only"
    echo "  all         - Run all tests (default)"
    echo "  coverage    - Run tests with coverage report"
}

run_unit_tests() {
    echo "🧪 Running unit tests..."
    go test -v ./pkg/...
    go test -v .
}

run_integration_tests() {
    echo "🔗 Running integration tests..."
    go test -tags=integration -v .
}

run_all_tests() {
    echo "🚀 Running all tests..."
    run_unit_tests
    echo ""
    run_integration_tests
}

run_coverage() {
    echo "📊 Running tests with coverage..."
    
    # Create coverage directory
    mkdir -p coverage
    
    # Run unit tests with coverage
    echo "Unit test coverage..."
    go test -coverprofile=coverage/unit.out -v ./pkg/...
    
    # Run integration tests with coverage
    echo "Integration test coverage..."
    go test -tags=integration -coverprofile=coverage/integration.out -v ./integration_test.go
    
    # Combine coverage reports
    echo "Combining coverage reports..."
    go run github.com/wadey/gocovmerge coverage/unit.out coverage/integration.out > coverage/combined.out
    
    # Generate HTML report
    go tool cover -html=coverage/combined.out -o coverage/report.html
    
    # Display coverage summary
    echo ""
    echo "📈 Coverage Summary:"
    go tool cover -func=coverage/combined.out | tail -1
    
    echo ""
    echo "📄 Full coverage report generated: coverage/report.html"
}

run_benchmarks() {
    echo "⚡ Running benchmarks..."
    go test -bench=. -benchmem ./pkg/...
}

run_race_tests() {
    echo "🏃 Running race condition tests..."
    go test -race -v ./pkg/...
}

# Check if go is installed
if ! command -v go &> /dev/null; then
    echo "❌ ERROR: Go is not installed or not in PATH"
    exit 1
fi

# Main script logic
case "${1:-all}" in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    all)
        run_all_tests
        ;;
    coverage)
        run_coverage
        ;;
    bench|benchmark)
        run_benchmarks
        ;;
    race)
        run_race_tests
        ;;
    *)
        print_usage
        exit 1
        ;;
esac

echo ""
echo "✅ Tests completed successfully!"