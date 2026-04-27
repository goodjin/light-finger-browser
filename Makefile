.PHONY: test test-integration test-unit test-clean test-up test-down test-logs test-coverage

# Test configuration
TEST_TIMEOUT ?= 120s
TEST_PARALLEL ?= 4
COVERAGE_DIR ?= coverage

# Default target
all: test-integration

# Start test environment
test-up:
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for services to be healthy..."
	@for i in $$(seq 1 30); do \
		pg_isready -h localhost -p 5432 -U test -d tmos_test > /dev/null 2>&1 && \
		redis-cli -h localhost -p 6379 ping > /dev/null 2>&1 && \
		echo "All services ready" && exit 0; \
		echo "Waiting for services... ($$i/30)"; \
		sleep 2; \
	done
	@echo "Services failed to start in time"
	@exit 1

# Stop test environment
test-down:
	docker-compose -f docker-compose.test.yml down -v

# Show test logs
test-logs:
	docker-compose -f docker-compose.test.yml logs -f

# Run all tests
test: test-up
	go test -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) ./...

# Run integration tests only
test-integration: test-up
	go test -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) -tags=integration ./integration/...

# Run unit tests only
test-unit:
	go test -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) ./internal/...

# Run tests with coverage
test-coverage: test-up
	mkdir -p $(COVERAGE_DIR)
	go test -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) \
		-coverprofile=$(COVERAGE_DIR)/coverage.out \
		-covermode=atomic \
		./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

# Run specific integration test file
test-integration/%: test-up
	go test -v -timeout=$(TEST_TIMEOUT) -tags=integration ./integration/$*

# Clean test environment and coverage
test-clean: test-down
	rm -rf $(COVERAGE_DIR)

# Run tests with race detector
test-race: test-up
	go test -v -timeout=$(TEST_TIMEOUT) -parallel=$(TEST_PARALLEL) -race ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Generate test report
test-report: test-integration
	mkdir -p docs/v1/04-execution/report
	gotestsum --jsonfile=docs/v1/04-execution/report/test-results.json -- \
		-timeout=$(TEST_TIMEOUT) ./integration/...
	mv docs/v1/04-execution/report/test-results.json docs/v1/04-execution/report/test-results-$$(date +%Y%m%d-%H%M%S).json