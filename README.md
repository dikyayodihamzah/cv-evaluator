# CV Evaluator API

A comprehensive backend service that evaluates candidate CVs and project reports using AI workflows. The system implements LLM chaining, RAG (Retrieval Augmented Generation), and resilient error handling for robust candidate assessment.

## Features

- **File Upload Support**: Accepts CV files in multiple formats (TXT, PDF, DOCX)
- **Real AI Integration**: Uses OpenAI GPT-4 for structured candidate assessment and embeddings
- **Cloud Storage**: MinIO integration for scalable file storage
- **RAG Integration**: Real vector embeddings and cosine similarity for context retrieval
- **PDF/DOCX Processing**: Real text extraction from PDF and Word documents
- **Flexible Job Descriptions**: Support for text input or file upload for job descriptions
- **Async Processing**: Non-blocking evaluation with job status tracking
- **Resilience**: Circuit breaker pattern, exponential backoff retry, and graceful degradation
- **Standardized Scoring**: AI-powered consistent evaluation parameters

## API Endpoints

### Core Endpoints

- `POST /api/v1/upload` - Upload CV files and job description files
- `POST /api/v1/evaluate` - Start evaluation process
- `GET /api/v1/result/{id}` - Get evaluation results
- `GET /health` - Service health check
- `GET /` - API documentation

### Evaluation Request Format

```json
{
  "cv_file": "cv-files/candidate.pdf",
  "project_file": "cv-files/project_report.pdf",
  "job_description": "Senior Backend Engineer with 5+ years experience...",
  "job_description_file": "cv-files/job_spec.txt"
}
```

**Required Fields:**
- `cv_file`: Path to uploaded CV file (required)
- Either `job_description` (text) OR `job_description_file` (file path) must be provided

**Optional Fields:**
- `project_file`: Path to uploaded project file (optional)

**Processing Logic:**
- **CV file**: Always required for candidate evaluation
- **Project content**: 
  - If `project_file` is provided → Full project evaluation with detailed scoring and feedback
  - If `project_file` is NOT provided → CV-only evaluation with default project response
- **Job description**: Can be provided as text or uploaded as a file
- **Evaluation modes**: 
  - **Full evaluation**: CV + Project assessment (when project file provided)
  - **CV-only evaluation**: CV assessment with informative project placeholder (when no project file)

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Controllers   │    │    Services     │    │   Data Layer    │
│                 │    │                 │    │                 │
│ • FileCtrl      │───▶│ • FileService   │    │ • File Storage  │
│ • EvalCtrl      │    │ • EvalService   │    │ • Job Memory    │
│                 │    │ • LLMService    │    │ • RAG Database  │
│                 │    │ • RAGService    │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   Resilience    │
                    │                 │
                    │ • Circuit       │
                    │   Breaker       │
                    │ • Retry Logic   │
                    │ • Error         │
                    │   Handling      │
                    └─────────────────┘
```

## Installation & Setup

### Prerequisites

- Go 1.24.4 or higher
- DeepSeek API Key (recommended) or OpenAI API Key
- MinIO Server (for cloud storage)
- Git

### Setup

1. **Clone the repository:**
```bash
git clone <repository-url>
cd cv-evaluator
```

2. **Install dependencies:**
```bash
go mod tidy
```

3. **Configure environment variables:**
Create a `.env` file in the project root:
```bash
# Environment Configuration
ENVIRONMENT=DEVELOPMENT

# LLM API Configuration (Choose one)
# Option 1: DeepSeek API (Recommended - Cost-effective)
DEEPSEEK_API_KEY=your_deepseek_api_key_here

# Option 2: OpenAI API (Alternative)
# OPENAI_API_KEY=your_openai_api_key_here

# MinIO Storage Configuration
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=your_minio_access_key
MINIO_SECRET_KEY=your_minio_secret_key
MINIO_BUCKET=cv-evaluator
MINIO_USE_SSL=false

# Server Configuration
PORT=8080
```

4. **Start MinIO Server:**
```bash
# Using Docker
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Or install MinIO locally and run:
minio server /path/to/data
```

5. **Build and run the application:**
```bash
go build -o cv-evaluator .
./cv-evaluator
```

The service will start on the configured port (default: 8080).

## Testing

The project includes comprehensive unit tests for all service layers using [testify](https://github.com/stretchr/testify) and [mockery](https://github.com/vektra/mockery) for mocking.

### Quick Test Commands

```bash
# Run all tests with our test runner script
./scripts/run-tests.sh

# Run specific service tests
./scripts/run-tests.sh -s file     # File service tests
./scripts/run-tests.sh -s llm      # LLM service tests
./scripts/run-tests.sh -s rag      # RAG service tests
./scripts/run-tests.sh -s evaluation # Evaluation service tests

# Run with coverage
./scripts/run-tests.sh -a

# Run benchmarks
./scripts/run-tests.sh -b

# Run with race detection
./scripts/run-tests.sh -r
```

### Manual Test Commands

```bash
# Run all service tests
go test ./service/... -v

# Run tests with coverage
go test ./service/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific service tests
go test ./service/filesrv -v
go test ./service/llmsrv -v
go test ./service/ragsrv -v
go test ./service/evalsrv -v
```

### Test Coverage

The test suite covers:
- **File Operations**: Upload validation, text extraction (PDF, DOCX, TXT)
- **LLM Integration**: CV extraction, scoring, project evaluation
- **RAG Functionality**: Document indexing, context retrieval, similarity matching
- **Job Management**: Async processing, status tracking, error handling
- **Logging Integration**: All operations include comprehensive logging
- **Error Scenarios**: Service failures, invalid inputs, edge cases

For detailed testing documentation, see [TESTING.md](TESTING.md).

## Usage Examples

### 1. Upload Files

```bash
# Upload CV file
curl -X POST -F "file=@cv.pdf" http://localhost:8080/api/v1/upload

# Upload Job Description (optional - can be provided as text)
curl -X POST -F "file=@job_description.txt" http://localhost:8080/api/v1/upload
```

### 2. Start Evaluation

**Option 1: Single CV file with text job description**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "cv_file": "cv-files/candidate.pdf",
    "job_description": "Senior Backend Engineer with AI/ML experience..."
  }' \
  http://localhost:8080/api/v1/evaluate
```

**Option 2: Separate CV and project files**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "cv_file": "cv-files/candidate.pdf",
    "project_file": "cv-files/project_report.pdf",
    "job_description": "Senior Backend Engineer with AI/ML experience..."
  }' \
  http://localhost:8080/api/v1/evaluate
```

**Option 3: With job description file**
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "cv_file": "cv-files/candidate.pdf",
    "project_file": "cv-files/project_report.pdf",
    "job_description_file": "cv-files/job_spec.txt"
  }' \
  http://localhost:8080/api/v1/evaluate
```

Response:
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "queued"
}
```

### 3. Check Results

```bash
curl http://localhost:8080/api/v1/result/123e4567-e89b-12d3-a456-426614174000
```

Responses:

**Processing:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "processing"
}
```

**Completed (with project file):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "completed",
  "result": {
    "cv_match_rate": 0.82,
    "cv_feedback": "Strong in backend and cloud, limited AI integration experience.",
    "project_score": 7.5,
    "project_feedback": "Meets prompt chaining requirements, lacks error handling robustness.",
    "overall_summary": "Good candidate fit, would benefit from deeper RAG knowledge."
  }
}
```

**Completed (CV-only, no project file):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "status": "completed",
  "result": {
    "cv_match_rate": 0.82,
    "cv_feedback": "Strong in backend and cloud, limited AI integration experience.",
    "project_score": 0.0,
    "project_feedback": "No project file provided for evaluation. To get a comprehensive project assessment, please upload a separate project file containing technical documentation, code samples, or project reports.",
    "overall_summary": "Good CV match for the position. Candidate shows solid qualifications with minor gaps in some areas. Project evaluation not available - consider uploading project files to get a comprehensive evaluation."
  }
}
```

## Evaluation Criteria

### CV Assessment (Match Rate 0.0-1.0)

The CV evaluation analyzes the uploaded CV file and extracts both CV content and project information for comprehensive assessment:

- **Technical Skills Match (40% weight)**
  - Backend technologies alignment
  - Cloud platform experience
  - Database knowledge
  - API development skills
  - AI/LLM exposure (bonus)

- **Experience Level (30% weight)**
  - Years of relevant experience
  - Project complexity and scale
  - Leadership and mentoring experience

- **Relevant Achievements (20% weight)**
  - Performance improvements
  - System design contributions
  - Team leadership
  - Process improvements

- **Cultural Fit (10% weight)**
  - Communication skills
  - Learning attitude
  - Collaboration indicators

### Project Assessment (Score 1-10)

Project information is automatically extracted from the CV file, analyzing any project descriptions, portfolios, or technical work mentioned:

- **Correctness (25% weight)**
  - Meets stated requirements
  - Proper technical implementation
  - Problem-solving approach
  - Requirements fulfillment

- **Code Quality (20% weight)**
  - Clean, readable code
  - Modular architecture
  - Proper abstractions
  - Following best practices

- **Technical Complexity (20% weight)**
  - System design skills
  - Technology choices
  - Architecture decisions
  - Scalability considerations

- **Documentation (15% weight)**
  - Project descriptions
  - Technical explanations
  - Problem statement clarity
  - Solution documentation

- **Impact/Innovation (20% weight)**
  - Business impact
  - Technical innovation
  - Performance improvements
  - Creative solutions

## Technical Implementation

### Real AI Integration

1. **LLM Integration (DeepSeek/OpenAI)**
   - Structured JSON extraction from CV content
   - Intelligent scoring with detailed feedback
   - Temperature-controlled responses for consistency
   - Retry logic with exponential backoff
   - Auto-detection of API provider

2. **Vector Embeddings & RAG**
   - OpenAI text-embedding-ada-002 for document vectorization (when using OpenAI)
   - Fallback to hash-based embeddings for DeepSeek
   - Cosine similarity for context retrieval
   - Real-time embedding generation for queries
   - Semantic search for job requirements matching

3. **Cloud Storage with MinIO**
   - Scalable object storage for uploaded files
   - Unique file naming with UUID
   - Presigned URLs for secure access
   - Automatic bucket management

4. **Real Document Processing**
   - **PDF Extraction**: Dual-engine approach using both `unidoc/unipdf/v3` and `ledongthuc/pdf` libraries
   - **DOCX Extraction**: Complete text extraction from paragraphs, tables, headers, and footers using `fumiama/go-docx`
   - **Text Files**: Direct UTF-8 text reading
   - **Error Recovery**: Graceful handling of corrupted files and encrypted PDFs
   - **Content Normalization**: Automatic whitespace cleanup and formatting

### Resilience Features

- **Circuit Breaker**: Prevents cascading failures when LLM APIs are down
- **Exponential Backoff**: Intelligent retry mechanism for transient failures
- **Graceful Degradation**: Continues operation with reduced functionality
- **Comprehensive Logging**: Detailed error tracking and monitoring

### File Processing

- **Supported Formats**: TXT, PDF, DOCX for CV files
- **Advanced Text Extraction**:
  - **PDF**: Robust dual-engine extraction with encryption handling
    - Primary: `unidoc/unipdf/v3` for complex PDFs with OCR capabilities
    - Fallback: `ledongthuc/pdf` for basic PDF parsing
    - Handles encrypted PDFs (tries empty password first)
    - Graceful error recovery for corrupted files
  - **DOCX**: Complete Microsoft Word document processing
    - Extracts text from paragraphs, tables, headers, and footers
    - Preserves document structure with proper spacing
    - Handles nested tables and complex formatting
    - Uses `fumiama/go-docx` for reliable parsing
  - **TXT**: Direct UTF-8 text file reading
- **Content Intelligence**: 
  - **Single-file mode**: Automatically extracts both CV and project information from CV files
  - **Multi-file mode**: Processes separate CV and project files for detailed evaluation
  - **Flexible input**: Supports both approaches based on user preference
- **File Validation**: Type checking and security validation (10MB max per file)
- **Cloud Storage**: MinIO-based storage with UUID naming to prevent conflicts
- **Job Description Flexibility**: Supports both text input and file upload for job descriptions

## Configuration

The application uses environment variables for configuration:

- `ENVIRONMENT`: Set to `DEVELOPMENT`, `LOCAL`, or `PRODUCTION`
- Default port: `8080`

## Development

### Project Structure

```
cv-evaluator/
├── controller/          # HTTP request handlers
│   ├── evalctrl/       # Evaluation endpoints
│   └── filectrl/       # File upload endpoints
├── service/            # Business logic layer
│   ├── evalsrv/       # Evaluation service
│   ├── filesrv/       # File processing service
│   ├── llmsrv/        # LLM integration service
│   └── ragsrv/        # RAG service
├── pkg/               # Shared packages
│   ├── exception/     # Error handling
│   ├── lib/          # Common utilities
│   ├── log/          # Logging utilities
│   ├── model/        # Data models
│   └── resilience/   # Circuit breaker & retry
└── main.go           # Application entry point
```

### Adding New Features

1. **New Evaluation Criteria**: Update `llmsrv` scoring functions
2. **File Format Support**: Extend `filesrv` extraction methods  
3. **Additional Endpoints**: Create new controllers following existing patterns
4. **Enhanced RAG**: Improve vector similarity in `ragsrv`
5. **CV Content Parsing**: Enhance intelligent separation of CV and project content

## Monitoring & Observability

- **Health Checks**: `/health` endpoint for service monitoring
- **Structured Logging**: Comprehensive request/response logging
- **Error Tracking**: Detailed error messages with context
- **Job Status Tracking**: Real-time evaluation progress monitoring

## Dependencies

### Core Libraries

- **[go-openai](https://github.com/sashabaranov/go-openai)**: OpenAI API client for GPT-4 and embeddings
- **[minio-go](https://github.com/minio/minio-go)**: MinIO client for cloud storage
- **[unipdf](https://github.com/unidoc/unipdf)**: Professional PDF processing
- **[ledongthuc/pdf](https://github.com/ledongthuc/pdf)**: Alternative PDF text extraction
- **[fiber](https://github.com/gofiber/fiber)**: Fast HTTP web framework
- **[uuid](https://github.com/google/uuid)**: UUID generation for file naming

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DEEPSEEK_API_KEY` | DeepSeek API key (recommended) | - | ✅* |
| `OPENAI_API_KEY` | OpenAI API key (alternative) | - | ✅* |
| `LLM_MODEL` | LLM model to use | Auto-detected | ❌ |
| `LLM_BASE_URL` | LLM API base URL | Auto-detected | ❌ |
| `MINIO_ENDPOINT` | MinIO server endpoint | - | ✅ |
| `MINIO_ACCESS_KEY` | MinIO access key | - | ✅ |
| `MINIO_SECRET_KEY` | MinIO secret key | - | ✅ |
| `MINIO_BUCKET` | MinIO bucket name | `cv-evaluator` | ❌ |
| `MINIO_USE_SSL` | Use SSL for MinIO connection | `false` | ❌ |
| `PORT` | Server port | `8080` | ❌ |
| `EMBEDDING_MODEL` | Embedding model (OpenAI only) | `text-embedding-ada-002` | ❌ |

*Either `DEEPSEEK_API_KEY` or `OPENAI_API_KEY` is required

## Future Enhancements

- **Alternative LLM Providers**: Support for Anthropic Claude, local models
- **Persistent Storage**: PostgreSQL/MongoDB integration for job persistence
- **Authentication**: JWT-based API security with role-based access
- **Rate Limiting**: API throttling and quota management
- **Webhooks**: Callback notifications for completed evaluations
- **Dashboard**: Web UI for evaluation management and analytics
- **Metrics**: Prometheus integration for operational metrics
- **Caching**: Redis integration for performance optimization

## Contributing

1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Submit a pull request

## License

This project is licensed under the MIT License.

## Support

For questions and support, please open an issue in the repository.
