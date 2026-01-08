---
description: Generate HTML work report and open in browser
argument-hint: "[--file PATH] [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--output PATH]"
---

## Name
taskledger:html-report

## Synopsis
```
/taskledger:html-report [--file PATH] [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--output PATH]
```

## Description
Generates an HTML work report using TaskLedger and opens it in the default browser. The report includes completed tasks, next up items, and blockers organized by JIRA ticket, with ticket summaries fetched from JIRA.

This command is useful for:
- Generating weekly status reports for sharing in Slack
- Creating formatted work summaries for team standups
- Documenting work progress in an easy-to-share format

## Prerequisites
- The Atlassian JIRA MCP server must be configured and accessible (for fetching ticket summaries)
- TaskLedger binary must be built and available at `~/bryan-cox/taskledger/bin/taskledger`
- A valid worklog.yml file must exist

## Implementation

Execute the following workflow step by step:

### Phase 1: Parse Arguments

1. Extract optional arguments from $ARGUMENTS:
   - `--file PATH`: Path to worklog.yml file (defaults to `~/worklog/worklog.yaml`)
   - `--start-date YYYY-MM-DD`: Start of date range (defaults to today)
   - `--end-date YYYY-MM-DD`: End of date range (defaults to today)
   - `--output PATH`: Path for the generated HTML file (defaults to `weekly-report.html`)

2. Validate date formats if provided (must be YYYY-MM-DD)

3. Store all parsed values for later use

### Phase 2: Validate Prerequisites

1. Check that the TaskLedger binary exists at `~/bryan-cox/taskledger/bin/taskledger`:
   ```bash
   ls ~/bryan-cox/taskledger/bin/taskledger
   ```
   If not found, inform the user:
   ```
   TaskLedger binary not found at ~/bryan-cox/taskledger/bin/taskledger
   Please build it with: cd ~/bryan-cox/taskledger && make build
   ```

2. Check that the worklog file exists at the specified path:
   ```bash
   ls {worklog-file-path}
   ```
   If not found, inform the user:
   ```
   Worklog file not found at {path}
   Please check the file path or create a worklog file.
   ```

3. Verify JIRA MCP is available by attempting a simple query:
   - Use `mcp__atlassian__jira_get_all_projects` with a limit of 1
   - If this fails, warn the user that ticket summaries will not be available

### Phase 3: Build and Execute Command

1. Construct the TaskLedger command with all arguments:
   ```bash
   ~/bryan-cox/taskledger/bin/taskledger report \
     --html-file {output} \
     --open-html \
     --start-date {start-date} \
     --end-date {end-date} \
     --file {worklog-file}
   ```

2. Execute the command using the Bash tool

3. Capture the output for reporting

### Phase 4: Report Results

1. If successful, inform the user:
   ```
   === HTML Report Generated ===

   Date Range: {start-date} to {end-date}
   Output File: {output-path}
   Worklog Source: {worklog-file}

   The report has been opened in your default browser.
   You can copy the content and paste it directly into Slack.
   ```

2. If there were any warnings (e.g., JIRA MCP not available), include them:
   ```
   Note: JIRA ticket summaries may not be included (MCP not available)
   ```

## Error Handling

- **TaskLedger binary not found**: Display build instructions
- **Worklog file not found**: Show the path checked and suggest alternatives
- **Invalid date format**: Show correct format example (YYYY-MM-DD)
- **JIRA MCP not available**: Warn but continue (report will work without ticket summaries)
- **TaskLedger execution error**: Display the error output from the command

## Examples

1. **Generate report for today**:
   ```
   /html-report
   ```
   Generates a report for today's tasks and opens it in the browser.

2. **Generate report for a date range**:
   ```
   /html-report --start-date 2025-12-18 --end-date 2026-01-06
   ```
   Generates a report for the specified date range.

3. **Generate report with custom worklog file**:
   ```
   /html-report --file ~/my-worklog/work.yaml
   ```
   Uses a worklog file from a different location.

4. **Generate report with custom output path**:
   ```
   /html-report --output ~/reports/january-week1.html --start-date 2026-01-01 --end-date 2026-01-07
   ```
   Saves the report to a specific location.

5. **Full example with all options**:
   ```
   /html-report --file ~/worklog/worklog.yaml --start-date 2025-12-18 --end-date 2026-01-06 --output weekly-report.html
   ```
   Generates a report with all options explicitly specified.

## Arguments

- `--file` *(optional)*: Path to the worklog.yml file. Defaults to `~/worklog/worklog.yaml`. Can be an absolute or relative path.
- `--start-date` *(optional)*: Start of date range in YYYY-MM-DD format. Defaults to today.
- `--end-date` *(optional)*: End of date range in YYYY-MM-DD format. Defaults to today.
- `--output` *(optional)*: Path for the generated HTML file. Defaults to `weekly-report.html` in the current directory.

## See Also

- TaskLedger `/update-jira` command for posting status updates to JIRA tickets
- TaskLedger `hours` command for calculating hours worked
