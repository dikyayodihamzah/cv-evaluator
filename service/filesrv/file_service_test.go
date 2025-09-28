package filesrv

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// FileServiceTestSuite contains test cases for file service
type FileServiceTestSuite struct {
	suite.Suite
	fileService *fileService
}

func (suite *FileServiceTestSuite) SetupTest() {
	// Create service with nil minio client for testing helper functions
	suite.fileService = &fileService{
		minioClient: nil, // We'll test helper functions that don't need MinIO
	}
}

func TestFileServiceSuite(t *testing.T) {
	suite.Run(t, new(FileServiceTestSuite))
}

// Test ExtractText basic validation
func (suite *FileServiceTestSuite) TestExtractText_EmptyObjectName() {
	result, err := suite.fileService.ExtractText("")
	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "object name is empty")
}

// Test helper functions
func (suite *FileServiceTestSuite) TestIsValidFileType() {
	// Valid types
	assert.True(suite.T(), suite.fileService.isValidFileType(".txt"))
	assert.True(suite.T(), suite.fileService.isValidFileType(".pdf"))
	assert.True(suite.T(), suite.fileService.isValidFileType(".docx"))
	assert.True(suite.T(), suite.fileService.isValidFileType(".doc"))

	// Invalid types
	assert.False(suite.T(), suite.fileService.isValidFileType(".exe"))
	assert.False(suite.T(), suite.fileService.isValidFileType(".bat"))
	assert.False(suite.T(), suite.fileService.isValidFileType(".zip"))
	assert.False(suite.T(), suite.fileService.isValidFileType(""))
}

func (suite *FileServiceTestSuite) TestExtractFromTxt() {
	testData := []byte("Hello, World!")
	result, err := suite.fileService.extractFromTxt(testData)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Hello, World!", result)
}

func (suite *FileServiceTestSuite) TestExtractFromTxt_EmptyContent() {
	testData := []byte("")
	result, err := suite.fileService.extractFromTxt(testData)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "", result)
}

// Test PDF extraction (this would fail with real PDF parsing, but tests the error handling)
func (suite *FileServiceTestSuite) TestExtractFromPDF_InvalidPDF() {
	// Invalid PDF data should trigger error handling
	invalidPDFData := []byte("not a real PDF")
	result, err := suite.fileService.extractFromPDF(invalidPDFData)

	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to extract text from PDF")
}

// Test DOCX extraction (this would fail with real DOCX parsing, but tests the error handling)
func (suite *FileServiceTestSuite) TestExtractFromDocx_InvalidDocx() {
	// Invalid DOCX data should trigger error handling
	invalidDocxData := []byte("not a real DOCX")
	result, err := suite.fileService.extractFromDocx(invalidDocxData)

	assert.Error(suite.T(), err)
	assert.Empty(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to parse DOCX file")
}

// Test cleanupText function
func (suite *FileServiceTestSuite) TestCleanupText() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multiple newlines",
			input:    "line1\n\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "carriage returns",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "mixed whitespace",
			input:    "  line1  \n  line2  \n  ",
			expected: "line1\nline2",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := suite.fileService.cleanupText(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkFileService_ExtractFromTxt(b *testing.B) {
	service := &fileService{}
	data := []byte(strings.Repeat("test content ", 1000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.extractFromTxt(data)
	}
}

func BenchmarkFileService_IsValidFileType(b *testing.B) {
	service := &fileService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.isValidFileType(".pdf")
	}
}

// Simple unit tests for helper functions
func TestFileService_HelperFunctions(t *testing.T) {
	service := &fileService{}

	// Test file type validation
	assert.True(t, service.isValidFileType(".pdf"))
	assert.True(t, service.isValidFileType(".txt"))
	assert.False(t, service.isValidFileType(".exe"))

	// Test text extraction
	result, err := service.extractFromTxt([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, "test", result)
}
