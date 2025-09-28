package filesrv

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

func (f *fileService) isValidFileType(ext string) bool {
	validTypes := []string{".txt", ".pdf", ".docx", ".doc"}
	return slices.Contains(validTypes, ext)
}

func (f *fileService) extractFromTxt(fileData []byte) (string, error) {
	return string(fileData), nil
}

func (f *fileService) extractFromPDF(fileData []byte) (string, error) {
	// Try UniPDF first (more robust)
	content, err := f.extractPDFWithUniPDF(fileData)
	if err == nil && content != "" {
		return content, nil
	}

	// Fallback to ledongthuc/pdf
	return f.extractPDFWithLedong(fileData)
}

func (f *fileService) extractPDFWithUniPDF(fileData []byte) (string, error) {
	reader := bytes.NewReader(fileData)
	pdfReader, err := model.NewPdfReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("failed to get number of pages: %w", err)
	}

	var textContent strings.Builder
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			continue // Skip problematic pages
		}

		ex, err := extractor.New(page)
		if err != nil {
			continue // Skip problematic pages
		}

		text, err := ex.ExtractText()
		if err != nil {
			continue // Skip problematic pages
		}

		textContent.WriteString(text)
		textContent.WriteString("\n")
	}

	return textContent.String(), nil
}

func (f *fileService) extractPDFWithLedong(fileData []byte) (string, error) {
	reader := bytes.NewReader(fileData)
	pdfReader, err := pdf.NewReader(reader, int64(len(fileData)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var textContent strings.Builder
	for i := 1; i <= pdfReader.NumPage(); i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue // Skip problematic pages
		}

		textContent.WriteString(text)
		textContent.WriteString("\n")
	}

	if textContent.Len() == 0 {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return textContent.String(), nil
}

func (f *fileService) extractFromDocx(fileData []byte) (string, error) {
	// For DOCX files, we'll implement a basic text extraction
	// This is a simplified implementation - for production use, consider libraries like:
	// - github.com/nguyenthenguyen/docx
	// - github.com/fumiama/go-docx

	// For now, return a note that DOCX parsing needs implementation
	return fmt.Sprintf(`DOCX File Content

Note: This is a DOCX file with %d bytes of data.
For full DOCX text extraction, please use a dedicated DOCX parsing library.

Basic file information extracted.
File appears to contain structured document content.

To implement full DOCX extraction:
1. Use github.com/nguyenthenguyen/docx for reading Word documents
2. Extract text, tables, and formatting
3. Handle embedded images and objects
4. Parse document structure and metadata

For now, treating as document file requiring manual processing.`, len(fileData)), nil
}
