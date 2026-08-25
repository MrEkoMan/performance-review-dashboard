package main

import (
	"testing"
)

// sampleDocument mirrors the manager's actual template: section headers, bulleted
// questions, and answers on the lines following each question.
const sampleDocument = `Career & Motivation:
	• What do you enjoy most about your current role?
		Building internal tooling that engineers actually adopt.
	• What type of work gives you energy?
		Greenfield work where the spec is loose.
	• What type of work drains your energy?
		Repeating status updates and ceremony meetings.
	• What skills are you hoping to develop this year?
		Distributed systems design and Go.
	• Where do you want your career to go over the next 2-3 years?
		Move toward staff engineer with platform focus.

Team & Organization:
	• What does this team do really well?
		Shipping reliably with strong test coverage.
	• What frustrates you about how the team operates?
		Decisions made in side channels without write-ups.
	• If you were Engineering Manager for a day, what would you change?
		Cut recurring meetings and protect focus time.
	• What is slowing us down?
		Long PR review queues.
	• Where do you see technical debt creating risk?
		The auth module has drifted from the current schema.

Individual Working Style:
	• How do you prefer feedback?
		Direct, written, with examples.
	• How often do you want coaching vs autonomy?
		Heavy autonomy, weekly check-ins.
	• What does a great manager look like to you?
		Someone who clears blockers and trusts the IC.
	• What has worked well with previous managers?
		Regular 1:1s that stayed consistent.
	• What hasn't worked?
		Micromanagement and shifting priorities weekly.

Current Work:
	• What projects are you most proud of?
		The review dashboard refactor.
	• What are you working on now?
		Evidence recency dashboard and review readiness.
	• What concerns you most about our roadmap?
		Too many parallel initiatives.
	• Where do you feel underutilized?
		I could be mentoring more juniors.

What is the one thing I should know about you that will help me be a better manager?
	I do my best thinking async and in writing.`

func TestParseOnboardingDocument_FullTemplate(t *testing.T) {
	got := parseOnboardingDocument(sampleDocument)

	check := func(name, want string, got string) {
		t.Helper()
		if got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	check("CareerMotivation.EnjoyMost", "Building internal tooling that engineers actually adopt.", got.CareerMotivation.EnjoyMost)
	check("CareerMotivation.EnergyGivers", "Greenfield work where the spec is loose.", got.CareerMotivation.EnergyGivers)
	check("CareerMotivation.EnergyDrainers", "Repeating status updates and ceremony meetings.", got.CareerMotivation.EnergyDrainers)
	check("CareerMotivation.SkillsThisYear", "Distributed systems design and Go.", got.CareerMotivation.SkillsThisYear)
	check("CareerMotivation.CareerNext2to3", "Move toward staff engineer with platform focus.", got.CareerMotivation.CareerNext2to3)
	check("TeamOrg.TeamDoesWell", "Shipping reliably with strong test coverage.", got.TeamOrg.TeamDoesWell)
	check("TeamOrg.Frustrations", "Decisions made in side channels without write-ups.", got.TeamOrg.Frustrations)
	check("TeamOrg.EMForADay", "Cut recurring meetings and protect focus time.", got.TeamOrg.EMForADay)
	check("TeamOrg.SlowingUsDown", "Long PR review queues.", got.TeamOrg.SlowingUsDown)
	check("TeamOrg.TechDebtRisk", "The auth module has drifted from the current schema.", got.TeamOrg.TechDebtRisk)
	check("WorkingStyle.PreferredFeedback", "Direct, written, with examples.", got.WorkingStyle.PreferredFeedback)
	check("WorkingStyle.CoachingVsAutonomy", "Heavy autonomy, weekly check-ins.", got.WorkingStyle.CoachingVsAutonomy)
	check("WorkingStyle.GreatManager", "Someone who clears blockers and trusts the IC.", got.WorkingStyle.GreatManager)
	check("WorkingStyle.WorkedWellWithPrev", "Regular 1:1s that stayed consistent.", got.WorkingStyle.WorkedWellWithPrev)
	check("WorkingStyle.HasntWorked", "Micromanagement and shifting priorities weekly.", got.WorkingStyle.HasntWorked)
	check("CurrentWork.ProudOf", "The review dashboard refactor.", got.CurrentWork.ProudOf)
	check("CurrentWork.WorkingOnNow", "Evidence recency dashboard and review readiness.", got.CurrentWork.WorkingOnNow)
	check("CurrentWork.RoadmapConcerns", "Too many parallel initiatives.", got.CurrentWork.RoadmapConcerns)
	check("CurrentWork.Underutilized", "I could be mentoring more juniors.", got.CurrentWork.Underutilized)
	check("OneThingToKnow", "I do my best thinking async and in writing.", got.OneThingToKnow)
}

func TestParseOnboardingDocument_InlineAnswers(t *testing.T) {
	doc := `Career & Motivation:
- What do you enjoy most about your current role? Building tools.
- What type of work gives you energy? Greenfield.`
	got := parseOnboardingDocument(doc)
	if got.CareerMotivation.EnjoyMost != "Building tools." {
		t.Errorf("EnjoyMost = %q", got.CareerMotivation.EnjoyMost)
	}
	if got.CareerMotivation.EnergyGivers != "Greenfield." {
		t.Errorf("EnergyGivers = %q", got.CareerMotivation.EnergyGivers)
	}
}

func TestParseOnboardingDocument_ParaphrasedQuestions(t *testing.T) {
	// Section header omitted; paraphrased wording; answer still matched by fragments.
	doc := `• What drains you at work?
	Meetings.
• Which projects are you proudest of?
	The dashboard.`
	got := parseOnboardingDocument(doc)
	if got.CareerMotivation.EnergyDrainers != "Meetings." {
		t.Errorf("EnergyDrainers = %q", got.CareerMotivation.EnergyDrainers)
	}
	if got.CurrentWork.ProudOf != "The dashboard." {
		t.Errorf("ProudOf = %q", got.CurrentWork.ProudOf)
	}
}

func TestParseOnboardingDocument_MarkdownHeaders(t *testing.T) {
	doc := `### Career & Motivation
**What do you enjoy most about your current role?**
Building tools.

## Team & Organization
**What frustrates you about how the team operates?**
Side-channel decisions.`
	got := parseOnboardingDocument(doc)
	if got.CareerMotivation.EnjoyMost != "Building tools." {
		t.Errorf("EnjoyMost = %q", got.CareerMotivation.EnjoyMost)
	}
	if got.TeamOrg.Frustrations != "Side-channel decisions." {
		t.Errorf("Frustrations = %q", got.TeamOrg.Frustrations)
	}
}

func TestParseOnboardingDocument_MissingSectionsAndEmpty(t *testing.T) {
	got := parseOnboardingDocument("")
	if got != (OnboardingAnswers{}) {
		t.Errorf("empty input should yield zero value, got %+v", got)
	}
	// A document with only one section should leave the others empty without error.
	got = parseOnboardingDocument(`Current Work:
• What projects are you most proud of?
	Only this.`)
	if got.CurrentWork.ProudOf != "Only this." {
		t.Errorf("ProudOf = %q", got.CurrentWork.ProudOf)
	}
	if got.CareerMotivation.EnjoyMost != "" {
		t.Errorf("EnjoyMost should be empty, got %q", got.CareerMotivation.EnjoyMost)
	}
}

func TestParseOnboardingDocument_UnmatchedQuestionDoesNotCrash(t *testing.T) {
	doc := `Career & Motivation:
• What is the meaning of life?
	Forty-two.`
	got := parseOnboardingDocument(doc)
	if got.CareerMotivation.EnjoyMost != "" {
		t.Errorf("EnjoyMost should be empty for an unmatched question, got %q", got.CareerMotivation.EnjoyMost)
	}
}

func TestParseOnboardingDocument_MultiLineAnswer(t *testing.T) {
	doc := `Career & Motivation:
• What do you enjoy most about your current role?
	First line of the answer.
	Second line continues here.`
	got := parseOnboardingDocument(doc)
	want := "First line of the answer. Second line continues here."
	if got.CareerMotivation.EnjoyMost != want {
		t.Errorf("EnjoyMost = %q, want %q", got.CareerMotivation.EnjoyMost, want)
	}
}
