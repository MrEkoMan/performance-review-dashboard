# Goals and Development Plans

## Overview

Goals are structured, forward-looking records associated with an engineer.
They complement historical performance evidence and provide the foundation for
1:1 preparation, overdue-work dashboards, review outcomes, and later AI
assistance.

Goals are managed from the engineer profile and are linked to the engineer
through a SQLite foreign key.

## Goal fields

- Title
- Description
- Goal type
- Status
- Priority
- Start date
- Target date
- Completion date
- Progress percentage
- Success criteria
- Manager notes
- Engineer notes
- Review cycle

## Goal lifecycle

Supported statuses:

- Not started
- In progress
- Blocked
- Completed
- Cancelled

Completed goals require 100 percent progress and a completion date. Active
goals with a target date in the past are highlighted as overdue in the
engineer profile.

## Profile functionality

- Create, edit, and delete goals.
- Filter goals by status.
- View active, blocked, and overdue counts.
- Track progress with a percentage and progress bar.
- Highlight blocked and overdue goals.
- Associate a goal with a review cycle.

The API additionally supports list filtering by status and review cycle.

## Goal types

- Delivery
- Technical growth
- Leadership
- Communication
- Operational excellence
- Mentoring
- Career development
- Stretch assignment

## API

- `GET /api/engineers/{engineerId}/goals`
- `POST /api/engineers/{engineerId}/goals`
- `GET /api/goals/{id}`
- `PUT /api/goals/{id}`
- `DELETE /api/goals/{id}`

The list endpoint supports optional `status` and `reviewCycle` query
parameters.

## Local data protection

SQLite database files are ignored by Git. The existing
`backend/data/performance.db` is retained locally but removed from version
control.
