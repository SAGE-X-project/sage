#!/bin/bash
# SAGE Benchmark Runner
# Runs comprehensive benchmarks and generates reports

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
BENCHMARK_DIR="tools/benchmark"
OUTPUT_DIR="tools/benchmark/results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="$OUTPUT_DIR/benchmark_$TIMESTAMP.txt"
JSON_FILE="$OUTPUT_DIR/benchmark_$TIMESTAMP.json"
ANALYSIS_FILE="$OUTPUT_DIR/analysis_$TIMESTAMP.md"

echo -e "${GREEN}SAGE Benchmark Suite${NC}"
echo "================================"
echo "Timestamp: $TIMESTAMP"
echo "Output: $OUTPUT_DIR"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Parse arguments
RUN_TYPE="all"
COMPARE_WITH=""
BENCH_TIME="10s"
COUNT=5

while [[ $# -gt 0 ]]; do
    case $1 in
        --type)
            RUN_TYPE="$2"
            shift 2
            ;;
        --compare)
            COMPARE_WITH="$2"
            shift 2
            ;;
        --benchtime)
            BENCH_TIME="$2"
            shift 2
            ;;
        --count)
            COUNT="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --type TYPE         Benchmark type: all, crypto, session, rfc9421, comparison (default: all)"
            echo "  --compare FILE      Compare with previous results"
            echo "  --benchtime TIME    Benchmark time per test (default: 10s)"
            echo "  --count N           Number of times to run each benchmark (default: 5)"
            echo "  -h, --help          Show this help"
            echo ""
            echo "Examples:"
            echo "  $0                                      # Run all benchmarks"
            echo "  $0 --type crypto                        # Run only crypto benchmarks"
            echo "  $0 --compare results/previous.json      # Compare with previous results"
            echo "  $0 --benchtime 30s --count 10           # Run longer benchmarks"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}Running benchmarks...${NC}"
echo "Type: $RUN_TYPE"
echo "Benchmark time: $BENCH_TIME"
echo "Count: $COUNT"
echo ""

# Determine which benchmarks to run
BENCH_PATTERN="."
case $RUN_TYPE in
    crypto)
        BENCH_PATTERN="^BenchmarkKey|^BenchmarkSign|^BenchmarkVerif|^BenchmarkMessage"
        ;;
    session)
        BENCH_PATTERN="^BenchmarkSession|^BenchmarkHandshake|^BenchmarkNonce|^BenchmarkConcurrent"
        ;;
    rfc9421)
        BENCH_PATTERN="^BenchmarkHTTP|^BenchmarkSignature|^BenchmarkHMAC|^BenchmarkPayload"
        ;;
    comparison)
        BENCH_PATTERN="^BenchmarkBaseline|^BenchmarkThroughput|^BenchmarkLatency|^BenchmarkMemory"
        ;;
    all)
        BENCH_PATTERN="."
        ;;
    *)
        echo -e "${RED}Unknown benchmark type: $RUN_TYPE${NC}"
        exit 1
        ;;
esac

# Run benchmarks
echo -e "${YELLOW}Executing benchmarks...${NC}"
go test -bench="$BENCH_PATTERN" \
    -benchmem \
    -benchtime="$BENCH_TIME" \
    -count="$COUNT" \
    -timeout=30m \
    ./$BENCHMARK_DIR \
    | tee "$RESULTS_FILE"

# Check if benchmarks ran successfully
if [ ${PIPESTATUS[0]} -ne 0 ]; then
    echo -e "${RED}Benchmarks failed!${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Benchmarks complete!${NC}"
echo "Results saved to: $RESULTS_FILE"
echo ""

# Note: JSON parsing requires benchstat or custom parser
# For now, the raw text results can be analyzed manually

# Generate analysis if analyze.go exists
if [ -f "./tools/benchmark/analyze.go" ]; then
    echo -e "${YELLOW}Generating analysis...${NC}"

    ANALYZE_ARGS="-input $RESULTS_FILE -output $ANALYSIS_FILE"
    if [ -n "$COMPARE_WITH" ]; then
        ANALYZE_ARGS="$ANALYZE_ARGS -compare $COMPARE_WITH"
    fi

    go run ./tools/benchmark/analyze.go $ANALYZE_ARGS

    if [ $? -eq 0 ]; then
        echo "Analysis saved to: $ANALYSIS_FILE"
        echo ""
        echo -e "${BLUE}Analysis Preview:${NC}"
        head -50 "$ANALYSIS_FILE"
    else
        echo -e "${RED}Failed to generate analysis${NC}"
    fi
else
    echo -e "${YELLOW}Note: Analysis tool not available. Results saved to text file.${NC}"
fi

# Summary
echo ""
echo "================================"
echo -e "${GREEN}Benchmark Summary${NC}"
echo "================================"
echo "Results: $RESULTS_FILE"
echo "JSON: $JSON_FILE"
echo "Analysis: $ANALYSIS_FILE"
echo ""

# Extract key metrics
echo -e "${BLUE}Quick Stats:${NC}"
grep -E "^Benchmark" "$RESULTS_FILE" | wc -l | xargs echo "Total benchmarks run:"
grep -E "PASS|FAIL" "$RESULTS_FILE" || echo "All tests passed"

# If comparison was made, show summary
if [ -n "$COMPARE_WITH" ]; then
    echo ""
    echo -e "${YELLOW}Performance Changes:${NC}"
    grep -E "⚠️|❌" "$ANALYSIS_FILE" | wc -l | xargs echo "Regressions found:"
fi

echo ""
echo -e "${GREEN}Done!${NC}"
