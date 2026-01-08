---
description: Post work report comments to JIRA tickets from worklog.yml
argument-hint: "[--file PATH] [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--dry-run]"
---

## Name
taskledger:update-jira

## Synopsis
```
/taskledger:update-jira [--file PATH] [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--dry-run]
```

## Description
Reads a worklog.yml file, generates a formatted status update comment for each JIRA ticket referenced in the tasks, checks for duplicate comments to avoid spam, and posts the updates to JIRA using the Atlassian MCP server.

This command is useful for:
- Keeping JIRA tickets updated with work progress
- Documenting completed work directly on tickets
- Sharing status with team members via JIRA

## Prerequisites
- The Atlassian JIRA MCP server must be configured and accessible
- A valid `worklog.yml` file must exist (either at the specified path or in the current working directory)

## Implementation

Execute the following workflow step by step:

### Phase 1: Parse Arguments

1. Extract optional arguments from $ARGUMENTS:
   - `--file PATH`: Path to worklog.yml file (defaults to `./worklog.yml` in current directory)
   - `--start-date YYYY-MM-DD`: Start of date range (defaults to today)
   - `--end-date YYYY-MM-DD`: End of date range (defaults to today)
   - `--dry-run`: Preview mode - show what would be posted without actually posting

2. Validate date formats if provided (must be YYYY-MM-DD)

3. Store the file path and dry-run flag for later use

### Phase 2: Load and Parse worklog.yml

1. Read the worklog.yml file from the specified path (or current directory if not specified) using the Read tool

2. Parse the YAML content. The structure is:
   ```yaml
   "YYYY-MM-DD":
     work_log:
       - start_time: "HH:MM"
         end_time: "HH:MM"
     tasks:
       - jira_ticket: "PROJ-123"
         description: "Task description"
         descriptions: ["Multiple", "descriptions"]
         status: "completed"  # or "in progress" or "not started"
         github_pr: "https://github.com/..."
         upnext_description: "Next steps"
         blocker: "Blocking issue"
   ```

3. Filter entries to only include dates within the specified range (inclusive)

4. If no tasks found in range, inform the user and exit:
   ```
   No tasks found in worklog.yml for the date range {start-date} to {end-date}.
   ```

### Phase 3: Categorize Tasks by JIRA Ticket

1. Create a map of jira_ticket -> list of tasks across all dates in range

2. For each unique jira_ticket, collect:
   - All descriptions (from both `description` and `descriptions` fields)
   - All GitHub PR links
   - Any blockers (from most recent entry)
   - Any upnext_description (from most recent entry)

3. Skip tasks without a jira_ticket value (they cannot be updated in JIRA)

4. Extract the ticket ID from each jira_ticket field:
   - If it's a URL like `https://issues.redhat.com/browse/PROJ-123`, extract `PROJ-123`
   - If it's already a ticket ID like `PROJ-123`, use it directly
   - Pattern: look for `[A-Z]+-\d+` format

### Phase 4: Generate Comment Content per Ticket

For each unique JIRA ticket, generate a comment in Jira wiki markup format:

```
h2. Status Update: {start-date} to {end-date}

*Work Completed:*
* {description 1}
* {description 2}
* {description N}

*Pull Requests:*
* [{PR URL short name}|{full PR URL}]

*Next Steps:* {upnext_description}

*Blockers:* {blocker text}

----
_Generated via TaskLedger /update-jira_
```

Rules for formatting:
- Only include "Pull Requests" section if there are PR links
- Only include "Next Steps" section if upnext_description is non-empty
- Only include "Blockers" section if there is an actual blocker (do NOT show "Blockers: None")
- Do NOT include task status (JIRA has its own status field)
- Use proper Jira wiki link syntax: `[Display Text|URL]`

### Phase 5: Check for Duplicate Comments

For each ticket that will be updated:

1. Fetch the issue with comments using `mcp__atlassian__jira_get_issue`:
   ```
   issue_key: "{TICKET-ID}"
   comment_limit: 20
   expand: "renderedFields"
   ```

2. Parse the comments and check if any contain BOTH:
   - The text "Generated via TaskLedger"
   - The same date range header "Status Update: {start-date} to {end-date}"

3. If a duplicate is found, mark the ticket as having an existing comment

### Phase 6: Preview and Confirmation

Display a summary to the user:

```
=== JIRA Update Preview ===

Date Range: {start-date} to {end-date}

Tickets to update:
1. {TICKET-1}: {summary from JIRA} - {N} work items
2. {TICKET-2}: {summary from JIRA} - {N} work items
   [DUPLICATE - comment already exists for this date range]

--- Comment Preview for {TICKET-1} ---
{full comment content}
---

Dry run: {yes/no}
```

If NOT in dry-run mode, ask for confirmation:
```
Post these comments to JIRA?
- yes: Post all comments (skip duplicates)
- all: Post all comments (including duplicates)
- no: Cancel
```

If in dry-run mode:
```
Dry run complete. No comments were posted.
To post comments, run without --dry-run flag.
```

### Phase 7: Post Comments

If user confirms with "yes" or "all":

1. For each ticket (skipping duplicates unless user said "all"):

2. Use `mcp__atlassian__jira_add_comment`:
   ```
   issue_key: "{TICKET-ID}"
   comment: "{generated comment content}"
   ```

3. Track success/failure for each ticket

4. Report results:
   ```
   === JIRA Update Results ===

   Posted successfully:
   - TICKET-1: https://issues.redhat.com/browse/TICKET-1
   - TICKET-2: https://issues.redhat.com/browse/TICKET-2

   Skipped (duplicates):
   - TICKET-3: Comment already exists for this date range

   Errors:
   - TICKET-4: {error message}
   ```

## Error Handling

- **worklog.yml not found**: Display error with hint about expected location
- **Invalid YAML**: Show parse error and line number if available
- **Invalid date format**: Show correct format example (YYYY-MM-DD)
- **JIRA MCP not available**: Inform user to check MCP configuration
- **JIRA API errors**: Log error, continue with remaining tickets, report at end
- **No JIRA tickets found**: Inform user that no tasks have jira_ticket values

## Examples

1. **Preview what would be posted (dry run)**:
   ```
   /update-jira --dry-run
   ```
   Shows all tickets and comments without posting anything.

2. **Update tickets for today**:
   ```
   /update-jira
   ```
   Posts comments for all tasks logged today.

3. **Update tickets for a specific date range**:
   ```
   /update-jira --start-date 2025-01-06 --end-date 2025-01-07
   ```
   Posts comments summarizing work from Jan 6-7.

4. **Update tickets for past week**:
   ```
   /update-jira --start-date 2025-01-01
   ```
   Posts comments from Jan 1 to today.

5. **Use a worklog file from a different location**:
   ```
   /update-jira --file ~/worklog/worklog.yaml
   ```
   Uses the worklog file from the specified path.

6. **Combine file path with date range**:
   ```
   /update-jira --file /path/to/worklog.yml --start-date 2025-01-06 --end-date 2025-01-07 --dry-run
   ```
   Preview comments for a specific date range using a custom worklog file.

## Arguments

- `--file` *(optional)*: Path to the worklog.yml file. Defaults to `./worklog.yml` in the current directory. Can be an absolute or relative path.
- `--start-date` *(optional)*: Start of date range in YYYY-MM-DD format. Defaults to today.
- `--end-date` *(optional)*: End of date range in YYYY-MM-DD format. Defaults to today.
- `--dry-run` *(optional)*: Preview mode. Shows what would be posted without actually posting to JIRA.

## See Also

- TaskLedger `report` command for generating text/HTML reports
- TaskLedger `hours` command for calculating hours worked
