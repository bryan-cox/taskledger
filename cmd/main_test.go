package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// --- Test Setup ---

func setupTests(t *testing.T) (string, func()) {
	t.Helper()
	content := []byte(`
"2024-08-01":
  work_log:
    - start_time: "09:00"
      end_time: "12:30"
    - start_time: "13:30"
      end_time: "17:00"
  tasks:
    - description: "Set up the Go module and initial file structure."
      status: "completed"
      jira_ticket: "SCR-1"
      github_pr: ""
      upnext_description: ""
      blocker: ""
"2024-08-02":
  work_log:
    - start_time: "10:00"
      end_time: "16:00"
  tasks:
    - description: "Implement the structs and parsing logic for the worklog YAML."
      status: "in progress"
      jira_ticket: "SCR-2"
      github_pr: ""
      upnext_description: "Continue working on YAML parsing logic"
      blocker: "Waiting on final YAML structure."
    - description: "Provided feedback on the new database schema."
      status: "completed"
      jira_ticket: "PROJ-99"
      github_pr: "https://github.com/example/repo/pull/123"
      upnext_description: ""
      blocker: ""
"2024-08-03":
  work_log:
    - start_time: "11:00"
      end_time: "13:00"
  tasks:
    - description: "Building the 'hours' and 'report' commands."
      status: "in progress"
      jira_ticket: "SCR-3"
      github_pr: ""
      upnext_description: "Implement CLI commands for hours and reports"
      blocker: ""
`)
	tmpfile, err := os.CreateTemp("", "test_worklog.*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpfile.Name(), func() {
		os.Remove(tmpfile.Name())
	}
}

// executeCommandText captures plain text output from a command.
func executeCommandText(t *testing.T, args ...string) string {
	t.Helper()
	b := new(bytes.Buffer)

	// Set the command's output to our buffer
	rootCmd.SetOut(b)
	rootCmd.SetErr(b) // Capture errors too if needed
	rootCmd.SetArgs(args)

	// Reset flags to default values before each run
	rootCmd.PersistentFlags().Set("file", "worklog.yml")
	hoursCmd.Flags().Set("start-date", "")
	hoursCmd.Flags().Set("end-date", "")
	reportCmd.Flags().Set("start-date", "")
	reportCmd.Flags().Set("end-date", "")

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	return b.String()
}

// --- Test Functions ---

func TestHoursCommand(t *testing.T) {
	tmpFile, cleanup := setupTests(t)
	defer cleanup()

	// Reset slog to default for other tests
	defer slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	t.Run("calculates hours and outputs text", func(t *testing.T) {
		output := executeCommandText(t, "hours", "--file", tmpFile, "--start-date", "2024-08-01")
		expected := "Total hours worked from 2024-08-01 to 2024-08-01: 7.00\n"
		if output != expected {
			t.Errorf("Expected output:\n%q\nGot:\n%q", expected, output)
		}
	})
}

func TestReportCommand(t *testing.T) {
	tmpFile, cleanup := setupTests(t)
	defer cleanup()

	t.Run("generates a human-readable report", func(t *testing.T) {
		output := executeCommandText(t, "report", "--file", tmpFile, "--start-date", "2024-08-01", "--end-date", "2024-08-03")

		// Check for key parts of the text report
		if !strings.Contains(output, "Work Report (2024-08-01 to 2024-08-03)") {
			t.Error("Report missing correct title")
		}
		if !strings.Contains(output, "🦀 Thing I've been working on") {
			t.Error("Report missing 'Thing I've been working on' section")
		}
		if !strings.Contains(output, "SCR-1:") {
			t.Error("Report missing completed Jira ticket SCR-1")
		}
		if !strings.Contains(output, "• Set up the Go module and initial file structure.") {
			t.Error("Report missing completed task description with bullet")
		}
		if !strings.Contains(output, ":starfleet: Thing I plan on working on next") {
			t.Error("Report missing 'Thing I plan on working on next' section")
		}
		if !strings.Contains(output, "• SCR-2") {
			t.Error("Report missing next up task with bullet")
		}
		if !strings.Contains(output, ":facepalm: Thing that is blocking me or that I could use some help / discussion about") {
			t.Error("Report missing 'Blockers' section")
		}
		if !strings.Contains(output, "SCR-2") {
			t.Error("Report missing blocked task ticket")
		}
		if !strings.Contains(output, "• Blocker: Waiting on final YAML structure.") {
			t.Error("Report missing blocker description with bullet")
		}
		// Check for GitHub PR integration
		if !strings.Contains(output, "PR: https://github.com/example/repo/pull/123") {
			t.Error("Report missing GitHub PR link")
		}
	})
}
