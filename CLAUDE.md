# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TaskLedger is a CLI tool for tracking work and generating reports from YAML log files. It's a single-binary Go application built with Cobra that parses `worklog.yml` files to calculate hours worked and generate status reports in text and HTML formats.

## Build and Development Commands

```bash
# Build the binary
make build            # Creates ./bin/taskledger

# Run tests
make test            # Run all tests with verbose output

# Clean build artifacts
make clean           # Remove bin/ directory and clean Go cache

# Run the application (after building)
make run ARGS="report --start-date=2024-07-26"

# Tidy dependencies
make tidy            # Run go mod tidy
```

## Testing

```bash
# Run all tests
go test -v ./...

# Run tests in specific package
go test -v ./cmd

# Run a single test
go test -v ./cmd -run TestReportCommand
```

## Code Architecture

### Single-File Architecture

The entire application is contained in `cmd/main.go` (~950 lines). This is intentional for simplicity - there are no internal packages or complex module structure.

### Core Components

1. **Data Structures** (lines 22-53):
   - `WorkLog`: Time entries with start/end times
   - `Task`: Work items with status, description, JIRA ticket, PR links, blockers
   - `DailyLog`: Combines work logs and tasks for a single date
   - `WorkData`: Top-level map of date strings to DailyLog
   - `TaskWithDate`: Extends Task with date information for sorting/grouping

2. **JIRA Integration** (lines 55-193):
   - Red Hat JIRA-specific (issues.redhat.com)
   - Uses regex patterns to extract ticket IDs from text or URLs
   - Fetches ticket summaries via REST API when `JIRA_PAT` env var is set
   - Falls back to basic links without summaries if PAT is missing
   - `processJiraTickets()`: Batch fetches ticket info for all tickets in a report
   - `formatJiraTicketHTML()`: Creates HTML links with optional summaries

3. **Cobra Commands** (lines 195-258):
   - `rootCmd`: Base command with persistent --file flag
   - `hoursCmd`: Calculate hours worked with --start-date and --end-date flags
   - `reportCmd`: Generate reports with date range and HTML output options
     - `--html-file`: Save HTML to file
     - `--open-html`: Auto-open HTML in browser
     - `--copy-html`: Copy HTML to clipboard
     - `--show-html`: Display HTML source in terminal

4. **Report Generation Logic** (lines 305-426):
   - Task categorization by status and blockers
   - Groups tasks by `jira_ticket` field (which serves as unique identifier)
   - Three sections:
     - **Completed**: Tasks with "completed" status OR "in progress" + description
     - **Next Up**: Tasks with `upnext_description` where most recent status is not "completed"
     - **Blocked**: Tasks where most recent entry has a blocker
   - Tracks "most recent task" per JIRA ticket to determine current status

5. **HTML Generation** (lines 526-727):
   - Simplified HTML structure for Slack compatibility
   - Uses nested `<ul>` and `<li>` tags (no complex CSS)
   - Processes JIRA tickets to fetch summaries before rendering
   - Includes clickable JIRA and GitHub PR links

6. **Platform-Specific Clipboard** (lines 433-506):
   - macOS: Uses osascript with HTML class
   - Linux: Tries wl-copy (Wayland), xclip, xsel (X11)
   - Windows: PowerShell clipboard commands

### Important Behavioral Details

**Task Grouping by jira_ticket**:
The `jira_ticket` field is the **unique identifier** for grouping related tasks across dates. Multiple entries with the same `jira_ticket` value are treated as updates to the same work item. The most recent task entry (by date) determines the current status for filtering "next up" and "blocked" sections.

**Status Progression Tracking**:
- `mostRecentTasks` map (line 321) tracks the latest task for each jira_ticket
- "Next up" tasks are filtered to only show tickets where the most recent status is "in progress" or "not started" (lines 355-362)
- This prevents completed tasks from appearing in future planning sections

**Completed Tasks Logic** (lines 335-338):
Tasks appear in the "completed" section if they have:
- Status = "completed", OR
- Status = "in progress" + non-empty description (representing actual work done)

## YAML Data Structure

The `worklog.yml` file uses dates as top-level keys (YYYY-MM-DD format):

```yaml
"2024-07-26":
  work_log:
    - start_time: "09:05"
      end_time: "12:15"
  tasks:
    - jira_ticket: "PROJ-1234"        # Required: unique identifier
      description: "Task description"
      status: "completed"              # completed | in progress | not started
      github_pr: "https://..."
      upnext_description: "Next steps"
      blocker: "Waiting for X"
```

## Environment Variables

- `JIRA_PAT`: Red Hat JIRA Personal Access Token (optional)
  - When set: Reports include JIRA ticket summaries
  - When unset: Reports include basic JIRA links without summaries

## HTML Output and Slack Integration

The HTML output is specifically designed for Slack compatibility:
- Simple nested list structure (`<ul>`/`<li>`)
- No complex CSS or styling
- Can be copied from browser and pasted directly into Slack
- Use `--html-file report.html --open-html` to generate, then copy/paste into Slack

## Dependencies

- `github.com/spf13/cobra`: CLI framework
- `gopkg.in/yaml.v3`: YAML parsing
- Standard library: encoding/json, net/http, time, etc.

## Logging

Uses structured logging with `slog` (standard library):
- JSON format output to stderr
- Warnings for JIRA API failures
- Errors for file operations and command execution
