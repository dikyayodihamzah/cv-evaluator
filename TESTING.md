# Testing Guide for CV Evaluator API

This document provides comprehensive information about the testing strategy and implementation for the CV Evaluator service layer.

## Test Architecture

### Testing Framework
- **Primary Framework**: [testify](https://github.com/stretchr/testify) for assertions and test suites
- **Mocking Framework**: [mockery](https://github.com/vektra/mockery) for generating interface mocks
- **Test Structure**: Table-driven tests with test suites for complex scenarios

### Mock Generation
Mocks are automatically generated using mockery from service interfaces:

```bash
# Generate all mocks
mockery

# Generate specific interface mock
mockery --name=FileService --dir=./service/filesrv --output=./mocks
```

Generated mocks are located in the `mocks/` directory:
- `mock_FileService.go`
- `mock_LLMService.go` 
- `mock_RAGService.go`
- `mock_EvaluationService.go`

## Service Layer Tests

### File Service Tests (`service/filesrv/file_service_test.go`)

**Coverage Areas:**
- File upload validation (type, size, format)
- Text extraction from multiple file formats (TXT, PDF, DOCX)
- MinIO integration (mocked)
- Error handling for invalid files
- Helper function validation

**Key Test Cases:**
```go
func (suite *FileServiceTestSuite) TestUpload_Success()
func (suite *FileServiceTestSuite) TestExtractText_Success()
func (suite *FileServiceTestSuite) TestIsValidFileType()
func (suite *FileServiceTestSuite) TestExtractFromPDF_InvalidPDF()
```

**Example Usage:**
```bash
# Run file service tests
go test ./service/filesrv -v

# Run with coverage
go test ./service/filesrv -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### LLM Service Tests (`service/llmsrv/llm_service_test.go`)

**Coverage Areas:**
- Full candidate evaluation (CV + Project)
- CV-only evaluation mode
- OpenAI client integration (mocked)
- RAG service integration (mocked)
- Error handling for API failures
- Summary generation logic

**Key Test Cases:**
```go
func (suite *LLMServiceTestSuite) TestEvaluateCandidate_Success()
func (suite *LLMServiceTestSuite) TestEvaluateCVOnly_Success()
func (suite *LLMServiceTestSuite) TestExtractCVInfo_Success()
func (suite *LLMServiceTestSuite) TestGenerateOverallSummary()
```

**Mock Setup Example:**
```go
suite.mockRAGService.EXPECT().
    GetRelevantContext("cv_evaluation", jobDesc).
    Return("CV evaluation context", nil)

suite.mockOpenAIClient.On("ExtractCVInfo", cvContent).
    Return(mockCVInfo, nil)
```

### RAG Service Tests (`service/ragsrv/rag_service_test.go`)

**Coverage Areas:**
- Document indexing with embeddings
- Context retrieval using vector similarity
- Keyword-based fallback matching
- Cosine similarity calculations
- Error handling for embedding failures

**Key Test Cases:**
```go
func (suite *RAGServiceTestSuite) TestIndexDocument_Success()
func (suite *RAGServiceTestSuite) TestGetRelevantContext_Success()
func (suite *RAGServiceTestSuite) TestCosineSimilarity()
func (suite *RAGServiceTestSuite) TestCalculateKeywordSimilarity()
```

**Algorithm Testing:**
```go
// Test cosine similarity with different vector combinations
testCases := []struct {
    name     string
    a        []float32
    b        []float32
    expected float64
}{
    {"identical vectors", []float32{1.0, 0.0, 0.0}, []float32{1.0, 0.0, 0.0}, 1.0},
    {"orthogonal vectors", []float32{1.0, 0.0}, []float32{0.0, 1.0}, 0.0},
}
```

### Evaluation Service Tests (`service/evalsrv/evaluation_service_test.go`)

**Coverage Areas:**
- Asynchronous job processing
- Job lifecycle management (queued → processing → completed/failed)
- Full vs CV-only evaluation modes
- Error handling and job failure scenarios
- Concurrent job access safety

**Key Test Cases:**
```go
func (suite *EvaluationServiceTestSuite) TestStartEvaluation_Success()
func (suite *EvaluationServiceTestSuite) TestProcessEvaluation_FullEvaluation_Success()
func (suite *EvaluationServiceTestSuite) TestProcessEvaluation_CVOnly_Success()
func (suite *EvaluationServiceTestSuite) TestConcurrentJobAccess()
```

**Async Testing Pattern:**
```go
// Start evaluation
jobID, err := suite.evalService.StartEvaluation(req)
assert.NoError(suite.T(), err)

// Wait for async processing
time.Sleep(3 * time.Second)

// Verify result
result, err := suite.evalService.GetResult(jobID)
assert.NoError(suite.T(), err)
assert.Equal(suite.T(), "completed", result.Status)
```

## Running Tests

### Individual Service Tests
```bash
# File Service
go test ./service/filesrv -v

# LLM Service  
go test ./service/llmsrv -v

# RAG Service
go test ./service/ragsrv -v

# Evaluation Service
go test ./service/evalsrv -v
```

### All Service Tests
```bash
# Run all service layer tests
go test ./service/... -v

# With coverage
go test ./service/... -v -coverprofile=coverage.out

# Short mode (skip long-running tests)
go test ./service/... -v -short
```

### Benchmark Tests
```bash
# Run benchmarks
go test ./service/... -bench=. -benchmem

# Specific benchmarks
go test ./service/filesrv -bench=BenchmarkFileService_ExtractFromTxt
go test ./service/llmsrv -bench=BenchmarkLLMService_GenerateOverallSummary
```

## Test Data and Mocking Strategy

### Mock External Dependencies
- **MinIO Client**: Mocked for file storage operations
- **OpenAI Client**: Mocked for LLM API calls
- **Service Interfaces**: Auto-generated mocks using mockery

### Test Data Patterns
```go
// Structured test cases
testCases := []struct {
    name     string
    input    string
    expected string
    hasError bool
}{
    {"valid input", "test content", "processed content", false},
    {"empty input", "", "", true},
}

for _, tc := range testCases {
    suite.T().Run(tc.name, func(t *testing.T) {
        result, err := service.Process(tc.input)
        if tc.hasError {
            assert.Error(t, err)
        } else {
            assert.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        }
    })
}
```

### Error Simulation
```go
// Simulate external service failures
suite.mockFileService.EXPECT().
    ExtractText("invalid-file.pdf").
    Return("", errors.New("file corruption error"))
```

## Test Coverage Goals

### Current Coverage Areas
- **File Operations**: Upload, validation, text extraction
- **LLM Integration**: CV extraction, scoring, project evaluation
- **RAG Functionality**: Document indexing, context retrieval
- **Job Management**: Async processing, status tracking
- **Error Handling**: Service failures, invalid inputs
- **Concurrency**: Thread-safe operations

### Coverage Metrics
```bash
# Generate coverage report
go test ./service/... -coverprofile=coverage.out
go tool cover -func=coverage.out

# HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

## Integration vs Unit Tests

### Unit Tests (Current Implementation)
- **Scope**: Individual service methods
- **Dependencies**: Mocked external services
- **Speed**: Fast execution
- **Isolation**: Tests don't depend on external systems

### Integration Tests (Future Enhancement)
- **Scope**: End-to-end service interactions
- **Dependencies**: Real external services (optional)
- **Setup**: Docker containers for MinIO, test LLM endpoints
- **Usage**: Validation of complete workflows

## Best Practices

### Test Organization
1. **Test Suites**: Group related tests using testify suites
2. **Setup/Teardown**: Use `SetupTest()` for consistent initialization
3. **Mock Management**: Create mocks in setup, verify in tests
4. **Assertion Quality**: Use specific assertions with clear error messages

### Mock Usage
```go
// Good: Specific expectations
suite.mockService.EXPECT().
    ProcessData("specific-input").
    Return("expected-output", nil).
    Once()

// Better: With argument matchers
suite.mockService.EXPECT().
    ProcessData(mock.MatchedBy(func(input string) bool {
        return len(input) > 0
    })).
    Return("dynamic-output", nil)
```

### Error Testing
```go
// Test both success and failure paths
func (suite *TestSuite) TestOperation_Success() { /* ... */ }
func (suite *TestSuite) TestOperation_Error() { /* ... */ }
func (suite *TestSuite) TestOperation_InvalidInput() { /* ... */ }
```

## Debugging Tests

### Verbose Output
```bash
# Detailed test output
go test ./service/filesrv -v -run TestFileService_Upload

# With coverage and race detection
go test ./service/... -v -race -coverprofile=coverage.out
```

### Test Debugging
```go
// Add debug logging in tests
suite.T().Logf("Testing with input: %+v", testInput)
suite.T().Logf("Mock was called %d times", mockService.AssertNumberOfCalls(suite.T(), "Method", 1))
```

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Run Tests
  run: |
    go test ./service/... -v -race -coverprofile=coverage.out
    go tool cover -func=coverage.out
```

### Test Requirements
- All tests must pass before deployment
- Minimum 80% code coverage for service layer
- No race conditions detected
- All mocks properly verified

## Troubleshooting

### Common Issues
1. **Import Cycles**: Keep mocks in separate package or use interfaces
2. **Mock Expectations**: Ensure all expected calls are made
3. **Async Tests**: Use appropriate timeouts for goroutine tests
4. **Resource Cleanup**: Properly close files and connections in tests

### Mock Debugging
```go
// Verify mock expectations were met
suite.mockService.AssertExpectations(suite.T())

// Check specific method calls
suite.mockService.AssertCalled(suite.T(), "MethodName", expectedArgs...)
```

This testing infrastructure provides comprehensive coverage of the service layer while maintaining fast execution and reliable test isolation through effective mocking strategies.
