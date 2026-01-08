package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bryan-cox/taskledger/internal/model"
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
    - jira_ticket: "SCR-1"
      description: "Set up the Go module and initial file structure."
      status: "completed"
      github_pr: ""
      upnext_description: ""
      blocker: ""
    - jira_ticket: ""
      description: "Organized project documentation and created initial README."
      status: "completed"
      github_pr: "https://github.com/example/repo/pull/456"
      upnext_description: ""
      blocker: ""
"2024-08-02":
  work_log:
    - start_time: "10:00"
      end_time: "16:00"
  tasks:
    - jira_ticket: "SCR-2"
      description: "Implement the structs and parsing logic for the worklog YAML."
      status: "in progress"
      github_pr: ""
      upnext_description: "Continue working on YAML parsing logic"
      blocker: "Waiting on final YAML structure."
    - jira_ticket: "PROJ-99"
      description: "Provided feedback on the new database schema."
      status: "completed"
      github_pr: "https://github.com/example/repo/pull/123"
      upnext_description: ""
      blocker: ""
    - jira_ticket: ""
      description: "Updated team wiki with new development processes."
      status: "completed"
      github_pr: ""
      upnext_description: ""
      blocker: ""
"2024-08-03":
  work_log:
    - start_time: "11:00"
      end_time: "13:00"
  tasks:
    - jira_ticket: "SCR-3"
      description: "Building the 'hours' and 'report' commands."
      status: "in progress"
      github_pr: ""
      upnext_description: "Implement CLI commands for hours and reports"
      blocker: ""
    - jira_ticket: ""
      description: "Fixed minor linting issues across the codebase."
      status: "not started"
      github_pr: ""
      upnext_description: "Run linter and fix all warnings"
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
		if !strings.Contains(output, "◦ Set up the Go module and initial file structure.") {
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
		if !strings.Contains(output, "◦ Blocker: Waiting on final YAML structure.") {
			t.Error("Report missing blocker description with bullet")
		}
		// Check for GitHub PR integration
		if !strings.Contains(output, "PR(s): https://github.com/example/repo/pull/123") {
			t.Error("Report missing GitHub PR link")
		}
		// Check for non-feature work grouping (tasks without Jira tickets)
		if !strings.Contains(output, "• Non-feature work:") {
			t.Error("Report should include 'Non-feature work' header for tasks without Jira tickets")
		}
		// Non-feature work items with empty tickets are grouped under "Misc"
		if !strings.Contains(output, "◦ Misc") {
			t.Error("Report should include 'Misc' sub-header for tasks without Jira tickets")
		}
		if !strings.Contains(output, "▪ Organized project documentation and created initial README.") {
			t.Error("Report should include completed task that has no Jira ticket under Non-feature work")
		}
		if !strings.Contains(output, "▪ Updated team wiki with new development processes.") {
			t.Error("Report should include completed task that has no Jira ticket under Non-feature work")
		}
		if !strings.Contains(output, "▪ Run linter and fix all warnings") {
			t.Error("Report should include next up task that has no Jira ticket under Non-feature work")
		}
		// Check for PR links for tasks without Jira tickets
		if !strings.Contains(output, "PR(s): https://github.com/example/repo/pull/456") {
			t.Error("Report should include PR link for task that has no Jira ticket")
		}
	})
}

func TestReportCommandWithDescriptionsArray(t *testing.T) {
	// Create test file with descriptions array
	content := []byte(`
"2024-08-10":
  work_log:
    - start_time: "09:00"
      end_time: "17:00"
  tasks:
    - jira_ticket: "TEST-100"
      descriptions:
        - "Morning: Reviewed code and identified issues"
        - "Midday: Implemented fixes for authentication bug"
        - "Afternoon: Added comprehensive test coverage"
      status: "completed"
      github_pr: "https://github.com/example/repo/pull/999"
      upnext_description: ""
      blocker: ""
    - jira_ticket: "TEST-200"
      descriptions:
        - "Started investigation into performance regression"
        - "Added profiling to identify bottleneck"
      status: "in progress"
      github_pr: ""
      upnext_description: "Complete performance optimization work"
      blocker: ""
`)
	tmpfile, err := os.CreateTemp("", "test_worklog_desc.*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Reset slog to default
	defer slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	t.Run("handles descriptions array correctly", func(t *testing.T) {
		output := executeCommandText(t, "report", "--file", tmpfile.Name())

		// Check that all descriptions from the array appear
		if !strings.Contains(output, "Morning: Reviewed code and identified issues") {
			t.Error("Report missing first description from array")
		}
		if !strings.Contains(output, "Midday: Implemented fixes for authentication bug") {
			t.Error("Report missing second description from array")
		}
		if !strings.Contains(output, "Afternoon: Added comprehensive test coverage") {
			t.Error("Report missing third description from array")
		}

		// Check next up section uses descriptions array
		if !strings.Contains(output, "Complete performance optimization work") {
			t.Error("Report should show upnext_description for in-progress task")
		}

		// Verify PR link is included
		if !strings.Contains(output, "https://github.com/example/repo/pull/999") {
			t.Error("Report missing PR link for completed task")
		}
	})
}

// --- Init Command Tests ---

func TestCreateInitialWorklog(t *testing.T) {
	// Use a fixed date for deterministic testing
	fixedDate := time.Date(2025, 10, 31, 12, 0, 0, 0, time.UTC)

	t.Run("creates worklog with correct dates", func(t *testing.T) {
		workData := createInitialWorklog(fixedDate)

		expectedToday := "2025-10-31"
		expectedYesterday := "2025-10-30"

		if _, ok := workData[expectedToday]; !ok {
			t.Errorf("Expected worklog to contain entry for today (%s), but it doesn't", expectedToday)
		}

		if _, ok := workData[expectedYesterday]; !ok {
			t.Errorf("Expected worklog to contain entry for yesterday (%s), but it doesn't", expectedYesterday)
		}

		if len(workData) != 2 {
			t.Errorf("Expected worklog to contain exactly 2 entries, got %d", len(workData))
		}
	})

	t.Run("populates work_log entries correctly", func(t *testing.T) {
		workData := createInitialWorklog(fixedDate)

		yesterday := workData["2025-10-30"]
		if len(yesterday.WorkLogEntries) != 2 {
			t.Errorf("Expected 2 work log entries for yesterday, got %d", len(yesterday.WorkLogEntries))
		}

		// Verify work log structure
		if yesterday.WorkLogEntries[0].StartTime == "" || yesterday.WorkLogEntries[0].EndTime == "" {
			t.Error("Work log entries should have start_time and end_time populated")
		}

		today := workData["2025-10-31"]
		if len(today.WorkLogEntries) != 2 {
			t.Errorf("Expected 2 work log entries for today, got %d", len(today.WorkLogEntries))
		}
	})

	t.Run("includes all task field types", func(t *testing.T) {
		workData := createInitialWorklog(fixedDate)

		// Collect all tasks from both days
		var allTasks []model.Task
		for _, dailyLog := range workData {
			allTasks = append(allTasks, dailyLog.Tasks...)
		}

		// Track which fields are used across all tasks
		hasStatus := false
		hasDescription := false
		hasDescriptions := false
		hasJiraTicket := false
		hasQCGoal := false
		hasUpnextDescription := false
		hasGithubPR := false
		hasBlocker := false

		for _, task := range allTasks {
			if task.Status != "" {
				hasStatus = true
			}
			if task.Description != "" {
				hasDescription = true
			}
			if len(task.Descriptions) > 0 {
				hasDescriptions = true
			}
			if task.JiraTicket != "" {
				hasJiraTicket = true
			}
			if task.QCGoal != "" {
				hasQCGoal = true
			}
			if task.UpnextDescription != "" {
				hasUpnextDescription = true
			}
			if task.GithubPR != "" {
				hasGithubPR = true
			}
			if task.Blocker != "" {
				hasBlocker = true
			}
		}

		// Verify all field types are represented
		if !hasStatus {
			t.Error("No tasks have status field populated")
		}
		if !hasDescription {
			t.Error("No tasks have description field populated")
		}
		if !hasDescriptions {
			t.Error("No tasks have descriptions array populated")
		}
		if !hasJiraTicket {
			t.Error("No tasks have jira_ticket field populated")
		}
		if !hasQCGoal {
			t.Error("No tasks have qc_goal field populated")
		}
		if !hasUpnextDescription {
			t.Error("No tasks have upnext_description field populated")
		}
		if !hasGithubPR {
			t.Error("No tasks have github_pr field populated")
		}
		if !hasBlocker {
			t.Error("No tasks have blocker field populated")
		}
	})

	t.Run("includes all task status types", func(t *testing.T) {
		workData := createInitialWorklog(fixedDate)

		// Collect all tasks
		var allTasks []model.Task
		for _, dailyLog := range workData {
			allTasks = append(allTasks, dailyLog.Tasks...)
		}

		// Track status types
		statuses := make(map[string]bool)
		for _, task := range allTasks {
			statuses[strings.ToLower(task.Status)] = true
		}

		// Verify all three status types are present
		expectedStatuses := []string{"completed", "in progress", "not started"}
		for _, status := range expectedStatuses {
			if !statuses[status] {
				t.Errorf("Expected to find task with status '%s', but none found", status)
			}
		}
	})

	t.Run("creates valid task structure", func(t *testing.T) {
		workData := createInitialWorklog(fixedDate)

		for date, dailyLog := range workData {
			if len(dailyLog.Tasks) == 0 {
				t.Errorf("Expected tasks for date %s, but found none", date)
			}

			for i, task := range dailyLog.Tasks {
				// Every task should have a status
				if task.Status == "" {
					t.Errorf("Task %d on %s has empty status", i, date)
				}
			}
		}
	})
}

func TestGenerateInitialWorklogYAML(t *testing.T) {
	fixedDate := time.Date(2025, 10, 31, 12, 0, 0, 0, time.UTC)

	t.Run("generates valid YAML", func(t *testing.T) {
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("generateInitialWorklogYAML failed: %v", err)
		}

		if len(yamlData) == 0 {
			t.Error("Generated YAML is empty")
		}

		// Verify it's valid YAML by unmarshaling it
		var workData model.WorkData
		err = yaml.Unmarshal(yamlData, &workData)
		if err != nil {
			t.Errorf("Generated YAML is not valid: %v", err)
		}
	})

	t.Run("generated YAML can be round-tripped", func(t *testing.T) {
		// Generate YAML
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("generateInitialWorklogYAML failed: %v", err)
		}

		// Unmarshal back to WorkData
		var workData model.WorkData
		err = yaml.Unmarshal(yamlData, &workData)
		if err != nil {
			t.Fatalf("Failed to unmarshal generated YAML: %v", err)
		}

		// Verify the data structure is intact
		expectedToday := "2025-10-31"
		expectedYesterday := "2025-10-30"

		if _, ok := workData[expectedToday]; !ok {
			t.Errorf("Unmarshaled data missing entry for %s", expectedToday)
		}

		if _, ok := workData[expectedYesterday]; !ok {
			t.Errorf("Unmarshaled data missing entry for %s", expectedYesterday)
		}

		// Verify tasks are preserved
		for date, dailyLog := range workData {
			if len(dailyLog.Tasks) == 0 {
				t.Errorf("Unmarshaled data has no tasks for date %s", date)
			}

			if len(dailyLog.WorkLogEntries) == 0 {
				t.Errorf("Unmarshaled data has no work log entries for date %s", date)
			}
		}
	})

	t.Run("YAML contains all expected field names", func(t *testing.T) {
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("generateInitialWorklogYAML failed: %v", err)
		}

		yamlString := string(yamlData)

		// Check for expected YAML field names
		expectedFields := []string{
			"work_log",
			"start_time",
			"end_time",
			"tasks",
			"status",
			"description",
			"descriptions",
			"jira_ticket",
			"qc_goal",
			"upnext_description",
			"github_pr",
			"blocker",
		}

		for _, field := range expectedFields {
			if !strings.Contains(yamlString, field) {
				t.Errorf("Generated YAML missing expected field: %s", field)
			}
		}
	})

	t.Run("preserves all task data after round-trip", func(t *testing.T) {
		// Create original data
		originalData := createInitialWorklog(fixedDate)

		// Generate YAML
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("generateInitialWorklogYAML failed: %v", err)
		}

		// Unmarshal back
		var roundTrippedData model.WorkData
		err = yaml.Unmarshal(yamlData, &roundTrippedData)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		// Compare task counts
		for date, originalLog := range originalData {
			roundTrippedLog, ok := roundTrippedData[date]
			if !ok {
				t.Errorf("Round-tripped data missing date %s", date)
				continue
			}

			if len(originalLog.Tasks) != len(roundTrippedLog.Tasks) {
				t.Errorf("Date %s: original has %d tasks, round-tripped has %d tasks",
					date, len(originalLog.Tasks), len(roundTrippedLog.Tasks))
			}

			if len(originalLog.WorkLogEntries) != len(roundTrippedLog.WorkLogEntries) {
				t.Errorf("Date %s: original has %d work log entries, round-tripped has %d entries",
					date, len(originalLog.WorkLogEntries), len(roundTrippedLog.WorkLogEntries))
			}

			// Verify specific task fields are preserved
			for i, originalTask := range originalLog.Tasks {
				if i >= len(roundTrippedLog.Tasks) {
					break
				}
				roundTrippedTask := roundTrippedLog.Tasks[i]

				if originalTask.Status != roundTrippedTask.Status {
					t.Errorf("Task %d on %s: status mismatch (original: %s, round-tripped: %s)",
						i, date, originalTask.Status, roundTrippedTask.Status)
				}

				if originalTask.JiraTicket != roundTrippedTask.JiraTicket {
					t.Errorf("Task %d on %s: jira_ticket mismatch", i, date)
				}

				if len(originalTask.Descriptions) != len(roundTrippedTask.Descriptions) {
					t.Errorf("Task %d on %s: descriptions array length mismatch", i, date)
				}
			}
		}
	})
}

func TestInitCommandDataValidation(t *testing.T) {
	fixedDate := time.Date(2025, 10, 31, 12, 0, 0, 0, time.UTC)

	t.Run("generated data has valid time entries", func(t *testing.T) {
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("Failed to generate YAML: %v", err)
		}

		var workData model.WorkData
		err = yaml.Unmarshal(yamlData, &workData)
		if err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		// Validate that all time entries can be parsed
		for date, dailyLog := range workData {
			for i, entry := range dailyLog.WorkLogEntries {
				_, err := time.Parse("15:04", entry.StartTime)
				if err != nil {
					t.Errorf("Date %s, entry %d: invalid start_time format '%s': %v",
						date, i, entry.StartTime, err)
				}

				_, err = time.Parse("15:04", entry.EndTime)
				if err != nil {
					t.Errorf("Date %s, entry %d: invalid end_time format '%s': %v",
						date, i, entry.EndTime, err)
				}
			}
		}
	})

	t.Run("generated data has valid status values", func(t *testing.T) {
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("Failed to generate YAML: %v", err)
		}

		var workData model.WorkData
		err = yaml.Unmarshal(yamlData, &workData)
		if err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		validStatuses := map[string]bool{
			"completed":   true,
			"in progress": true,
			"not started": true,
		}

		// Validate all task statuses are valid
		for date, dailyLog := range workData {
			for i, task := range dailyLog.Tasks {
				if !validStatuses[task.Status] {
					t.Errorf("Date %s, task %d: invalid status '%s'", date, i, task.Status)
				}
			}
		}
	})

	t.Run("generated data structure matches schema", func(t *testing.T) {
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("Failed to generate YAML: %v", err)
		}

		var workData model.WorkData
		err = yaml.Unmarshal(yamlData, &workData)
		if err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		// Verify the unmarshaled data conforms to WorkData structure
		for date, dailyLog := range workData {
			// Check date format
			_, err := time.Parse("2006-01-02", date)
			if err != nil {
				t.Errorf("Invalid date format: %s", date)
			}

			// Verify DailyLog structure
			if dailyLog.WorkLogEntries == nil {
				t.Errorf("Date %s: WorkLogEntries is nil (should be initialized)", date)
			}

			if dailyLog.Tasks == nil {
				t.Errorf("Date %s: Tasks is nil (should be initialized)", date)
			}

			// Verify each task can access all fields without panic
			for _, task := range dailyLog.Tasks {
				_ = task.Status
				_ = task.Description
				_ = task.Descriptions
				_ = task.JiraTicket
				_ = task.QCGoal
				_ = task.UpnextDescription
				_ = task.GithubPR
				_ = task.Blocker

				// Test GetDescriptions method works (nil is valid for empty descriptions)
				_ = task.GetDescriptions()
			}
		}
	})

	t.Run("generated data is suitable for report generation", func(t *testing.T) {
		yamlData, err := generateInitialWorklogYAML(fixedDate)
		if err != nil {
			t.Fatalf("Failed to generate YAML: %v", err)
		}

		var workData model.WorkData
		err = yaml.Unmarshal(yamlData, &workData)
		if err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		// Simulate the report generation logic to ensure data is suitable
		hasCompletedTask := false
		hasInProgressTask := false
		hasBlockedTask := false

		for _, dailyLog := range workData {
			for _, task := range dailyLog.Tasks {
				if strings.EqualFold(task.Status, "completed") ||
					(strings.EqualFold(task.Status, "in progress") && len(task.GetDescriptions()) > 0) {
					hasCompletedTask = true
				}

				if strings.EqualFold(task.Status, "in progress") {
					hasInProgressTask = true
				}

				if task.Blocker != "" {
					hasBlockedTask = true
				}
			}
		}

		if !hasCompletedTask {
			t.Error("Generated data should have at least one task suitable for 'completed' section")
		}

		if !hasInProgressTask {
			t.Error("Generated data should have at least one in-progress task")
		}

		if !hasBlockedTask {
			t.Error("Generated data should have at least one blocked task")
		}
	})

	t.Run("YAML serialization produces consistent output", func(t *testing.T) {
		// Generate YAML twice
		yaml1, err1 := generateInitialWorklogYAML(fixedDate)
		yaml2, err2 := generateInitialWorklogYAML(fixedDate)

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to generate YAML: %v, %v", err1, err2)
		}

		// The output should be identical for the same input date
		if !bytes.Equal(yaml1, yaml2) {
			t.Error("generateInitialWorklogYAML should produce consistent output for the same date")
		}
	})
}
