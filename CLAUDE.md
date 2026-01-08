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
go test -v ./internal/report

# Run a single test
go test -v ./cmd -run TestReportCommand
```

## Code Architecture

### Package Structure

```
taskledger/
├── cmd/
│   ├── main.go           # CLI entry point, Cobra commands, orchestration
│   └── main_test.go      # Integration tests for CLI commands
├── internal/
│   ├── model/
│   │   └── model.go      # Core data structures (Task, WorkLog, etc.)
│   ├── jira/
│   │   └── jira.go       # JIRA API client and ticket formatting
│   ├── report/
│   │   ├── categorize.go # Task categorization logic
│   │   ├── text.go       # Text report rendering
│   │   └── html.go       # HTML report rendering
│   └── clipboard/
│       └── clipboard.go  # Platform-specific clipboard operations
├── CLAUDE.md
├── Makefile
└── go.mod
```

### Package Responsibilities

#### `internal/model`
Core data structures shared across the application:
- `WorkLog`: Time entries with start/end times
- `Task`: Work items with status, description, JIRA ticket, PR links, blockers
- `TaskWithDate`: Extends Task with date for sorting/grouping
- `DailyLog`: Combines work logs and tasks for a single date
- `WorkData`: Top-level map of date strings to DailyLog
- `CategorizedTasks`: Holds tasks organized by report section (completed/next up/blocked)
- Status constants: `StatusCompleted`, `StatusInProgress`, `StatusNotStarted`

#### `internal/jira`
Red Hat JIRA integration (issues.redhat.com):
- `ExtractTicketID()`: Extract ticket IDs from URLs or text using regex
- `FetchTicketSummary()`: Fetch ticket info via REST API when `JIRA_PAT` is set
- `ProcessTickets()`: Batch fetch ticket info for all tickets in a report
- `FormatTicketHTML()`: Create HTML links with optional summaries

#### `internal/report`
Report generation and rendering:
- `CategorizeTasks()`: Groups tasks into completed, next up, and blocked categories
- `PrintCompletedTasks()`, `PrintNextUpTasks()`, `PrintBlockedTasks()`: Text rendering
- `GenerateHTML()`: HTML report generation with JIRA integration

#### `internal/clipboard`
Platform-specific clipboard operations:
- `CopyHTML()`: Copy HTML to clipboard on macOS, Linux (Wayland/X11), Windows

#### `cmd/main.go`
CLI orchestration (~380 lines):
- Cobra command definitions (`hours`, `report`, `init`)
- Flag parsing and validation
- Data loading and date range handling
- HTML output handling (save, display, clipboard, browser)

### Important Behavioral Details

**Task Grouping by jira_ticket**:
The `jira_ticket` field is the **unique identifier** for grouping related tasks across dates. Multiple entries with the same `jira_ticket` value are treated as updates to the same work item. The most recent task entry (by date) determines the current status for filtering "next up" and "blocked" sections.

**Status Progression Tracking**:
- The categorization logic in `report.CategorizeTasks()` tracks the latest task for each jira_ticket
- "Next up" tasks are filtered to only show tickets where the most recent status is "in progress" or "not started"
- This prevents completed tasks from appearing in future planning sections

**Completed Tasks Logic**:
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
