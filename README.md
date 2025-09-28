# CV Evaluator API

A comprehensive backend service that evaluates candidate CVs and project reports using AI workflows. The system implements LLM chaining, RAG (Retrieval Augmented Generation), and resilient error handling for robust candidate assessment.

## Features

- **File Upload Support**: Accepts CV and project reports in multiple formats (TXT, PDF, DOCX)
- **Real AI Integration**: Uses OpenAI GPT-4 for structured candidate assessment and embeddings
- **Cloud Storage**: MinIO integration for scalable file storage
- **RAG Integration**: Real vector embeddings and cosine similarity for context retrieval
- **PDF/DOCX Processing**: Real text extraction from PDF and Word documents
- **Async Processing**: Non-blocking evaluation with job status tracking
- **Resilience**: Circuit breaker pattern, exponential backoff retry, and graceful degradation
- **Standardized Scoring**: AI-powered consistent evaluation parameters

## API Endpoints

### Core Endpoints

- `POST /api/v1/upload` - Upload CV or project files
- `POST /api/v1/evaluate` - Start evaluation process
- `GET /api/v1/result/{id}` - Get evaluation results
- `GET /health` - Service health check
- `GET /` - API documentation

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
- OpenAI API Key
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

# LLM API Configuration
OPENAI_API_KEY=your_openai_api_key_here
OPENAI_MODEL=gpt-4
OPENAI_BASE_URL=https://api.openai.com/v1

# MinIO Storage Configuration
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=your_minio_access_key
MINIO_SECRET_KEY=your_minio_secret_key
MINIO_BUCKET=cv-evaluator
MINIO_USE_SSL=false

# Server Configuration
PORT=8080

# Vector Database Configuration
EMBEDDING_MODEL=text-embedding-ada-002
VECTOR_DIMENSIONS=1536
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

## Usage Examples

### 1. Upload Files

```bash
# Upload CV
curl -X POST -F "file=@cv.pdf" http://localhost:8080/api/v1/upload

# Upload Project Report
curl -X POST -F "file=@project.docx" http://localhost:8080/api/v1/upload
```

### 2. Start Evaluation

```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "cv_file": "uploads/cv-file-path.pdf",
    "job_description": "Senior Backend Engineer with AI/ML experience..."
  }' \
  http://localhost:8080/api/v1/evaluate
```

Or with job description file:
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{
    "cv_files": ["cv-files/candidate.pdf"],
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

**Completed:**
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

## Evaluation Criteria

### CV Assessment (Match Rate 0.0-1.0)

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

- **Correctness (25% weight)**
  - Meets stated requirements
  - Proper prompt design and engineering
  - LLM chaining implementation
  - RAG integration
  - Error handling implementation

- **Code Quality (20% weight)**
  - Clean, readable code
  - Modular architecture
  - Proper abstractions
  - Following best practices

- **Resilience (20% weight)**
  - Error handling and recovery
  - Retry mechanisms
  - Circuit breakers
  - Graceful degradation

- **Documentation (15% weight)**
  - Clear README
  - API documentation
  - Architecture decisions
  - Trade-offs explanation

- **Creativity/Bonus (20% weight)**
  - Additional features
  - Performance optimizations
  - Security implementations
  - Deployment automation

## Technical Implementation

### Real AI Integration

1. **OpenAI GPT-4 Integration**
   - Structured JSON extraction from CV content
   - Intelligent scoring with detailed feedback
   - Temperature-controlled responses for consistency
   - Retry logic with exponential backoff

2. **Vector Embeddings & RAG**
   - OpenAI text-embedding-ada-002 for document vectorization
   - Cosine similarity for context retrieval
   - Real-time embedding generation for queries
   - Semantic search for job requirements matching

3. **Cloud Storage with MinIO**
   - Scalable object storage for uploaded files
   - Unique file naming with UUID
   - Presigned URLs for secure access
   - Automatic bucket management

4. **Real Document Processing**
   - UniPDF for robust PDF text extraction
   - Fallback to ledongthuc/pdf for compatibility
   - Multi-page document handling
   - Error recovery for corrupted files

### Resilience Features

- **Circuit Breaker**: Prevents cascading failures when LLM APIs are down
- **Exponential Backoff**: Intelligent retry mechanism for transient failures
- **Graceful Degradation**: Continues operation with reduced functionality
- **Comprehensive Logging**: Detailed error tracking and monitoring

### File Processing

- **Supported Formats**: TXT, PDF, DOCX
- **Text Extraction**: Mock implementation (ready for real PDF/DOCX libraries)
- **File Validation**: Type checking and security validation
- **Unique Storage**: UUID-based file naming to prevent conflicts

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
| `OPENAI_API_KEY` | OpenAI API key for GPT-4 and embeddings | - | ✅ |
| `OPENAI_MODEL` | OpenAI model to use | `gpt-4` | ❌ |
| `MINIO_ENDPOINT` | MinIO server endpoint | - | ✅ |
| `MINIO_ACCESS_KEY` | MinIO access key | - | ✅ |
| `MINIO_SECRET_KEY` | MinIO secret key | - | ✅ |
| `MINIO_BUCKET` | MinIO bucket name | `cv-evaluator` | ❌ |
| `MINIO_USE_SSL` | Use SSL for MinIO connection | `false` | ❌ |
| `PORT` | Server port | `8080` | ❌ |
| `EMBEDDING_MODEL` | OpenAI embedding model | `text-embedding-ada-002` | ❌ |

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
