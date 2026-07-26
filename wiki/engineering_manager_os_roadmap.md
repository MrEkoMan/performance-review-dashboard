# Engineering Manager OS Roadmap

## Vision

Evolve the application from a performance-evidence tracker into a local-first Engineering Manager operating system.

The platform should help an Engineering Manager:

* Prepare for and conduct effective 1:1s.
* Track goals, development plans, and follow-up items.
* Capture accomplishments, recognition, coaching themes, and operational contributions.
* Identify areas requiring attention before they become performance concerns.
* Prepare evidence-based performance reviews and promotion cases.
* Reduce reliance on memory and fragmented notes.
* Use AI to synthesize collected evidence without inventing unsupported conclusions.

The system should remain API-first, locally runnable, and designed to support future integrations with engineering and collaboration tools.

---

# Phase 2: Engineering Manager OS

## 1. Engineer Profiles

Each engineer should have a dedicated workspace containing:

* Name, role, level, and team.
* Review cycle.
* Career aspirations.
* Development plan.
* Skills and areas of interest.
* Performance evidence.
* Recognition history.
* Coaching and growth themes.
* Open follow-ups.
* Goals.
* 1:1 history.
* Projects and operational contributions.
* Review history.

### Desired Outcomes

* Provide a single source of truth for each engineer.
* Make 1:1 and review preparation faster.
* Present performance evidence in both tabular and chronological formats.

---

## 2. Goals and Development Plans

Create structured goals linked to individual engineers.

### Goal Fields

* Title.
* Description.
* Goal type.
* Status.
* Priority.
* Start date.
* Target date.
* Completion date.
* Progress percentage.
* Success criteria.
* Manager notes.
* Engineer notes.
* Review cycle.

### Goal Types

* Delivery.
* Technical growth.
* Leadership.
* Communication.
* Operational excellence.
* Mentoring.
* Career development.
* Stretch assignment.

### Desired Outcomes

* Track forward-looking development instead of only historical evidence.
* Surface goals that are overdue, blocked, or at risk.
* Connect completed goals to performance reviews.

---

## 3. 1:1 Management

Add recurring 1:1 records linked to each engineer.

### 1:1 Fields

* Meeting date.
* Wins.
* Challenges.
* Career discussion.
* Feedback.
* Manager topics.
* Engineer topics.
* Action items.
* Follow-up date.
* Private manager notes.
* Shared notes.
* Meeting status.

### Desired Outcomes

* Preserve continuity between conversations.
* Track recurring themes.
* Prevent action items from being forgotten.
* Provide context for coaching and career development.

---

## 4. Follow-Up Management

Treat follow-ups as structured action items rather than only a boolean flag.

### Follow-Up Fields

* Engineer.
* Source record.
* Description.
* Owner.
* Due date.
* Status.
* Priority.
* Completion date.
* Notes.

### Desired Outcomes

* Surface overdue actions.
* Show outstanding commitments by engineer.
* Provide a manager-focused “Needs Attention” dashboard.

---

## 5. Recognition

Add recognition as a first-class entity.

### Recognition Sources

* Manager.
* Peer.
* Product.
* Customer.
* Leadership.
* Cross-functional stakeholder.
* External partner.

### Recognition Categories

* Business impact.
* Technical excellence.
* Operational excellence.
* Mentoring.
* Collaboration.
* Leadership.
* Innovation.
* Customer focus.

### Desired Outcomes

* Prevent positive feedback from being lost in email or chat.
* Ensure reviews include balanced evidence.
* Identify recurring strengths and organizational influence.

---

## 6. Engineer Timeline

Provide a chronological view of:

* Performance notes.
* Recognition.
* Goals.
* Goal progress.
* 1:1 meetings.
* Incidents.
* Project milestones.
* Coaching conversations.
* Promotions and role changes.
* Review outcomes.

### Desired Outcomes

* Make review preparation easier than scanning individual tables.
* Show growth and impact over time.
* Provide context around related events.

---

## 7. Dashboard Intelligence

Transform the dashboard from a record repository into an action-oriented workspace.

### Suggested Widgets

* Open follow-ups.
* Overdue follow-ups.
* Engineers without recent evidence.
* Upcoming 1:1s.
* Overdue goals.
* Goals at risk.
* Recent recognition.
* Growth areas requiring follow-up.
* Review cycles approaching completion.
* Engineers with increasing scope.
* Engineers who may be ready for additional responsibility.

### Desired Outcomes

The dashboard should answer:

* Who needs attention?
* What commitments are overdue?
* Who has not received recent feedback or recognition?
* What should I prepare before my next 1:1?
* Which engineers are demonstrating expanded scope?

---

## 8. Review Management

Add structured review periods and review records.

### Review Fields

* Engineer.
* Review cycle.
* Review type.
* Status.
* Performance summary.
* Major accomplishments.
* Strengths.
* Growth opportunities.
* Goal outcomes.
* Scope and responsibility changes.
* Manager recommendation.
* Calibration notes.
* Final rating.
* Submission date.

### Review Types

* Quarterly check-in.
* Mid-year review.
* Annual review.
* Promotion review.
* Performance improvement review.

### Desired Outcomes

* Build reviews continuously rather than at the deadline.
* Link every conclusion to supporting evidence.
* Preserve previous review history.

---

## 9. Promotion Readiness

Track promotion readiness without reducing it to a single unsupported score.

### Promotion Evidence

* Demonstrated next-level scope.
* Sustained impact.
* Technical leadership.
* Organizational influence.
* Mentoring.
* Operational ownership.
* Cross-team collaboration.
* Stakeholder feedback.
* Repeated performance over time.

### Desired Outcomes

* Create evidence-based promotion discussions.
* Identify missing evidence early.
* Generate promotion packets from verified records.

---

# Integrations

## Initial Integrations

Add locally configured integrations for:

* GitHub.
* Jira.
* Slack.
* Microsoft Teams.

Credentials should be encrypted before being stored in SQLite.

### GitHub

Potential capabilities:

* Pull request activity.
* Review activity.
* Repository contributions.
* Issue participation.
* Release activity.
* Team and repository ownership.

### Jira

Potential capabilities:

* Assigned work.
* Completed work.
* Delivery trends.
* Initiative participation.
* Cycle-time context.
* Blocked or overdue work.

### Slack

Potential capabilities:

* Import selected recognition messages.
* Capture manually selected feedback.
* Link discussion references to performance evidence.
* Avoid indiscriminate collection of private conversations.

### Microsoft Teams

Potential capabilities:

* Import selected recognition or feedback.
* Link meeting references.
* Support webhook or Microsoft Graph integration.
* Avoid broad surveillance or automatic employee scoring.

### Integration Principles

* Require explicit user configuration.
* Use least-privilege credentials.
* Store credentials encrypted.
* Make integrations optional.
* Allow credentials to be revoked.
* Provide connection tests.
* Show the last successful synchronization.
* Never infer performance solely from activity volume.
* Avoid treating commits, messages, or ticket counts as direct measures of productivity.

---

# AI Capabilities

## AI Design Principles

AI output must be:

* Grounded in stored evidence.
* Traceable to source records.
* Clearly identified as generated content.
* Reviewable and editable by the manager.
* Balanced across accomplishments and growth areas.
* Protected from unsupported conclusions.
* Designed to assist judgment, not replace it.

AI should not:

* Invent accomplishments.
* Infer sensitive personal characteristics.
* diagnose health, burnout, or motivation.
* rank engineers based only on activity counts.
* make autonomous employment decisions.
* generate final ratings without manager review.

---

## 1. Evidence Summaries

Example requests:

* Summarize this engineer’s quarter.
* Summarize the current review cycle.
* Identify the strongest examples of business impact.
* Summarize operational contributions.
* Identify recurring coaching themes.
* Show evidence of increased scope.

The response should link each statement to supporting records.

---

## 2. Performance Review Drafting

Generate editable drafts containing:

* Overall performance summary.
* Major accomplishments.
* Technical contributions.
* Business impact.
* Operational excellence.
* Team contribution.
* Recognition.
* Growth opportunities.
* Goal outcomes.
* Suggested focus areas for the next cycle.

Each section should cite the underlying evidence used.

---

## 3. Promotion Packet Drafting

Generate a draft promotion narrative based on:

* Sustained performance.
* Demonstrated next-level behaviors.
* Scope and complexity.
* Cross-team influence.
* Mentoring.
* Technical leadership.
* Operational ownership.
* Stakeholder recognition.

The system should also identify evidence gaps.

Example:

> Strong evidence exists for technical leadership and operational ownership. Additional evidence may be needed for sustained cross-team influence.

---

## 4. 1:1 Preparation

Before a 1:1, generate:

* Recent accomplishments.
* Open follow-ups.
* Goals requiring discussion.
* Previous action items.
* Recent recognition.
* Recurring topics.
* Suggested questions.

Example questions:

* What progress has been made on the architecture leadership goal?
* Is the current on-call workload sustainable?
* What additional support would help complete the current initiative?
* Which recent work felt most meaningful?

---

## 5. Coaching Support

Use documented patterns to suggest coaching prompts.

Examples:

* Several notes mention delegation challenges.
* Stakeholder communication has appeared in multiple growth discussions.
* The engineer has repeatedly expressed interest in architecture work.
* Follow-up actions have remained open across several 1:1s.

Suggestions must remain advisory and evidence-based.

---

## 6. Recognition Assistance

Identify positive contributions that may deserve recognition.

Examples:

* Repeated mentoring contributions.
* Incident leadership.
* Cross-team support.
* Reliability improvements.
* Successful completion of a stretch assignment.

The manager should approve any recognition before it is shared.

---

## 7. Evidence Gap Analysis

Help identify where review conclusions lack sufficient supporting evidence.

Examples:

* No recent evidence exists for collaboration.
* The review contains a technical leadership claim but no linked examples.
* Most evidence comes from one project.
* No stakeholder feedback has been recorded.
* Growth feedback has not been followed by documented progress.

---

## 8. Natural-Language Queries

Allow managers to ask questions such as:

* What were this engineer’s most impactful accomplishments?
* What follow-ups remain open?
* What evidence supports promotion readiness?
* What themes have appeared in recent 1:1s?
* Who has not received recognition recently?
* Which goals are at risk?
* What changed since the last review?

---

# AI Architecture

## Recommended Approach

Use retrieval-augmented generation rather than training a model on employee records.

### Flow

1. The user selects an engineer, review cycle, or question.
2. The application retrieves relevant records from SQLite.
3. Records are converted into a structured evidence package.
4. The package is sent to the configured model.
5. The model produces a draft with source references.
6. The manager reviews and edits the result.
7. Generated content is not saved as final review content until explicitly approved.

### Benefits

* Better traceability.
* Lower hallucination risk.
* Easier deletion and correction.
* No requirement to fine-tune a model on sensitive records.
* Supports local or hosted language models.

---

# Privacy and Security

Performance information and integration credentials are sensitive.

## Requirements

* Keep the application local by default.
* Encrypt integration credentials before storing them.
* Keep encryption keys outside SQLite.
* Do not commit databases, credentials, or encryption keys to source control.
* Add `.env` and database files to `.gitignore`.
* Use least-privilege integration tokens.
* Support credential removal and rotation.
* Avoid broad collection of private messages.
* Provide clear data-retention controls.
* Require explicit manager review before saving AI-generated review content.
* Maintain an audit trail for generated summaries and review changes.

---

# Suggested Delivery Sequence

## Phase 2A: Manager Workflow

1. Engineer profiles.
2. Goals.
3. 1:1 history.
4. Structured follow-ups.
5. Recognition.
6. Timeline view.

## Phase 2B: Operational Dashboard

1. Needs Attention panel.
2. Upcoming 1:1s.
3. Overdue follow-ups.
4. Goal status.
5. Evidence recency.
6. Review-cycle readiness.

## Phase 2C: Integrations

1. Integration settings.
2. Encrypted credential storage.
3. Connection tests.
4. GitHub import.
5. Jira import.
6. Slack recognition import.
7. Microsoft Teams integration.
8. Synchronization history.

## Phase 2D: AI Assistance

1. Evidence summaries.
2. 1:1 preparation.
3. Quarterly summaries.
4. Review drafting.
5. Evidence gap analysis.
6. Promotion packet drafting.
7. Natural-language querying.
8. Local-model support.

---

# Success Criteria

The Engineering Manager OS is successful when it:

* Reduces time required to prepare for 1:1s and reviews.
* Improves the quality and specificity of performance feedback.
* Ensures accomplishments and recognition are not forgotten.
* Surfaces overdue commitments and missing evidence.
* Produces review drafts grounded in verified records.
* Supports managerial judgment without automating employment decisions.
* Remains secure, local-first, and maintainable.
