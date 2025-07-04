# TaskLedger

TaskLedger is a command-line interface (CLI) tool for tracking work and generating reports from a YAML log file. It helps you maintain a structured log of your daily tasks, calculate hours worked, and generate progress reports in a clean, human-readable format.

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
    - description: "Implemented a new OAuth 2.0 authentication flow for user login. This included front-end and back-end changes."
      status: "completed" # Can be: not started, in progress, completed
      jira_ticket: "PROJ-1234"
      github_pr: "https://github.com/example/repo/pull/123"
      upnext_description: ""
      blocker: "" # Leave empty if not blocked

"2024-07-27":
  work_log:
    - start_time: "09:00"
      end_time: "17:00"
  tasks:
    - description: "Investigating a bug where the quarterly report fails to generate for large datasets. The issue seems to be a memory leak."
      status: "in progress"
      jira_ticket: "PROJ-5678"
      github_pr: ""
      upnext_description: "Continue debugging the memory leak issue"
      blocker: "Waiting for access to the production database logs to replicate the issue."
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

## YAML Fields Reference

### Task Fields

- `description`: The main description of the task (required)
- `status`: Task status - "completed", "in progress", or "not started"
- `jira_ticket`: Jira ticket identifier or URL
- `github_pr`: GitHub pull request URL
- `upnext_description`: Specific description for next up tasks
- `blocker`: Description of what's blocking the task (if any)
