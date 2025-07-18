# TaskLedger

TaskLedger is a command-line interface (CLI) tool for tracking work and generating reports from a YAML log file. It helps you maintain a structured log of your daily tasks, calculate hours worked, and generate progress reports in both human-readable text and formatted HTML.

## Disclaimer: AI-Assisted Development

This project heavily utilizes artificial intelligence for code generation. While AI has been a powerful tool in the development process, it's important to understand the following:

* **No Guarantees:** The code is provided on an "as-is" basis. There is no guarantee that it is free of bugs, security vulnerabilities, or other issues.
* **Use with Caution:** You are solely responsible for any outcomes that result from using this code. Always review and test the code thoroughly before implementing it in a production environment.
* **AI Is Not Perfect:** The AI may have generated code that is suboptimal or contains inaccuracies.

By using the code in this repository, you acknowledge and agree to these terms. 

## Features

* Log daily work entries, including tasks, status, and blockers.
* Calculate total hours worked for any given day or date range.
* Generate human-readable reports with emoji sections:
    * 🦀 **Thing I've been working on** - Completed tasks grouped by Jira ticket
    * :starfleet: **Thing I plan on working on next** - In-progress tasks without blockers
    * :facepalm: **Thing that is blocking me** - Tasks with blockers that need attention
* **JIRA Integration:**
    * Automatic conversion of JIRA ticket references to clickable links
    * Fetch and display JIRA ticket summaries (when `JIRA_PAT` is configured)
    * Support for both ticket IDs and full JIRA URLs
* **HTML Output Options:**
    * Export reports as formatted HTML files
    * Automatically open HTML reports in default browser
    * Copy HTML to system clipboard (when clipboard tools are available)
    * Display HTML source in terminal
* Support for GitHub PR tracking and upnext descriptions
* Simple and extensible command structure powered by Cobra
* Structured logging with `slog` for easy integration with other tools

## Installation & Setup

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/your-username/taskledger.git](https://github.com/your-username/taskledger.git)
    cd taskledger
    ```

2.  **Initialize Go Module (if not already done):**
    ```bash
    go mod init [github.com/your-username/taskledger](https://github.com/your-username/taskledger)
    go mod tidy
    ```

3.  **Build the binary:**
    Use the provided `Makefile` to build the application.
    ```bash
    make build
    ```
    This will create an executable at `./bin/taskledger`.

## The `worklog.yml` File

TaskLedger reads from a `worklog.yml` file in the project root by default. You can create this file and structure it as follows:

```yaml
# A log of work, organized by date.
# Each date is a top-level key in "YYYY-MM-DD" format.
"2024-07-26":
  work_log:
    - start_time: "09:05"
      end_time: "12:15"
    - start_time: "13:00"
      end_time: "17:30"
  tasks:
    - jira_ticket: "PROJ-1234"
      description: "Implemented a new OAuth 2.0 authentication flow for user login. This included front-end and back-end changes."
      status: "completed" # Can be: not started, in progress, completed
      github_pr: "https://github.com/example/repo/pull/123"
      upnext_description: ""
      blocker: "" # Leave empty if not blocked

"2024-07-27":
  work_log:
    - start_time: "09:00"
      end_time: "17:00"
  tasks:
    - jira_ticket: "PROJ-5678"
      description: "Investigating a bug where the quarterly report fails to generate for large datasets. The issue seems to be a memory leak."
      status: "in progress"
      github_pr: ""
      upnext_description: "Continue debugging the memory leak issue"
      blocker: "Waiting for access to the production database logs to replicate the issue."
```

## JIRA Integration

TaskLedger can automatically convert JIRA ticket references into clickable links and fetch ticket summaries from the Red Hat JIRA instance.

## Slack Integration

TaskLedger's HTML output is specifically optimized for Slack compatibility:

* **Nested List Structure**: HTML reports use proper `<ul>` and `<li>` tags that Slack interprets correctly
* **Simple Formatting**: Removes complex CSS that Slack doesn't support  
* **Copy/Paste Friendly**: HTML can be copied directly from browser and pasted into Slack with preserved formatting
* **Maintains Hierarchy**: JIRA tickets appear as main bullets with task details as sub-bullets

**Usage for Slack**: Generate an HTML report, open it in your browser with `--open-html`, then copy the content and paste directly into Slack for properly formatted status updates.

### Configuration

To enable JIRA ticket summary fetching, set the `JIRA_PAT` environment variable with your Personal Access Token:

```bash
export JIRA_PAT="your_personal_access_token_here"
```

### Getting a JIRA Personal Access Token

1. Log into [Red Hat JIRA](https://issues.redhat.com/)
2. Go to your **Account Settings** → **Security** → **Create and manage API tokens**
3. Click **Create API token**
4. Give it a name (e.g., "TaskLedger CLI")
5. Copy the generated token and set it as the `JIRA_PAT` environment variable

### JIRA Integration Features

* **Without `JIRA_PAT`:** JIRA references become clickable links (e.g., `CNTRLPLANE-123` → link to ticket)
* **With `JIRA_PAT`:** Links include ticket summaries (e.g., `CNTRLPLANE-123: FBC Integration`)
* **Supported formats:**
  - Ticket IDs: `PROJ-123`, `CNTRLPLANE-456`
  - Full URLs: `https://issues.redhat.com/browse/PROJ-123`
* **Error handling:** If API calls fail, falls back to basic links with warning logs

### Example YAML with JIRA Integration

```yaml
"2024-07-26":
  tasks:
    - jira_ticket: "CNTRLPLANE-123"  # Will become a clickable link
      description: "Implemented FBC integration"
      status: "completed"
    - jira_ticket: "https://issues.redhat.com/browse/PROJ-456"  # Also works with full URLs
      description: "Bug investigation"
      status: "in progress"
```

## Usage

Here are some examples of how to run the CLI tool from your terminal.

### Calculating Hours

* **Calculate hours for a single day:**
    ```bash
    ./bin/taskledger hours --start-date=2024-07-26
    ```

* **Calculate total hours over a date range:**
    ```bash
    ./bin/taskledger hours --start-date=2024-07-26 --end-date=2024-07-27
    ```

* **Calculate total hours for all entries in the log:**
    ```bash
    ./bin/taskledger hours
    ```

### Generating Reports

* **Generate a report for a single day:**
    ```bash
    ./bin/taskledger report --start-date=2024-07-27
    ```

* **Generate a report for a date range:**
    ```bash
    ./bin/taskledger report --start-date=2024-07-26 --end-date=2024-07-27
    ```

* **Generate a report for all entries in the log:**
    ```bash
    ./bin/taskledger report
    ```

### HTML Output Options

TaskLedger can generate beautifully formatted HTML reports with clickable JIRA links and styled sections.

* **Save report as HTML file:**
    ```bash
    ./bin/taskledger report --html-file report.html
    ```

* **Save and automatically open HTML file in browser:**
    ```bash
    ./bin/taskledger report --html-file report.html --open-html
    ```

* **Copy HTML report to clipboard (when clipboard tools available):**
    ```bash
    ./bin/taskledger report --copy-html
    ```

* **Display HTML source in terminal:**
    ```bash
    ./bin/taskledger report --show-html
    ```

* **Combine options:**
    ```bash
    # Generate HTML report with JIRA summaries, save to file, and auto-open
    export JIRA_PAT="your_token"
    ./bin/taskledger report --html-file weekly-report.html --open-html --start-date=2024-07-26 --end-date=2024-07-27
    
    # Generate HTML and both save to file and copy to clipboard
    ./bin/taskledger report --html-file report.html --copy-html --open-html
    ```

**HTML Features:**
- Clean, modern styling with proper typography
- Clickable JIRA ticket links (with summaries when `JIRA_PAT` is configured)
- Clickable GitHub PR links
- Responsive design that works in browsers and email clients
- Professional formatting suitable for sharing with stakeholders
- Cross-platform auto-open support (macOS, Linux, Windows)
- Slack-compatible nested list structure for easy copy/paste

### Sample Report Output

```
Work Report (2024-07-26 to 2024-07-27)
========================================
=======Autogenerated by TaskLedger=======

🦀 Thing I've been working on
    • PROJ-1234: 
        • Implemented a new OAuth 2.0 authentication flow for user login. This included front-end and back-end changes.
            • PR: https://github.com/example/repo/pull/123

:starfleet: Thing I plan on working on next
    • PROJ-5678
        • Continue debugging the memory leak issue

:facepalm: Thing that is blocking me or that I could use some help / discussion about
• PROJ-5678 
  • Blocker: Waiting for access to the production database logs to replicate the issue.
```

### Using a Different Log File

* You can target any YAML file using the `--file` flag.
    ```bash
    ./bin/taskledger report --file=./archive/old_log.yml
    ```

### Getting Help

* **Get help for the main application:**
    ```bash
    ./bin/taskledger --help
    ```

* **Get help for a specific subcommand:**
    ```bash
    ./bin/taskledger hours --help
    ./bin/taskledger report --help
    ```

## Important: Task Grouping Behavior

**The `jira_ticket` field serves as a unique identifier for grouping related tasks.** TaskLedger tracks the progression of work items by grouping all tasks with the same `jira_ticket` value together. This is crucial for proper status tracking and report generation.

### Key Points:

1. **Every task should have a unique `jira_ticket` identifier** - even for non-Jira work
2. **Tasks are grouped by `jira_ticket`** - multiple entries with the same identifier are treated as updates to the same work item
3. **Status progression is tracked chronologically** - the most recent task entry for each `jira_ticket` determines current status
4. **Completed tasks disappear from "next up"** - when the latest entry for a `jira_ticket` is marked "completed", it won't appear in future planning sections

### Examples of Good `jira_ticket` Values:

```yaml
# Official Jira tickets
jira_ticket: "PROJ-1234"
jira_ticket: "https://company.atlassian.net/browse/PROJ-1234"

# Custom identifiers for non-Jira work
jira_ticket: "NO-JIRA: Update Documentation"
jira_ticket: "ADMIN-001: Setup New Environment" 
jira_ticket: "BUG-FIX: Login Page CSS Issue"
jira_ticket: "RESEARCH: Evaluate New Framework"
```

### What Happens Without Unique Identifiers:

If you leave `jira_ticket` empty or use the same value for unrelated tasks, TaskLedger cannot properly track task progression, and you may see completed work still appearing in "next up" sections.

## YAML Fields Reference

### Task Fields

- `jira_ticket`: **Required** - Unique identifier for grouping related tasks (Jira ticket ID, URL, or custom identifier)
- `description`: The main description of the task (required)
- `status`: Task status - "completed", "in progress", or "not started"
- `github_pr`: GitHub pull request URL
- `upnext_description`: Specific description for next up tasks
- `blocker`: Description of what's blocking the task (if any)
