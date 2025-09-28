package filesrv

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/log"
	"github.com/fumiama/go-docx"
	"github.com/ledongthuc/pdf"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

func (f *fileService) isValidFileType(ext string) bool {
	validTypes := []string{".txt", ".pdf", ".docx", ".doc"}
	return slices.Contains(validTypes, ext)
}

func (f *fileService) extractFromTxt(fileData []byte) (string, error) {
	logger := log.WithContext("FileService", "ExtractTxt")
	start := time.Now()

	logger.Debug("Starting TXT extraction, size: %d bytes", len(fileData))

	text := string(fileData)
	logger.Info("TXT extraction completed successfully, extracted %d characters", len(text))
	logger.WithDuration(start)

	return text, nil
}

func (f *fileService) extractFromPDF(fileData []byte) (string, error) {
	logger := log.WithContext("FileService", "ExtractPDF")
	start := time.Now()

	logger.Debug("Starting PDF extraction, size: %d bytes", len(fileData))

	// Try with unipdf first (more robust for complex PDFs)
	logger.Debug("Attempting PDF extraction with unipdf engine")
	text, err := f.extractPDFWithUnipdf(fileData)
	if err == nil && strings.TrimSpace(text) != "" {
		logger.Info("PDF extraction successful with unipdf, extracted %d characters", len(text))
		logger.WithDuration(start)
		return text, nil
	}
	logger.Warn("Unipdf extraction failed, trying fallback: %v", err)

	// Fallback to ledongthuc/pdf
	logger.Debug("Attempting PDF extraction with ledongthuc engine")
	text, err = f.extractPDFWithLedongthuc(fileData)
	if err == nil && strings.TrimSpace(text) != "" {
		logger.Info("PDF extraction successful with ledongthuc fallback, extracted %d characters", len(text))
		logger.WithDuration(start)
		return text, nil
	}
	logger.Error("Both PDF extraction engines failed: %v", err)

	// If both fail, return a meaningful error
	return "", fmt.Errorf("failed to extract text from PDF: both extraction methods failed")
}

func (f *fileService) extractPDFWithUnipdf(fileData []byte) (string, error) {
	logger := log.WithContext("FileService", "UnipdfEngine")

	reader := bytes.NewReader(fileData)
	pdfReader, err := model.NewPdfReader(reader)
	if err != nil {
		logger.Error("Failed to create unipdf reader: %v", err)
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Check if PDF is encrypted
	isEncrypted, err := pdfReader.IsEncrypted()
	if err != nil {
		logger.Error("Failed to check PDF encryption status: %v", err)
		return "", fmt.Errorf("failed to check encryption: %w", err)
	}

	if isEncrypted {
		logger.Debug("PDF is encrypted, attempting empty password decryption")
		// Try to decrypt with empty password
		success, err := pdfReader.Decrypt([]byte(""))
		if err != nil || !success {
			logger.Warn("Failed to decrypt encrypted PDF with empty password")
			return "", fmt.Errorf("PDF is encrypted and cannot be decrypted")
		}
		logger.Debug("Successfully decrypted PDF with empty password")
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		logger.Error("Failed to get PDF page count: %v", err)
		return "", fmt.Errorf("failed to get number of pages: %w", err)
	}

	logger.Debug("Processing PDF with %d pages", numPages)

	var textBuilder strings.Builder
	processedPages := 0

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page, err := pdfReader.GetPage(pageNum)
		if err != nil {
			logger.Debug("Skipping page %d due to read error: %v", pageNum, err)
			continue // Skip pages that can't be read
		}

		ex, err := extractor.New(page)
		if err != nil {
			logger.Debug("Skipping page %d due to extractor error: %v", pageNum, err)
			continue // Skip pages that can't be processed
		}

		pageText, err := ex.ExtractText()
		if err != nil {
			logger.Debug("Skipping page %d due to text extraction error: %v", pageNum, err)
			continue // Skip pages with extraction errors
		}

		if strings.TrimSpace(pageText) != "" {
			textBuilder.WriteString(pageText)
			textBuilder.WriteString("\n\n")
			processedPages++
		}
	}

	logger.Debug("Successfully processed %d/%d pages", processedPages, numPages)

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		logger.Warn("No text content extracted from any page")
		return "", fmt.Errorf("no text content found in PDF")
	}

	logger.Debug("Unipdf extraction completed, text length: %d characters", len(text))
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
