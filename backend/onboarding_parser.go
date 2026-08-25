package main

import "strings"

// sectionKey identifies which of the five logical sections a parsed question belongs to.
type sectionKey int

const (
	sectionUnknown sectionKey = iota
	sectionCareerMotivation
	sectionTeamOrg
	sectionWorkingStyle
	sectionCurrentWork
	sectionOneThing
)

// parseOnboardingDocument extracts structured answers from a manager's initial 1:1 notes.
// It is intentionally tolerant: paraphrased questions, missing sections, and free-form
// formatting never cause an error — they simply yield empty fields the manager can fix
// in the form before saving. Parsing is best-effort and always human-reviewed upstream.
func parseOnboardingDocument(raw string) OnboardingAnswers {
	var out OnboardingAnswers
	if strings.TrimSpace(raw) == "" {
		return out
	}
	lines := strings.Split(raw, "\n")

	current := sectionUnknown
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}
		// A line containing '?' is a question, never a section header. Section headers
		// (e.g. "Career & Motivation:") do not contain '?', so detect them only on prose.
		if !strings.Contains(trimmed, "?") {
			if matched := detectSection(trimmed); matched != sectionUnknown {
				current = matched
			}
			// Non-question prose between questions is skipped here; multi-line answer
			// text is consumed inside extractAnswer instead.
			continue
		}
		key, ok := matchQuestion(trimmed, current)
		if !ok {
			continue
		}
		answer, consumed := extractAnswer(trimmed, lines, i+1)
		assignAnswer(&out, key, strings.TrimSpace(answer))
		i += consumed
	}
	return out
}

// detectSection recognizes section headers, ignoring leading markdown markers (#, **, :)
// and surrounding whitespace.
func detectSection(line string) sectionKey {
	cleaned := normalizeHeader(line)
	switch {
	case strings.Contains(cleaned, "career") && strings.Contains(cleaned, "motivation"):
		return sectionCareerMotivation
	case strings.Contains(cleaned, "team") && strings.Contains(cleaned, "organization"):
		return sectionTeamOrg
	case strings.Contains(cleaned, "individual") && strings.Contains(cleaned, "working style"):
		return sectionWorkingStyle
	case strings.Contains(cleaned, "current work"):
		return sectionCurrentWork
	case strings.Contains(cleaned, "one thing"):
		return sectionOneThing
	}
	return sectionUnknown
}

// normalizeHeader strips markdown emphasis and trailing colons/punctuation so headers
// like "### Career & Motivation:" or "**Team & Organization**" reduce to a plain phrase.
func normalizeHeader(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimLeft(s, "#")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ":")
	s = strings.TrimSpace(s)
	return strings.ToLower(s)
}

// matchQuestion maps a question line to a canonical field key. When the line lives in a
// known section, only that section's keys are candidates; the final "one thing" question
// is recognized in any section (it usually appears after all sections).
func matchQuestion(line string, section sectionKey) (answerKey, bool) {
	q := strings.ToLower(line)

	if strings.Contains(q, "one thing") {
		return keyOneThingToKnow, true
	}

	candidates := sectionKeys(section)
	for _, k := range candidates {
		if questionMatches(q, k) {
			return k, true
		}
	}
	// Fall back to scanning all keys if the section was not detected (parser tolerant of
	// documents that omit section headers entirely).
	if section == sectionUnknown {
		for _, k := range allQuestionKeys {
			if questionMatches(q, k) {
				return k, true
			}
		}
	}
	return 0, false
}

// answerKey enumerates the parseable answer fields.
type answerKey int

const (
	keyUnknown answerKey = iota
	keyEnjoyMost
	keyEnergyGivers
	keyEnergyDrainers
	keySkillsThisYear
	keyCareerNext2to3
	keyTeamDoesWell
	keyFrustrations
	keyEMForADay
	keySlowingUsDown
	keyTechDebtRisk
	keyPreferredFeedback
	keyCoachingVsAutonomy
	keyGreatManager
	keyWorkedWellWithPrev
	keyHasntWorked
	keyProudOf
	keyWorkingOnNow
	keyRoadmapConcerns
	keyUnderutilized
	keyOneThingToKnow
)

var allQuestionKeys = []answerKey{
	keyEnjoyMost, keyEnergyGivers, keyEnergyDrainers, keySkillsThisYear, keyCareerNext2to3,
	keyTeamDoesWell, keyFrustrations, keyEMForADay, keySlowingUsDown, keyTechDebtRisk,
	keyPreferredFeedback, keyCoachingVsAutonomy, keyGreatManager, keyWorkedWellWithPrev, keyHasntWorked,
	keyProudOf, keyWorkingOnNow, keyRoadmapConcerns, keyUnderutilized,
}

func sectionKeys(section sectionKey) []answerKey {
	switch section {
	case sectionCareerMotivation:
		return []answerKey{keyEnjoyMost, keyEnergyGivers, keyEnergyDrainers, keySkillsThisYear, keyCareerNext2to3}
	case sectionTeamOrg:
		return []answerKey{keyTeamDoesWell, keyFrustrations, keyEMForADay, keySlowingUsDown, keyTechDebtRisk}
	case sectionWorkingStyle:
		return []answerKey{keyPreferredFeedback, keyCoachingVsAutonomy, keyGreatManager, keyWorkedWellWithPrev, keyHasntWorked}
	case sectionCurrentWork:
		return []answerKey{keyProudOf, keyWorkingOnNow, keyRoadmapConcerns, keyUnderutilized}
	}
	return nil
}

// questionMatches applies fragment tests. Ordering within sectionKeys is most-specific
// first (e.g. "gives you energy" is tested before a looser "energy") so a single line
// cannot match the wrong field.
func questionMatches(q string, k answerKey) bool {
	switch k {
	case keyEnjoyMost:
		return strings.Contains(q, "enjoy most")
	case keyEnergyGivers:
		return strings.Contains(q, "gives you energy") || strings.Contains(q, "gives me energy")
	case keyEnergyDrainers:
		return strings.Contains(q, "drains")
	case keySkillsThisYear:
		return strings.Contains(q, "skills") && (strings.Contains(q, "develop") || strings.Contains(q, "year"))
	case keyCareerNext2to3:
		return strings.Contains(q, "2-3 years") || strings.Contains(q, "2–3 years") ||
			strings.Contains(q, "next 2-3") || strings.Contains(q, "next 2–3") ||
			(strings.Contains(q, "career") && strings.Contains(q, "next"))
	case keyTeamDoesWell:
		return strings.Contains(q, "do really well") || strings.Contains(q, "does really well") ||
			strings.Contains(q, "does well") || strings.Contains(q, "do well")
	case keyFrustrations:
		return strings.Contains(q, "frustrat")
	case keyEMForADay:
		return strings.Contains(q, "manager for a day") || strings.Contains(q, "engineering manager for a day")
	case keySlowingUsDown:
		return strings.Contains(q, "slowing us") || strings.Contains(q, "slowing")
	case keyTechDebtRisk:
		return strings.Contains(q, "technical debt") || strings.Contains(q, "tech debt")
	case keyPreferredFeedback:
		return strings.Contains(q, "prefer feedback") || strings.Contains(q, "prefer to receive feedback") ||
			strings.Contains(q, "how do you prefer feedback")
	case keyCoachingVsAutonomy:
		return strings.Contains(q, "coaching") && strings.Contains(q, "autonomy")
	case keyGreatManager:
		return strings.Contains(q, "great manager")
	case keyWorkedWellWithPrev:
		return strings.Contains(q, "worked well") && (strings.Contains(q, "previous") || strings.Contains(q, "managers"))
	case keyHasntWorked:
		return strings.Contains(q, "hasn't worked") || strings.Contains(q, "has not worked") || strings.Contains(q, "hasnt worked")
	case keyProudOf:
		return strings.Contains(q, "proud of") || strings.Contains(q, "proudest of")
	case keyWorkingOnNow:
		return strings.Contains(q, "working on now") || strings.Contains(q, "working on")
	case keyRoadmapConcerns:
		return strings.Contains(q, "roadmap")
	case keyUnderutilized:
		return strings.Contains(q, "underutilized") || strings.Contains(q, "under-utilized") || strings.Contains(q, "under utilized")
	}
	return false
}

// extractAnswer pulls the answer text for a question at lines[startIdx..]. If the question
// line carries an inline answer after the '?', that is used; otherwise following lines
// are consumed until the next question or section header. Returns the joined answer and
// the number of additional lines consumed.
func extractAnswer(questionLine string, lines []string, startIdx int) (string, int) {
	if idx := strings.Index(questionLine, "?"); idx >= 0 {
		rest := strings.TrimSpace(questionLine[idx+1:])
		rest = stripMarkdown(rest)
		rest = strings.TrimLeft(rest, "-: ")
		rest = strings.TrimSpace(rest)
		if rest != "" {
			return rest, 0
		}
	}
	var parts []string
	consumed := 0
	for j := startIdx; j < len(lines); j++ {
		line := lines[j]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Allow a single blank line within an answer; two blanks or a new question ends it.
			if len(parts) > 0 {
				break
			}
			continue
		}
		if strings.Contains(trimmed, "?") {
			break
		}
		if detectSection(trimmed) != sectionUnknown {
			break
		}
		parts = append(parts, trimmed)
		consumed++
	}
	return strings.Join(parts, " "), consumed
}

// stripMarkdown removes emphasis markers so an inline answer like "**Building tools.**"
// or "Building tools.**" reduces to the plain text.
func stripMarkdown(s string) string {
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// assignAnswer writes a parsed answer into the appropriate struct field.
func assignAnswer(a *OnboardingAnswers, k answerKey, v string) {
	switch k {
	case keyEnjoyMost:
		a.CareerMotivation.EnjoyMost = v
	case keyEnergyGivers:
		a.CareerMotivation.EnergyGivers = v
	case keyEnergyDrainers:
		a.CareerMotivation.EnergyDrainers = v
	case keySkillsThisYear:
		a.CareerMotivation.SkillsThisYear = v
	case keyCareerNext2to3:
		a.CareerMotivation.CareerNext2to3 = v
	case keyTeamDoesWell:
		a.TeamOrg.TeamDoesWell = v
	case keyFrustrations:
		a.TeamOrg.Frustrations = v
	case keyEMForADay:
		a.TeamOrg.EMForADay = v
	case keySlowingUsDown:
		a.TeamOrg.SlowingUsDown = v
	case keyTechDebtRisk:
		a.TeamOrg.TechDebtRisk = v
	case keyPreferredFeedback:
		a.WorkingStyle.PreferredFeedback = v
	case keyCoachingVsAutonomy:
		a.WorkingStyle.CoachingVsAutonomy = v
	case keyGreatManager:
		a.WorkingStyle.GreatManager = v
	case keyWorkedWellWithPrev:
		a.WorkingStyle.WorkedWellWithPrev = v
	case keyHasntWorked:
		a.WorkingStyle.HasntWorked = v
	case keyProudOf:
		a.CurrentWork.ProudOf = v
	case keyWorkingOnNow:
		a.CurrentWork.WorkingOnNow = v
	case keyRoadmapConcerns:
		a.CurrentWork.RoadmapConcerns = v
	case keyUnderutilized:
		a.CurrentWork.Underutilized = v
	case keyOneThingToKnow:
		a.OneThingToKnow = v
	}
}
