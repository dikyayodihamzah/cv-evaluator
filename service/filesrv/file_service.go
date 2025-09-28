package filesrv

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/exception"
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
	// set default path
	if path == "" {
		path = "cv-files"
	}

	bucket := env.GetString("MINIO_BUCKET", "cv-evaluator")

	resURLs := make([]string, 0)
	for _, file := range files {
		// retrieve file extension
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == "" {
			return nil, exception.ErrorBadRequest("File must have an extension")
		}

		// validate file type
		if !s.isValidFileType(ext) {
			return nil, exception.ErrorBadRequest("Unsupported file type: " + ext)
		}

		// validate file size (max 10MB)
		maxSize := int64(10 * 1024 * 1024) // 10MB
		if file.Size > maxSize {
			return nil, exception.ErrorBadRequest("File size exceeds 10MB limit")
		}

		// generate unique file name with timestamp and uuid
		fileName := uuid.New().String() + ext
		objectName := fmt.Sprintf("%s/%s", path, fileName)

		// open file
		buffer, err := file.Open()
		if err != nil {
			return nil, exception.ErrorInternal("Opening File Failed", err.Error())
		}
		defer buffer.Close()

		// prepare content type
		contentType := "application/octet-stream"
		if len(file.Header["Content-Type"]) > 0 {
			contentType = file.Header["Content-Type"][0]
		}

		// upload to MinIO
		if _, err := s.minioClient.PutObject(ctx, bucket, objectName, buffer, file.Size,
			minio.PutObjectOptions{ContentType: contentType},
		); err != nil {
			return nil, exception.ErrorInternal("Upload File Failed", err.Error())
		}

		resURLs = append(resURLs, objectName)
	}

	return resURLs, nil
}

func (f *fileService) ExtractText(objectName string) (string, error) {
	if objectName == "" {
		return "", fmt.Errorf("object name is empty")
	}

	bucket := env.GetString("MINIO_BUCKET", "cv-evaluator")

	// Get file from MinIO
	object, err := f.minioClient.GetObject(context.Background(), bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get file from storage: %w", err)
	}
	defer object.Close()

	// Read file data
	fileData, err := io.ReadAll(object)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(objectName))

	switch ext {
	case ".txt":
		return f.extractFromTxt(fileData)
	case ".pdf":
		return f.extractFromPDF(fileData)
	case ".docx":
		return f.extractFromDocx(fileData)
	default:
		// Try to read as plain text
		return f.extractFromTxt(fileData)
	}
}

func (f *fileService) isValidFileType(ext string) bool {
	validTypes := []string{".txt", ".pdf", ".docx", ".doc"}
	return slices.Contains(validTypes, ext)
}

func (f *fileService) extractFromTxt(fileData []byte) (string, error) {
	return string(fileData), nil
}

func (f *fileService) extractFromPDF(fileData []byte) (string, error) {
	// For PDF files, return a placeholder implementation
	// In a real implementation, you would use a PDF library
	return fmt.Sprintf(`PDF File Content

This is a PDF file with %d bytes of data.
For full PDF text extraction, please integrate a PDF parsing library such as:
- github.com/ledongthuc/pdf
- github.com/unidoc/unipdf/v3

Treating as document file requiring proper PDF processing implementation.`, len(fileData)), nil
}

func (f *fileService) extractFromDocx(fileData []byte) (string, error) {
	// For DOCX files, return a placeholder implementation
	// In a real implementation, you would use a DOCX library
	return fmt.Sprintf(`DOCX File Content

This is a DOCX file with %d bytes of data.
For full DOCX text extraction, please integrate a DOCX parsing library such as:
- github.com/nguyenthenguyen/docx
- github.com/fumiama/go-docx

Treating as document file requiring proper DOCX processing implementation.`, len(fileData)), nil
}
