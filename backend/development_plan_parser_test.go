package main

import (
	"reflect"
	"testing"
)

// filledPlan mirrors the manager's filled-in development plan: a header
// metadata table, a goal workshop (strengths/growth, next-role gaps, focus
// areas), two SMART goals with metadata and monthly progress, and one
// accomplishment block.
const filledPlan = `# Personal Development Plan

| | |
|---|---|
| **Developer** | Ada Lovelace |
| **Current Role** | Senior Engineer |
| **Target Role** | Staff Engineer |
| **Plan Period** | FY 2026-2027 |
| **Manager** | Cassie Flint |

---

## How to Use This Document

Fill this in and bring it to 1:1s.

---

## Goal Workshop

### Step 1: Where Am I Now?

| Strengths (keep doing) | Growth Areas (develop) |
|---|---|
| _e.g. Strong debugging skills, reliable delivery_ | _e.g. System design, leading code reviews_ |
| Strong debugging skills | System design |
| Reliable delivery | Leading code reviews |

### Step 2: What Does the Next Role Require?

| What the next role looks like | Gap from where I am today |
|---|---|
| _e.g. Owns technical design for a feature end-to-end_ | _e.g. I contribute to designs but haven't led one_ |
| Owns technical design for a feature end-to-end | I contribute to designs but haven't led one |
| Mentors junior developers | I haven't done any formal mentoring |

### Step 3: Pick Your Focus Areas

1. Lead a technical design
2. Improve system design skills
3. Start mentoring

---

## SMART Goal Checklist

- [ ] **Specific**
- [ ] **Measurable**

---

## Goals

### Goal 1: Lead the authentication redesign

| | |
|---|---|
| **Goal** | Lead the end-to-end technical design of the auth redesign |
| **Why** | Closes the system design gap to staff |
| **Success Looks Like** | Design doc reviewed and adopted by the team |
| **Target Date** | September 2026 |

**Monthly Progress:**

| Month | Status | Update |
|---|---|---|
| Jun | Not Started |  |
| Jul | In Progress | Drafted the design doc |
| Aug | Complete | Design adopted |

---

### Goal 2: Start mentoring juniors

| | |
|---|---|
| **Goal** | Mentor two junior engineers through their first on-call rotation |
| **Why** | Closes the mentoring gap |
| **Success Looks Like** | Both complete a rotation without escalation |
| **Target Date** | December 2026 |

**Monthly Progress:**

| Month | Status | Update |
|---|---|---|
| Jun |  |  |
| Jul |  |  |

---

## Additional Accomplishments

### Accomplishment 1

> **Accomplishment:** Built a shared test fixture library for the team's integration tests
>
> **Problem:** Each developer wrote their own setup code, leading to flaky tests.
>
> **Value Delivered:** Reduced integration test setup code by ~40% across 3 services.
`

func TestParseDevelopmentPlan_FullTemplate(t *testing.T) {
	got := parseDevelopmentPlan(filledPlan)

	// Header metadata table.
	if got.Header.Developer != "Ada Lovelace" {
		t.Errorf("Developer = %q", got.Header.Developer)
	}
	if got.Header.CurrentRole != "Senior Engineer" {
		t.Errorf("CurrentRole = %q", got.Header.CurrentRole)
	}
	if got.Header.TargetRole != "Staff Engineer" {
		t.Errorf("TargetRole = %q", got.Header.TargetRole)
	}
	if got.Header.PlanPeriod != "FY 2026-2027" {
		t.Errorf("PlanPeriod = %q", got.Header.PlanPeriod)
	}
	if got.Header.Manager != "Cassie Flint" {
		t.Errorf("Manager = %q", got.Header.Manager)
	}

	// Strengths / growth areas: the example placeholder row must be skipped,
	// leaving two real rows each.
	wantStrengths := []string{"Strong debugging skills", "Reliable delivery"}
	if !reflect.DeepEqual(got.Strengths, wantStrengths) {
		t.Errorf("Strengths = %v, want %v", got.Strengths, wantStrengths)
	}
	wantGrowth := []string{"System design", "Leading code reviews"}
	if !reflect.DeepEqual(got.GrowthAreas, wantGrowth) {
		t.Errorf("GrowthAreas = %v, want %v", got.GrowthAreas, wantGrowth)
	}

	// Next-role gaps: placeholder row skipped, two real rows retained.
	if len(got.NextRole) != 2 {
		t.Fatalf("NextRole len = %d, want 2", len(got.NextRole))
	}
	if got.NextRole[0].NextRoleLooksLike != "Owns technical design for a feature end-to-end" {
		t.Errorf("NextRole[0].LooksLike = %q", got.NextRole[0].NextRoleLooksLike)
	}
	if got.NextRole[1].Gap != "I haven't done any formal mentoring" {
		t.Errorf("NextRole[1].Gap = %q", got.NextRole[1].Gap)
	}

	// Focus areas: three numbered items.
	wantFocus := []string{"Lead a technical design", "Improve system design skills", "Start mentoring"}
	if !reflect.DeepEqual(got.FocusAreas, wantFocus) {
		t.Errorf("FocusAreas = %v, want %v", got.FocusAreas, wantFocus)
	}

	// Two goals parsed.
	if len(got.Goals) != 2 {
		t.Fatalf("Goals len = %d, want 2", len(got.Goals))
	}
	g1 := got.Goals[0]
	if g1.Title != "Lead the authentication redesign" {
		t.Errorf("Goal1 Title = %q", g1.Title)
	}
	if g1.Goal != "Lead the end-to-end technical design of the auth redesign" {
		t.Errorf("Goal1 Goal = %q", g1.Goal)
	}
	if g1.Why != "Closes the system design gap to staff" {
		t.Errorf("Goal1 Why = %q", g1.Why)
	}
	if g1.SuccessLooksLike != "Design doc reviewed and adopted by the team" {
		t.Errorf("Goal1 Success = %q", g1.SuccessLooksLike)
	}
	if g1.TargetDate != "September 2026" {
		t.Errorf("Goal1 TargetDate = %q", g1.TargetDate)
	}
	if len(g1.MonthlyProgress) != 3 {
		t.Errorf("Goal1 MonthlyProgress len = %d, want 3", len(g1.MonthlyProgress))
	} else {
		if g1.MonthlyProgress[1].Status != "In Progress" {
			t.Errorf("Goal1 Jul Status = %q", g1.MonthlyProgress[1].Status)
		}
		if g1.MonthlyProgress[1].Update != "Drafted the design doc" {
			t.Errorf("Goal1 Jul Update = %q", g1.MonthlyProgress[1].Update)
		}
		if g1.MonthlyProgress[2].Status != "Complete" {
			t.Errorf("Goal1 Aug Status = %q", g1.MonthlyProgress[2].Status)
		}
	}

	// One accomplishment parsed.
	if len(got.Accomplishments) != 1 {
		t.Fatalf("Accomplishments len = %d, want 1", len(got.Accomplishments))
	}
	acc := got.Accomplishments[0]
	if acc.Label != "Accomplishment 1" {
		t.Errorf("Acc Label = %q", acc.Label)
	}
	if acc.Accomplishment != "Built a shared test fixture library for the team's integration tests" {
		t.Errorf("Acc Accomplishment = %q", acc.Accomplishment)
	}
	if acc.ValueDelivered != "Reduced integration test setup code by ~40% across 3 services." {
		t.Errorf("Acc Value = %q", acc.ValueDelivered)
	}
}

// TestParseDevelopmentPlan_BlankTemplate confirms the blank template parses
// without error and yields empty (not panicked-on) fields, since every cell is
// either empty or a placeholder.
func TestParseDevelopmentPlan_BlankTemplate(t *testing.T) {
	got := parseDevelopmentPlan(blankTemplate)
	if len(got.Strengths) != 0 {
		t.Errorf("blank Strengths = %v, want empty", got.Strengths)
	}
	if len(got.GrowthAreas) != 0 {
		t.Errorf("blank GrowthAreas = %v, want empty", got.GrowthAreas)
	}
	if len(got.NextRole) != 0 {
		t.Errorf("blank NextRole = %v, want empty", got.NextRole)
	}
	if len(got.FocusAreas) != 0 {
		t.Errorf("blank FocusAreas = %v, want empty", got.FocusAreas)
	}
	if len(got.Goals) != 5 {
		t.Errorf("blank Goals len = %d, want 5 (goal headings still detected)", len(got.Goals))
	}
	for i, g := range got.Goals {
		if g.Title != "" {
			t.Errorf("blank Goal[%d] Title = %q, want empty", i, g.Title)
		}
	}
	if len(got.Accomplishments) != 0 {
		t.Errorf("blank Accomplishments = %v, want empty", got.Accomplishments)
	}
}

// TestParseDevelopmentPlan_EmptyAndMissingSections confirms tolerant behavior
// on empty input and documents missing the workshop section.
func TestParseDevelopmentPlan_EmptyAndMissingSections(t *testing.T) {
	got := parseDevelopmentPlan("")
	if !reflect.DeepEqual(got, DevelopmentPlanFields{}) {
		t.Errorf("empty input should yield zero value, got %+v", got)
	}

	noWorkshop := `# Personal Development Plan

| | |
|---|---|
| **Developer** | Grace Hopper |

## Goals

### Goal 1: Ship the thing
| | |
|---|---|
| **Goal** | Ship it |
| **Target Date** | Soon |
`
	got = parseDevelopmentPlan(noWorkshop)
	if got.Header.Developer != "Grace Hopper" {
		t.Errorf("Developer = %q", got.Header.Developer)
	}
	if len(got.Goals) != 1 {
		t.Fatalf("Goals len = %d, want 1", len(got.Goals))
	}
	if got.Goals[0].Goal != "Ship it" {
		t.Errorf("Goal Goal = %q", got.Goals[0].Goal)
	}
	if got.Goals[0].TargetDate != "Soon" {
		t.Errorf("Goal TargetDate = %q", got.Goals[0].TargetDate)
	}
}

// blankTemplate is the unmodified template from the manager, used to verify
// the parser tolerates a fully blank upload.
const blankTemplate = `# Personal Development Plan

| | |
|---|---|
| **Developer** |  |
| **Current Role** |  |
| **Target Role** |  |
| **Plan Period** | FY 2026-2027 |
| **Manager** | Cassie Flint |

---

## How to Use This Document

This is **your** document.

1. **Identify** where you need to grow
2. **Set** 3–5 SMART goals

---

## Goal Workshop

### Step 1: Where Am I Now?

| Strengths (keep doing) | Growth Areas (develop) |
|---|---|
| _e.g. Strong debugging skills, reliable delivery_ | _e.g. System design, leading code reviews_ |
|  |  |
|  |  |

### Step 2: What Does the Next Role Require?

| What the next role looks like | Gap from where I am today |
|---|---|
| _e.g. Owns technical design for a feature end-to-end_ | _e.g. I contribute to designs but haven't led one_ |
|  |  |

### Step 3: Pick Your Focus Areas

1. _e.g. Lead a technical design_
2.
3.

---

## SMART Goal Checklist

- [ ] **Specific**

---

## Goals

### Goal 1: _Title_

| | |
|---|---|
| **Goal** |  |
| **Why** |  |
| **Success Looks Like** |  |
| **Target Date** |  |

**Monthly Progress:**

| Month | Status | Update |
|---|---|---|
| Jun |  |  |

---

### Goal 2: _Title_

| | |
|---|---|
| **Goal** |  |

---

### Goal 3: _Title_

| | |
|---|---|
| **Goal** |  |

---

### Goal 4: _Title_ (optional)

| | |
|---|---|
| **Goal** |  |

---

### Goal 5: _Title_ (optional)

| | |
|---|---|
| **Goal** |  |

---

## Additional Accomplishments

### Accomplishment 1

> **Accomplishment:**
>
> **Problem:**
>
> **Value Delivered:**
`
