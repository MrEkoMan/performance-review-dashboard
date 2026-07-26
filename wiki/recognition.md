# Recognition

## Purpose

Recognition is stored as a first-class record so positive feedback, influence,
and accomplishments are not lost before review preparation.

## Fields

- Engineer
- Recognition date
- Source
- Source type
- Category
- Summary
- Details
- Related work
- Review cycle

## Source types

- Manager
- Peer
- Product
- Customer
- Leadership
- Cross-functional stakeholder
- External partner

## Categories

- Business impact
- Technical excellence
- Operational excellence
- Mentoring
- Collaboration
- Leadership
- Innovation
- Customer focus

## Engineer profile functionality

- Create, edit, and delete recognition records.
- Filter recognition by category.
- Show total recognition, recognition from the last 90 days, and distinct
  source totals.
- Display recognition totals on the profile overview.
- Preserve related project, incident, initiative, or evidence context.

## APIs

- `GET /api/engineers/{engineerId}/recognitions`
- `GET /api/engineers/{engineerId}/recognitions?category={category}`
- `GET /api/engineers/{engineerId}/recognitions?reviewCycle={reviewCycle}`
- `POST /api/engineers/{engineerId}/recognitions`
- `GET /api/recognitions/{id}`
- `PUT /api/recognitions/{id}`
- `DELETE /api/recognitions/{id}`

## Current boundary

Recognition is entered manually. Importing selected recognition from Slack or
Microsoft Teams remains an optional integration roadmap feature.
