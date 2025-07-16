package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- Data Structures to Match YAML ---

// WorkLog represents a single time entry (start and end).
type WorkLog struct {
	StartTime string `yaml:"start_time"`
	EndTime   string `yaml:"end_time"`
}

// Task represents a single work item.
type Task struct {
	Status            string `yaml:"status"`
	Description       string `yaml:"description"`
	JiraTicket        string `yaml:"jira_ticket"`
	UpnextDescription string `yaml:"upnext_description"`
	GithubPR          string `yaml:"github_pr"`
	Blocker           string `yaml:"blocker"`
}

// TaskWithDate represents a task with its associated date for sorting.
type TaskWithDate struct {
	Task
	Date string
}

// DailyLog contains all information for a single day.
type DailyLog struct {
	WorkLogEntries []WorkLog `yaml:"work_log"`
	Tasks          []Task    `yaml:"tasks"`
}

// WorkData is the top-level structure, mapping dates to daily logs.
type WorkData map[string]DailyLog

// --- Cobra Command Definitions ---

var (
	// Used for flags.
	filePath  string
	startDate string
	endDate   string

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "taskledger",
		Short: "A CLI tool to track work and generate reports from a YAML log.",
		Long:  `TaskLedger is a command-line interface for parsing a work log YAML file to calculate hours worked and generate status reports.`,
	}

	// hoursCmd represents the hours command
	hoursCmd = &cobra.Command{
		Use:   "hours",
		Short: "Calculate total hours worked.",
		Long:  `Calculates the total work hours based on the start and end times in the worklog.yml file. You can specify a single date or a range.`,
		Run:   runHoursCommand,
	}

	// reportCmd represents the report command
	reportCmd = &cobra.Command{
		Use:   "report",
		Short: "Generate a human-readable work report.",
		Long:  `Generates a formatted text report detailing completed tasks, blockers, and ongoing work for the specified date or date range.`,
		Run:   runReportCommand,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Errors from commands are now handled by slog, so we just exit.
		os.Exit(1)
	}
}

func init() {
	// Add persistent flags to the root command (available to all subcommands)
	rootCmd.PersistentFlags().StringVar(&filePath, "file", "worklog.yml", "Path to the YAML work log file.")

	// Add local flags to the 'hours' command
	hoursCmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD).")
	hoursCmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD).")

	// Add local flags to the 'report' command
	reportCmd.Flags().StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD).")
	reportCmd.Flags().StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD).")

	// Add subcommands to the root command
	rootCmd.AddCommand(hoursCmd)
	rootCmd.AddCommand(reportCmd)
}

// --- Main Application Entry Point ---

func main() {
	// Setup structured JSON logger for errors.
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	Execute()
}

// --- Command Execution Logic ---

func runHoursCommand(cmd *cobra.Command, args []string) {
	workData, err := loadWorkData(filePath)
	if err != nil {
		slog.Error("failed to load work log file", "error", err, "path", filePath)
		os.Exit(1)
	}

	dates, err := getDatesInRange(workData, startDate, endDate)
	if err != nil {
		slog.Error("failed to process date range", "error", err, "start_date", startDate, "end_date", endDate)
		os.Exit(1)
	}

	var totalDuration time.Duration
	for _, date := range dates {
		dailyLog, exists := workData[date]
		if !exists {
			continue
		}
		for _, logEntry := range dailyLog.WorkLogEntries {
			start, err1 := time.Parse("15:04", logEntry.StartTime)
			end, err2 := time.Parse("15:04", logEntry.EndTime)
			if err1 != nil || err2 != nil {
				slog.Warn("could not parse time entry, skipping", "date", date, "entry", logEntry)
				continue
			}
			totalDuration += end.Sub(start)
		}
	}

	// Print the output as human-readable text
	cmd.Printf("Total hours worked from %s to %s: %.2f\n", dates[0], dates[len(dates)-1], totalDuration.Hours())
}

func runReportCommand(cmd *cobra.Command, args []string) {
	workData, err := loadWorkData(filePath)
	if err != nil {
		slog.Error("failed to load work log file", "error", err, "path", filePath)
		os.Exit(1)
	}

	dates, err := getDatesInRange(workData, startDate, endDate)
	if err != nil {
		slog.Error("failed to process date range", "error", err, "start_date", startDate, "end_date", endDate)
		os.Exit(1)
	}

	// Categorize tasks with better logic
	completedTasks := make(map[string][]TaskWithDate) // Jira ticket -> list of tasks with dates
	allNextUpTasks := make(map[string][]TaskWithDate) // Jira ticket -> list of tasks with next up descriptions
	mostRecentTasks := make(map[string]TaskWithDate)  // Jira ticket -> most recent task (for blockers and filtering)

	for _, date := range dates {
		dailyLog, exists := workData[date]
		if !exists {
			continue
		}
		for _, task := range dailyLog.Tasks {
			taskWithDate := TaskWithDate{Task: task, Date: date}

			// Keep the original Jira ticket field (full URL)
			jiraTicket := task.JiraTicket

			// Track completed tasks - include both completed and in-progress tasks with descriptions (actual work done)
			if strings.EqualFold(task.Status, "completed") ||
				(strings.EqualFold(task.Status, "in progress") && task.Description != "") {
				completedTasks[jiraTicket] = append(completedTasks[jiraTicket], taskWithDate)
			}

			// Collect all tasks with upnext descriptions (we'll filter by most recent status later)
			if task.UpnextDescription != "" {
				allNextUpTasks[jiraTicket] = append(allNextUpTasks[jiraTicket], taskWithDate)
			}

			// Track most recent task per Jira ticket (for blockers and filtering)
			if jiraTicket != "" {
				if existing, exists := mostRecentTasks[jiraTicket]; !exists || date > existing.Date {
					mostRecentTasks[jiraTicket] = taskWithDate
				}
			}
		}
	}

	// Filter next up tasks: only include tickets where the most recent task is still in progress or not started
	nextUpTasks := make(map[string][]TaskWithDate)
	for jiraTicket, taskList := range allNextUpTasks {
		if mostRecent, exists := mostRecentTasks[jiraTicket]; exists {
			if strings.EqualFold(mostRecent.Status, "in progress") || strings.EqualFold(mostRecent.Status, "not started") {
				nextUpTasks[jiraTicket] = taskList
			}
		}
	}

	// Filter blocked tasks: only include tickets where the most recent task has a blocker
	var blockedTasks []Task
	for _, taskWithDate := range mostRecentTasks {
		if taskWithDate.Blocker != "" {
			blockedTasks = append(blockedTasks, taskWithDate.Task)
		}
	}

	// Generate and print the human-readable report to standard output
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Work Report (%s to %s)\n", dates[0], dates[len(dates)-1])
	fmt.Fprintln(out, "=======Autogenerated by TaskLedger=======")

	printCompletedTasks(out, completedTasks)
	printNextUpTasks(out, nextUpTasks)
	printBlockedTasks(out, blockedTasks)
}

// --- Helper Functions ---

func loadWorkData(filePath string) (WorkData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s': %w", filePath, err)
	}

	var workData WorkData
	err = yaml.Unmarshal(data, &workData)
	if err != nil {
		safeData, _ := json.Marshal(string(data))
		return nil, fmt.Errorf("could not parse YAML from '%s': %w. Content: %s", filePath, err, safeData)
	}

	return workData, nil
}

func getDatesInRange(workData WorkData, startStr, endStr string) ([]string, error) {
	if startStr != "" && endStr == "" {
		endStr = startStr
	}
	if endStr != "" && startStr == "" {
		startStr = endStr
	}

	if startStr == "" && endStr == "" {
		var allDates []string
		for date := range workData {
			allDates = append(allDates, date)
		}
		sort.Strings(allDates)
		if len(allDates) == 0 {
			return nil, fmt.Errorf("no data found in the work log file")
		}
		return allDates, nil
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format, use YYYY-MM-DD: %w", err)
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format, use YYYY-MM-DD: %w", err)
	}

	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date cannot be before start date")
	}

	var datesInRange []string
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if _, exists := workData[dateStr]; exists {
			datesInRange = append(datesInRange, dateStr)
		}
	}

	if len(datesInRange) == 0 {
		return nil, fmt.Errorf("no data found for the specified date range")
	}
	sort.Strings(datesInRange)
	return datesInRange, nil
}

// --- Report Printing Functions ---

func printCompletedTasks(out io.Writer, tasks map[string][]TaskWithDate) {
	if len(tasks) == 0 {
		return
	}
	fmt.Fprintln(out, "\n🦀 Thing I've been working on")

	var tickets []string
	for t := range tasks {
		tickets = append(tickets, t)
	}
	sort.Strings(tickets)

	for _, ticket := range tickets {
		taskList := tasks[ticket]

		// Sort tasks chronologically (oldest to newest)
		sort.Slice(taskList, func(i, j int) bool {
			return taskList[i].Date < taskList[j].Date
		})

		// Group tasks by Jira ticket and consolidate
		if ticket != "" {
			// Print the Jira ticket header
			fmt.Fprintf(out, "    • %s: \n", ticket)

			// Collect all descriptions and unique PR links
			var descriptions []string
			prLinks := make(map[string]bool)

			for _, taskWithDate := range taskList {
				if taskWithDate.Description != "" {
					descriptions = append(descriptions, taskWithDate.Description)
				}
				if taskWithDate.GithubPR != "" {
					prLinks[taskWithDate.GithubPR] = true
				}
			}

			// Print all descriptions
			for _, desc := range descriptions {
				fmt.Fprintf(out, "        ◦ %s", desc)
				// If there are PR links, add them after the last description
				if len(descriptions) > 0 && desc == descriptions[len(descriptions)-1] && len(prLinks) > 0 {
					var links []string
					for link := range prLinks {
						links = append(links, link)
					}
					sort.Strings(links)
					output := fmt.Sprintf("\n        ◦ PR(s): %s", strings.Join(links, "; "))
					fmt.Fprint(out, output)
				}
				fmt.Fprintln(out)
			}
		} else {
			// Handle tasks without Jira tickets
			for _, taskWithDate := range taskList {
				if taskWithDate.GithubPR != "" {
					fmt.Fprintf(out, "    • %s\n", taskWithDate.Description)
					fmt.Fprintf(out, "        ◦ PR(s): %s\n", taskWithDate.GithubPR)
				} else {
					fmt.Fprintf(out, "    • %s\n", taskWithDate.Description)
				}
			}
		}
	}
}

func printNextUpTasks(out io.Writer, nextUp map[string][]TaskWithDate) {
	if len(nextUp) == 0 {
		return
	}
	fmt.Fprintln(out, "\n:starfleet: Thing I plan on working on next")

	// Sort tickets alphabetically
	var tickets []string
	for ticket := range nextUp {
		tickets = append(tickets, ticket)
	}
	sort.Strings(tickets)

	for _, ticket := range tickets {
		taskList := nextUp[ticket]

		// Sort tasks chronologically (oldest to newest)
		sort.Slice(taskList, func(i, j int) bool {
			return taskList[i].Date < taskList[j].Date
		})

		if ticket != "" {
			fmt.Fprintf(out, "    • %s\n", ticket)

			// For next up tasks, only use the most recent entry per ticket
			// Get the most recent task with an upnext description
			var mostRecentDesc string
			prLinks := make(map[string]bool)

			// Work backwards to find the most recent upnext description
			for i := len(taskList) - 1; i >= 0; i-- {
				taskWithDate := taskList[i]
				if mostRecentDesc == "" {
					if taskWithDate.UpnextDescription != "" {
						mostRecentDesc = taskWithDate.UpnextDescription
					} else if taskWithDate.Description != "" {
						mostRecentDesc = taskWithDate.Description
					}
				}
				// Collect all unique PR links
				if taskWithDate.GithubPR != "" {
					prLinks[taskWithDate.GithubPR] = true
				}
			}

			// Print the most recent description
			if mostRecentDesc != "" {
				fmt.Fprintf(out, "        ◦ %s", mostRecentDesc)
				// Add PR links if any exist
				if len(prLinks) > 0 {
					var links []string
					for link := range prLinks {
						links = append(links, link)
					}
					sort.Strings(links)
					output := fmt.Sprintf("\n        ◦ PR(s): %s", strings.Join(links, "; "))
					fmt.Fprint(out, output)
				}
				fmt.Fprintln(out)
			}
		} else {
			// Handle tasks without Jira tickets - use most recent entry
			if len(taskList) > 0 {
				taskWithDate := taskList[len(taskList)-1] // Get most recent
				var desc string
				if taskWithDate.UpnextDescription != "" {
					desc = taskWithDate.UpnextDescription
				} else {
					desc = taskWithDate.Description
				}

				if taskWithDate.GithubPR != "" {
					fmt.Fprintf(out, "    • %s\n", desc)
					fmt.Fprintf(out, "        ◦ PR(s): %s\n", taskWithDate.GithubPR)
				} else {
					fmt.Fprintf(out, "    • %s\n", desc)
				}
			}
		}
	}
}

func printBlockedTasks(out io.Writer, blocked []Task) {
	if len(blocked) == 0 {
		return
	}
	fmt.Fprintln(out, "\n:facepalm: Thing that is blocking me or that I could use some help / discussion about")
	for _, task := range blocked {
		fmt.Fprintf(out, "    • %s \n", task.JiraTicket)
		fmt.Fprintf(out, "        ◦ Blocker: %s\n", task.Blocker)
	}
}
