# Upcoming 1:1s Dashboard

## Purpose

The Upcoming 1:1s widget provides preparation context for scheduled
conversations across the team rather than showing only a calendar reminder.

## Meeting window

The dashboard defaults to the next 14 days and supports 7-, 14-, and 30-day
windows. Scheduled meetings with dates in the past remain visible as overdue
until their status is updated.

## Preparation context

Each meeting displays:

- Engineer and meeting date
- Days until the meeting, or days overdue
- Most recent completed 1:1 before the scheduled meeting
- Open structured follow-ups
- Blocked goals
- Overdue active goals
- Performance evidence from the last 30 days
- Recognition from the last 90 days

The Prepare action opens the engineer's 1:1 tab.

## API

`GET /api/dashboard/upcoming-one-on-ones`

The optional `days` query parameter accepts a value from 1 through 90 and
defaults to 14.

## Interpretation boundary

Preparation counts provide conversation context. They are not employee scores,
and a zero evidence or recognition count indicates only that no corresponding
record was entered during the configured lookback period.
