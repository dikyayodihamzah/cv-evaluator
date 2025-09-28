package filesrv

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/dikyayodihamzah/cv-evaluator/pkg/exception"
	"github.com/fumiama/go-docx"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"github.com/minio/minio-go/v7"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
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
	// Try with unipdf first (more robust for complex PDFs)
	text, err := f.extractPDFWithUnipdf(fileData)
	if err == nil && strings.TrimSpace(text) != "" {
		return text, nil
	}

	// Fallback to ledongthuc/pdf
	text, err = f.extractPDFWithLedongthuc(fileData)
	if err == nil && strings.TrimSpace(text) != "" {
		return text, nil
	}

	// If both fail, return a meaningful error
	return "", fmt.Errorf("failed to extract text from PDF: both extraction methods failed")
}

func (f *fileService) extractPDFWithUnipdf(fileData []byte) (string, error) {
	reader := bytes.NewReader(fileData)
	pdfReader, err := model.NewPdfReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Check if PDF is encrypted
	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		return "", fmt.Errorf("failed to check encryption: %w", err)
	}

	if isEncrypted {
		// Try to decrypt with empty password
		success, err := pdfReader.Decrypt([]byte(""))
		if err != nil || !success {
			return "", fmt.Errorf("PDF is encrypted and cannot be decrypted")
		}
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("failed to get number of pages: %w", err)
	}

	var textBuilder strings.Builder

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			continue // Skip pages that can't be read
		}

		ex, err := extractor.New(page)
		if err != nil {
			continue // Skip pages that can't be processed
		}

		pageText, err := ex.ExtractText()
		if err != nil {
			continue // Skip pages with extraction errors
		}

		textBuilder.WriteString(pageText)
		textBuilder.WriteString("\n\n")
	}

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return text, nil
}

func (f *fileService) extractPDFWithLedongthuc(fileData []byte) (string, error) {
	reader := bytes.NewReader(fileData)
	pdfReader, err := pdf.NewReader(reader, int64(len(fileData)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var textBuilder strings.Builder
	numPages := pdfReader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue // Skip pages with errors
		}

		textBuilder.WriteString(pageText)
		textBuilder.WriteString("\n\n")
	}

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return text, nil
}

func (f *fileService) extractFromDocx(fileData []byte) (string, error) {
	reader := bytes.NewReader(fileData)

	// Parse the DOCX file
	doc, err := docx.Parse(reader, int64(len(fileData)))
	if err != nil {
		return "", fmt.Errorf("failed to parse DOCX file: %w", err)
	}

	var textBuilder strings.Builder

	// Extract text from document body items
	for _, item := range doc.Document.Body.Items {
		switch v := item.(type) {
		case *docx.Paragraph:
			// Extract text from paragraph using built-in String() method
			paragraphText := strings.TrimSpace(v.String())
			if paragraphText != "" {
				textBuilder.WriteString(paragraphText)
				textBuilder.WriteString("\n")
			}
		case *docx.Table:
			// Extract text from table
			tableText := f.extractTableText(v)
			if tableText != "" {
				textBuilder.WriteString(tableText)
				textBuilder.WriteString("\n")
			}
		}
	}

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		return "", fmt.Errorf("no text content found in DOCX file")
	}

	// Clean up excessive whitespace
	text = f.cleanupText(text)

	return text, nil
}

// extractTableText extracts text from a table
func (f *fileService) extractTableText(table *docx.Table) string {
	var textBuilder strings.Builder

	for _, row := range table.TableRows {
		for _, cell := range row.TableCells {
			cellText := f.extractTableCellText(cell)
			if cellText != "" {
				textBuilder.WriteString(cellText)
				textBuilder.WriteString("\t") // Tab separation for table cells
			}
		}
		textBuilder.WriteString("\n") // New line for table rows
	}

	return strings.TrimSpace(textBuilder.String())
}

// extractTableCellText extracts text from a table cell
func (f *fileService) extractTableCellText(cell *docx.WTableCell) string {
	var textBuilder strings.Builder

	// Extract text from paragraphs in the cell using built-in String() method
	for _, paragraph := range cell.Paragraphs {
		paragraphText := strings.TrimSpace(paragraph.String())
		if paragraphText != "" {
			textBuilder.WriteString(paragraphText)
			textBuilder.WriteString(" ")
		}
	}

	// Extract text from nested tables in the cell
	for _, table := range cell.Tables {
		tableText := f.extractTableText(table)
		if tableText != "" {
			textBuilder.WriteString(tableText)
			textBuilder.WriteString(" ")
		}
	}

	return strings.TrimSpace(textBuilder.String())
}

// cleanupText removes excessive whitespace and normalizes line breaks
func (f *fileService) cleanupText(text string) string {
	// Replace multiple consecutive newlines with double newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Split into lines and clean each line
	lines := strings.Split(text, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Trim whitespace from each line
		cleanLine := strings.TrimSpace(line)
		cleanLines = append(cleanLines, cleanLine)
	}

	// Join lines back and normalize spacing
	cleaned := strings.Join(cleanLines, "\n")

	// Replace multiple consecutive newlines with just two newlines (paragraph break)
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(cleaned)
}
