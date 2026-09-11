package main

import (
	"regexp"
	"strings"
)

// parseDevelopmentPlan extracts structured fields from a personal development
// plan written in the manager's markdown template. The template is built around
// markdown tables (a header metadata table, goal tables, monthly-progress
// tables) plus blockquote accomplishment entries. The parser is intentionally
// tolerant: paraphrased headings, missing sections, and the blank template's
// `_e.g. …` placeholder rows never cause an error — they simply yield empty
// fields the manager can fix in the form before saving. Parsing is best-effort
// and always human-reviewed upstream.
func parseDevelopmentPlan(raw string) DevelopmentPlanFields {
	var out DevelopmentPlanFields
	if strings.TrimSpace(raw) == "" {
		return out
	}
	lines := strings.Split(raw, "\n")

	// Track which top-level (##) and sub (###) section we are inside so table
	// rows and list items route to the right field.
	currentH2 := ""
	currentH3 := ""

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Headings take precedence over table/list detection. normalizeHeader
		// strips leading '#', emphasis, and trailing ':' so "### Step 1: Where
		// Am I Now?" reduces to a comparable phrase.
		if strings.HasPrefix(trimmed, "#") {
			level := headingLevel(trimmed)
			heading := normalizeDevHeading(trimmed)
			if level == 2 {
				currentH2 = heading
				currentH3 = ""
			} else if level >= 3 {
				currentH3 = heading
				// A goal heading ("### Goal N: …") under the Goals section starts a
				// new goal entry that the following tables attach to.
				if currentH2 == "goals" {
					if goal := matchGoalHeading(trimmed); goal != nil {
						out.Goals = append(out.Goals, *goal)
					}
				}
				// An accomplishment heading ("### Accomplishment N") under the
				// Additional Accomplishments section starts a new entry whose
				// blockquote body follows on the next lines.
				if currentH2 == "additional accomplishments" && matchAccomplishmentHeading(trimmed) {
					block, next := readBlockquote(lines, i+1)
					i = next - 1
					acc := parseAccomplishmentBlock(block)
					acc.Label = stripAccomplishmentLabel(trimmed)
					if !acc.isEmpty() {
						out.Accomplishments = append(out.Accomplishments, acc)
					}
				}
			}
			continue
		}

		// The first table in the document (before any ## section, or inside the
		// title block) is the identifying header metadata table.
		if isTableRow(trimmed) && currentH2 == "" {
			rows, next := readTable(lines, i)
			i = next - 1
			parseHeaderTable(&out, rows)
			continue
		}

		switch currentH2 {
		case "goal workshop":
			if isTableRow(trimmed) {
				rows, next := readTable(lines, i)
				i = next - 1
				switch currentH3 {
				case "step 1: where am i now?", "step 1 where am i now", "step 1":
					parseStrengthsGrowth(&out, rows)
				case "step 2: what does the next role require?", "step 2 what does the next role require", "step 2":
					parseNextRoleGaps(&out, rows)
				}
				continue
			}
			if currentH3 == "step 3: pick your focus areas" || currentH3 == "step 3 pick your focus areas" || currentH3 == "step 3" {
				if focus := parseListItem(trimmed); focus != "" {
					out.FocusAreas = append(out.FocusAreas, focus)
				}
				continue
			}

		case "goals":
			// A goal table begins immediately under a "### Goal N" heading. We
			// attach the table to the most recently started goal.
			if isTableRow(trimmed) && len(out.Goals) > 0 {
				rows, next := readTable(lines, i)
				i = next - 1
				parseGoalTable(&out.Goals[len(out.Goals)-1], rows)
				// A monthly-progress table, if present, follows the goal table
				// after a "**Monthly Progress:**" label line.
				continue
			}
			if isMonthlyProgressStart(trimmed) && len(out.Goals) > 0 {
				// Advance to the next table row; read it as the progress table.
				j := i + 1
				for j < len(lines) && !isTableRow(strings.TrimSpace(lines[j])) {
					j++
				}
				if j < len(lines) && isTableRow(strings.TrimSpace(lines[j])) {
					rows, next := readTable(lines, j)
					i = next - 1
					parseMonthlyProgress(&out.Goals[len(out.Goals)-1], rows)
				}
				continue
			}

		case "additional accomplishments":
			// Accomplishment headings and their blockquotes are handled in the
			// heading branch above; nothing to do here for body lines.
		}
	}
	return out
}

// headingLevel returns the count of leading '#' characters on a heading line.
func headingLevel(line string) int {
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	return level
}

// normalizeDevHeading strips markdown emphasis, trailing colons, and lowercases
// a heading so section matching is robust to "### Step 1: Where Am I Now?" vs
// "## Step 1 Where Am I Now".
func normalizeDevHeading(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimLeft(s, "#")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.TrimSpace(s)
	// Drop everything from the first ':' onward? No — step headings use ':' as
	// a separator before the descriptive title, but the title after it is what
	// we want to keep here is just the leading "Step N" token for step 3, and
	// the full phrase for steps 1/2. Keep the whole phrase minus trailing ':'.
	s = strings.TrimSuffix(s, ":")
	s = strings.TrimSpace(s)
	return strings.ToLower(s)
}

// isTableRow reports whether a line looks like a markdown table row. A table
// row begins and ends with '|' and contains at least one cell separator.
func isTableRow(line string) bool {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "|") || !strings.HasSuffix(s, "|") {
		return false
	}
	return strings.Contains(s, "|")
}

// isSeparatorRow reports whether a table row is a column separator (e.g.
// "|---|---|"), which carries no data and is skipped during parsing. Each
// cell must be composed solely of dashes and colons (the markdown table
// separator syntax).
func isSeparatorRow(line string) bool {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "|") {
		return false
	}
	cells := splitTableRow(s)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		c = strings.TrimSpace(c)
		if c == "" {
			return false
		}
		for _, r := range c {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

// splitTableRow splits a markdown table row into trimmed cell values.
func splitTableRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	parts := strings.Split(s, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// readTable reads consecutive table rows starting at lines[start], skipping the
// header and separator rows, and returns the data rows plus the index of the
// first line after the table.
func readTable(lines []string, start int) ([][]string, int) {
	var rows [][]string
	i := start
	first := true
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if !isTableRow(trimmed) {
			break
		}
		if isSeparatorRow(trimmed) {
			i++
			first = false
			continue
		}
		// Skip the header row (the first non-separator row of the table).
		if first {
			first = false
			i++
			continue
		}
		rows = append(rows, splitTableRow(trimmed))
		i++
	}
	return rows, i
}

// isPlaceholder reports whether a cell is a blank-template placeholder
// ("_e.g. …_" or empty). Such rows are examples in the template and are not
// real plan content.
func isPlaceholder(cell string) bool {
	cell = strings.TrimSpace(cell)
	if cell == "" {
		return true
	}
	// Italics placeholders like "_e.g. Strong debugging skills …_".
	lowered := strings.ToLower(cell)
	if strings.HasPrefix(lowered, "_e.g.") || strings.HasPrefix(lowered, "e.g.") {
		return true
	}
	return false
}

// parseHeaderTable maps the header metadata table (Developer / Current Role /
// Target Role / Plan Period / Manager) onto the Header struct.
func parseHeaderTable(out *DevelopmentPlanFields, rows [][]string) {
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		label := strings.ToLower(stripEmphasis(row[0]))
		value := cleanValue(row[1])
		switch {
		case strings.Contains(label, "developer"):
			out.Header.Developer = value
		case strings.Contains(label, "current role"):
			out.Header.CurrentRole = value
		case strings.Contains(label, "target role"):
			out.Header.TargetRole = value
		case strings.Contains(label, "plan period"):
			out.Header.PlanPeriod = value
		case strings.Contains(label, "manager"):
			out.Header.Manager = value
		}
	}
}

// parseStrengthsGrowth reads the two-column strengths/growth-areas table. Each
// row contributes one strength and one growth area; placeholder rows are
// skipped.
func parseStrengthsGrowth(out *DevelopmentPlanFields, rows [][]string) {
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		strength := cleanValue(row[0])
		growth := cleanValue(row[1])
		if !isPlaceholder(row[0]) && strength != "" {
			out.Strengths = append(out.Strengths, strength)
		}
		if !isPlaceholder(row[1]) && growth != "" {
			out.GrowthAreas = append(out.GrowthAreas, growth)
		}
	}
}

// parseNextRoleGaps reads the two-column "what the next role looks like" /
// "gap" table.
func parseNextRoleGaps(out *DevelopmentPlanFields, rows [][]string) {
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		looksLike := cleanValue(row[0])
		gap := cleanValue(row[1])
		if isPlaceholder(row[0]) && isPlaceholder(row[1]) {
			continue
		}
		out.NextRole = append(out.NextRole, DevelopmentPlanGap{
			NextRoleLooksLike: looksLike,
			Gap:               gap,
		})
	}
}

// parseGoalTable maps a single goal's metadata table onto the goal struct.
func parseGoalTable(goal *DevelopmentPlanGoal, rows [][]string) {
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		label := strings.ToLower(stripEmphasis(row[0]))
		value := cleanValue(row[1])
		switch {
		case strings.Contains(label, "success looks like"):
			goal.SuccessLooksLike = value
		case strings.Contains(label, "target date"):
			goal.TargetDate = value
		case strings.Contains(label, "why"):
			goal.Why = value
		case strings.Contains(label, "goal"):
			goal.Goal = value
		}
	}
}

// parseMonthlyProgress reads a goal's monthly-progress table.
func parseMonthlyProgress(goal *DevelopmentPlanGoal, rows [][]string) {
	for _, row := range rows {
		if len(row) < 3 {
			continue
		}
		month := cleanValue(row[0])
		status := cleanValue(row[1])
		update := cleanValue(row[2])
		// The template's status hint line ("Not Started / In Progress /
		// Complete") lives in a blockquote, not the table, so empty status rows
		// are still meaningful (an unstarted month).
		goal.MonthlyProgress = append(goal.MonthlyProgress, DevelopmentPlanMonth{
			Month: month, Status: status, Update: update,
		})
	}
}

// matchGoalHeading recognizes "### Goal N: _Title_" and returns a goal shell
// with the title populated (or empty if the title is a placeholder).
var goalHeadingRe = regexp.MustCompile(`(?i)^#{2,4}\s*goal\s+\d+\s*:?\s*(.*)$`)

func matchGoalHeading(line string) *DevelopmentPlanGoal {
	m := goalHeadingRe.FindStringSubmatch(strings.TrimSpace(line))
	if m == nil {
		return nil
	}
	title := cleanValue(m[1])
	if strings.EqualFold(title, "title") || strings.EqualFold(title, "title (optional)") {
		title = ""
	}
	return &DevelopmentPlanGoal{Title: title}
}

// isMonthlyProgressStart recognizes the "**Monthly Progress:**" label line
// that precedes a goal's monthly-progress table.
func isMonthlyProgressStart(line string) bool {
	l := strings.ToLower(stripEmphasis(strings.TrimSpace(line)))
	return strings.Contains(l, "monthly progress")
}

// matchAccomplishmentHeading recognizes "### Accomplishment N" (but not the
// "### Example" sample heading).
var accomplishmentHeadingRe = regexp.MustCompile(`(?i)^#{2,4}\s*accomplishment\b`)

func matchAccomplishmentHeading(line string) bool {
	return accomplishmentHeadingRe.MatchString(strings.TrimSpace(line))
}

// stripAccomplishmentLabel returns the trailing label (e.g. "Accomplishment 1")
// from an accomplishment heading for display.
func stripAccomplishmentLabel(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimLeft(s, "#")
	s = strings.TrimSpace(s)
	return s
}

// readBlockquote collects consecutive '>' lines starting at lines[start],
// returning the joined block text and the index after the block.
func readBlockquote(lines []string, start int) (string, int) {
	var parts []string
	i := start
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			// A blank line ends the blockquote.
			if len(parts) > 0 {
				break
			}
			i++
			continue
		}
		if !strings.HasPrefix(trimmed, ">") {
			break
		}
		body := strings.TrimPrefix(trimmed, ">")
		body = strings.TrimSpace(body)
		if body != "" {
			parts = append(parts, body)
		}
		i++
	}
	return strings.Join(parts, "\n"), i
}

// parseAccomplishmentBlock extracts the Accomplishment / Problem / Value
// Delivered fields from a blockquote block.
func parseAccomplishmentBlock(block string) DevelopmentPlanAccomplishment {
	var acc DevelopmentPlanAccomplishment
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		label, value := splitLabelValue(line)
		switch label {
		case "accomplishment":
			acc.Accomplishment = value
		case "problem":
			acc.Problem = value
		case "value delivered":
			acc.ValueDelivered = value
		}
	}
	return acc
}

// splitLabelValue splits a "label: value" line (after emphasis stripping),
// returning the lowercased label and the cleaned value.
func splitLabelValue(line string) (string, string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", ""
	}
	label := strings.ToLower(stripEmphasis(line[:idx]))
	value := cleanValue(line[idx+1:])
	return strings.TrimSpace(label), value
}

// parseListItem extracts the text of a numbered list item ("1. …") or a
// bulleted list item, skipping placeholders.
func parseListItem(line string) string {
	s := strings.TrimSpace(line)
	var rest string
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") {
		rest = s[2:]
	} else {
		// Numbered list: "1. …"
		dot := strings.Index(s, ". ")
		if dot < 0 {
			return ""
		}
		rest = s[dot+2:]
	}
	value := cleanValue(rest)
	if isPlaceholder(rest) {
		return ""
	}
	return value
}

// stripEmphasis removes markdown emphasis markers so labels like "**Goal**"
// reduce to "Goal".
func stripEmphasis(s string) string {
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "`", "")
	return s
}

// cleanValue strips surrounding emphasis and trims a cell's value.
func cleanValue(s string) string {
	return strings.TrimSpace(stripEmphasis(s))
}

// isEmpty reports whether an accomplishment entry has no filled fields.
func (a DevelopmentPlanAccomplishment) isEmpty() bool {
	return a.Accomplishment == "" && a.Problem == "" && a.ValueDelivered == ""
}
