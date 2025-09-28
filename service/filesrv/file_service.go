package filesrv

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/exception"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/log"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type FileService interface {
	Upload(ctx context.Context, path string, files []*multipart.FileHeader) ([]string, error)
	ExtractText(filePath string) (string, error)
}

type fileService struct {
	minioClient *minio.Client
}

func New(minioClient *minio.Client) FileService {
	return &fileService{
		minioClient: minioClient,
	}
}

func (s *fileService) Upload(ctx context.Context, path string, files []*multipart.FileHeader) ([]string, error) {
	logger := log.WithContext("FileService", "Upload")
	start := time.Now()

	logger.Info("Starting file upload, %d files", len(files))

	// set default path
	if path == "" {
		path = "cv-files"
	}

	bucket := env.GetString("MINIO_BUCKET", "cv-evaluator")
	logger.Debug("Uploading to bucket: %s, path: %s", bucket, path)

	resURLs := make([]string, 0)
	for i, file := range files {
		logger.Debug("Processing file %d/%d: %s (size: %d bytes)", i+1, len(files), file.Filename, file.Size)
		// retrieve file extension
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == "" {
			logger.Error("File %s has no extension", file.Filename)
			return nil, exception.ErrorBadRequest("File must have an extension")
		}

		// validate file type
		if !s.isValidFileType(ext) {
			logger.Error("Unsupported file type: %s for file %s", ext, file.Filename)
			return nil, exception.ErrorBadRequest("Unsupported file type: " + ext)
		}

		// validate file size (max 10MB)
		maxSize := int64(10 * 1024 * 1024) // 10MB
		if file.Size > maxSize {
			logger.Error("File %s exceeds size limit: %d bytes > %d bytes", file.Filename, file.Size, maxSize)
			return nil, exception.ErrorBadRequest("File size exceeds 10MB limit")
		}

		logger.Debug("File validation passed for %s", file.Filename)

		// generate unique file name with timestamp and uuid
		fileName := uuid.New().String() + ext
		objectName := fmt.Sprintf("%s/%s", path, fileName)
		logger.Debug("Generated object name: %s", objectName)

		// open file
		buffer, err := file.Open()
		if err != nil {
			logger.Error("Failed to open file %s: %v", file.Filename, err)
			return nil, exception.ErrorInternal("Opening File Failed", err.Error())
		}
		defer buffer.Close()

		// prepare content type
		contentType := "application/octet-stream"
		if len(file.Header["Content-Type"]) > 0 {
			contentType = file.Header["Content-Type"][0]
		}
		logger.Debug("Using content type: %s", contentType)

		// upload to MinIO
		logger.Debug("Uploading file to MinIO: %s", objectName)
		if _, err := s.minioClient.PutObject(ctx, bucket, objectName, buffer, file.Size,
			minio.PutObjectOptions{ContentType: contentType},
		); err != nil {
			logger.Error("Failed to upload file %s to MinIO: %v", objectName, err)
			return nil, exception.ErrorInternal("Upload File Failed", err.Error())
		}

		logger.Debug("File uploaded successfully: %s", objectName)
		resURLs = append(resURLs, objectName)
	}

	logger.Info("All files uploaded successfully, %d files processed", len(files))
	logger.WithDuration(start)
	return resURLs, nil
}

func (f *fileService) ExtractText(objectName string) (string, error) {
	logger := log.WithContext("FileService", "ExtractText")
	start := time.Now()

	logger.Info("Starting text extraction from object: %s", objectName)

	if objectName == "" {
		logger.Error("Empty object name provided")
		return "", fmt.Errorf("object name is empty")
	}

	bucket := env.GetString("MINIO_BUCKET", "cv-evaluator")
	logger.Debug("Reading from bucket: %s, object: %s", bucket, objectName)

	// Get file from MinIO
	object, err := f.minioClient.GetObject(context.Background(), bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		logger.Error("Failed to get object from MinIO: %v", err)
		return "", fmt.Errorf("failed to get file from storage: %w", err)
	}
	defer object.Close()

	// Read file data
	fileData, err := io.ReadAll(object)
	if err != nil {
		logger.Error("Failed to read file data: %v", err)
		return "", fmt.Errorf("failed to read file data: %w", err)
	}
	logger.Debug("File data read successfully: %d bytes", len(fileData))

	ext := strings.ToLower(filepath.Ext(objectName))
	logger.Debug("Detected file extension: %s", ext)

	var text string
	switch ext {
	case ".txt":
		logger.Debug("Using TXT extraction method")
		text, err = f.extractFromTxt(fileData)
	case ".pdf":
		logger.Debug("Using PDF extraction method")
		text, err = f.extractFromPDF(fileData)
	case ".docx":
		logger.Debug("Using DOCX extraction method")
		text, err = f.extractFromDocx(fileData)
	default:
		logger.Debug("Unknown extension, trying TXT extraction method")
		// Try to read as plain text
		text, err = f.extractFromTxt(fileData)
	}

	if err != nil {
		logger.Error("Text extraction failed: %v", err)
		return "", err
	}

	logger.Info("Text extraction completed successfully, extracted %d characters", len(text))
	logger.WithDuration(start)
	return text, nil
}
