package ragsrv

import (
	"math"
	"strings"
)

func (s *ragService) findBestMatchVector(docs []Document, queryVector []float32) *Document {
	if len(docs) == 0 {
		return nil
	}

	bestScore := -1.0 // Cosine similarity ranges from -1 to 1
	var bestDoc *Document

	for i := range docs {
		if len(docs[i].Vector) == 0 {
			continue // Skip documents without vectors
		}

		score := s.cosineSimilarity(queryVector, docs[i].Vector)
		if score > bestScore {
			bestScore = score
			bestDoc = &docs[i]
		}
	}

	// Fallback to first document if no good match found
	if bestDoc == nil && len(docs) > 0 {
		bestDoc = &docs[0]
	}

	return bestDoc
}

func (s *ragService) findBestMatchKeyword(docs []Document, query string) *Document {
	if len(docs) == 0 {
		return nil
	}

	queryWords := strings.Fields(strings.ToLower(query))
	bestScore := 0.0
	var bestDoc *Document

	for i := range docs {
		score := s.calculateKeywordSimilarity(&docs[i], queryWords)
		if score > bestScore {
			bestScore = score
			bestDoc = &docs[i]
		}
	}

	if bestDoc == nil && len(docs) > 0 {
		bestDoc = &docs[0] // Fallback to first document
	}

	return bestDoc
}

func (s *ragService) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *ragService) calculateKeywordSimilarity(doc *Document, queryWords []string) float64 {
	docContent := strings.ToLower(doc.Content)
	matches := 0

	for _, word := range queryWords {
		if strings.Contains(docContent, word) {
			matches++
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	return float64(matches) / float64(len(queryWords))
}
