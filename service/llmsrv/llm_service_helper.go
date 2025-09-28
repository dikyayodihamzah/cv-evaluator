package llmsrv

func (s *llmService) generateOverallSummary(cvEval *CVEvaluation, projectEval *ProjectEvaluation) string {
	totalScore := cvEval.MatchRate*10 + projectEval.Score

	if totalScore >= 15 {
		return "Excellent candidate fit. Strong technical background with proven project delivery capabilities."
	} else if totalScore >= 12 {
		return "Good candidate fit. Solid foundation with room for growth in specific areas."
	} else if totalScore >= 9 {
		return "Moderate candidate fit. Shows potential but would benefit from additional training and experience."
	}

	return "Limited candidate fit. Significant gaps in required skills and experience."
}

func (s *llmService) generateCVOnlySummary(cvEval *CVEvaluation) string {
	if cvEval.MatchRate >= 0.8 {
		return "Strong CV match for the position. Candidate demonstrates excellent alignment with job requirements based on background and experience. Project evaluation not available - upload project files for complete assessment."
	} else if cvEval.MatchRate >= 0.7 {
		return "Good CV match for the position. Candidate shows solid qualifications with minor gaps in some areas. Project evaluation not available - consider uploading project files to get a comprehensive evaluation."
	} else if cvEval.MatchRate >= 0.6 {
		return "Moderate CV match for the position. Candidate has relevant experience but may require additional training in specific areas. Project evaluation not available - project files would provide better insight into technical capabilities."
	} else {
		return "Limited CV match for the position. Significant gaps identified in required qualifications. Project evaluation not available - project files could potentially demonstrate practical skills not evident in the CV."
	}
}
