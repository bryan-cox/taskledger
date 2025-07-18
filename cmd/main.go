package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
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

// --- JIRA Integration ---

// JiraTicketInfo holds information about a JIRA ticket
type JiraTicketInfo struct {
	Key     string
	Summary string
	URL     string
}

// JiraAPIResponse represents the response from JIRA API
type JiraAPIResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

// JIRA ticket ID regex patterns
var (
	jiraTicketRegex = regexp.MustCompile(`\b([A-Z]+-\d+)\b`)
	jiraURLRegex    = regexp.MustCompile(`https://issues\.redhat\.com/browse/([A-Z]+-\d+)`)
)

// extractJiraTicketID extracts JIRA ticket ID from URL or text
func extractJiraTicketID(input string) string {
	// First try to extract from URL
	if matches := jiraURLRegex.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1]
	}

	// Then try to extract from plain text
	if matches := jiraTicketRegex.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// fetchJiraTicketSummary fetches the summary of a JIRA ticket using the API
func fetchJiraTicketSummary(ticketID string) (JiraTicketInfo, error) {
	ticket := JiraTicketInfo{
		Key: ticketID,
		URL: fmt.Sprintf("https://issues.redhat.com/browse/%s", ticketID),
	}

	// Check if JIRA Personal Access Token is available
	jiraPAT := os.Getenv("JIRA_PAT")
	if jiraPAT == "" {
		// Return ticket info without summary if no PAT is available
		return ticket, nil
	}

	// Make API request to fetch ticket summary
	apiURL := fmt.Sprintf("https://issues.redhat.com/rest/api/2/issue/%s?fields=summary", ticketID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return ticket, fmt.Errorf("failed to create request: %w", err)
	}

	// Set authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jiraPAT))
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ticket, fmt.Errorf("failed to fetch ticket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ticket, fmt.Errorf("JIRA API returned status %d", resp.StatusCode)
	}

	var jiraResp JiraAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&jiraResp); err != nil {
		return ticket, fmt.Errorf("failed to decode response: %w", err)
	}

	ticket.Summary = jiraResp.Fields.Summary
	return ticket, nil
}

// processJiraTickets processes a map of JIRA tickets and fetches their summaries
func processJiraTickets(tickets map[string][]TaskWithDate) map[string]JiraTicketInfo {
	jiraInfo := make(map[string]JiraTicketInfo)

	for ticketReference := range tickets {
		if ticketReference == "" {
			continue
		}

		ticketID := extractJiraTicketID(ticketReference)
		if ticketID == "" {
			continue
		}

		if _, exists := jiraInfo[ticketID]; !exists {
			// Fetch ticket info (will include summary only if JIRA_PAT is available)
			if info, err := fetchJiraTicketSummary(ticketID); err == nil {
				jiraInfo[ticketID] = info
			} else {
				// If fetch fails, still create basic info
				jiraInfo[ticketID] = JiraTicketInfo{
					Key: ticketID,
					URL: fmt.Sprintf("https://issues.redhat.com/browse/%s", ticketID),
				}
				slog.Warn("failed to fetch JIRA ticket summary", "ticket", ticketID, "error", err)
			}
		}
	}

	return jiraInfo
}

// formatJiraTicketHTML formats a JIRA ticket reference as HTML with optional summary
func formatJiraTicketHTML(ticketReference string, jiraInfo map[string]JiraTicketInfo) string {
	ticketID := extractJiraTicketID(ticketReference)
	if ticketID == "" {
		// No JIRA ticket found, return escaped original text
		return html.EscapeString(ticketReference)
	}

	info, exists := jiraInfo[ticketID]
	if !exists {
		// Fallback: create basic link
		url := fmt.Sprintf("https://issues.redhat.com/browse/%s", ticketID)
		return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, url, html.EscapeString(ticketID))
	}

	// Create link with summary if available
	linkText := info.Key
	if info.Summary != "" {
		linkText = fmt.Sprintf("%s: %s", info.Key, info.Summary)
	}

	return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, info.URL, html.EscapeString(linkText))
}

// --- Cobra Command Definitions ---

var (
	// Used for flags.
	filePath  string
	startDate string
	endDate   string
	copyHTML  bool   // Flag for attempting to copy HTML to clipboard
	htmlFile  string // Flag for saving HTML to file
	showHTML  bool   // Flag for displaying HTML content
	openHTML  bool   // Flag for automatically opening HTML file in browser

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
	reportCmd.Flags().BoolVar(&copyHTML, "copy-html", false, "Attempt to copy the report as formatted HTML to clipboard.")
	reportCmd.Flags().StringVar(&htmlFile, "html-file", "", "Save the report as HTML to the specified file.")
	reportCmd.Flags().BoolVar(&showHTML, "show-html", false, "Display the HTML content in the terminal.")
	reportCmd.Flags().BoolVar(&openHTML, "open-html", false, "Automatically open the HTML file in the default browser after saving.")

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

	// Handle HTML output options
	if copyHTML || htmlFile != "" || showHTML || openHTML {
		htmlContent := generateHTMLReport(dates, completedTasks, nextUpTasks, blockedTasks)

		// Save to file if requested
		if htmlFile != "" {
			err := saveHTMLToFile(htmlContent, htmlFile)
			if err != nil {
				slog.Error("failed to save HTML to file", "error", err, "file", htmlFile)
			} else {
				fmt.Fprintf(out, "\n✅ HTML report saved to: %s\n", htmlFile)

				// Open HTML file in browser if requested
				if openHTML {
					err := openHTMLInBrowser(htmlFile)
					if err != nil {
						fmt.Fprintf(out, "⚠️  Failed to open HTML file in browser: %v\n", err)
					} else {
						fmt.Fprintf(out, "🌐 Opened HTML report in default browser\n")
					}
				}
			}
		} else if openHTML {
			// If openHTML is requested but no file is specified, show a helpful message
			fmt.Fprintf(out, "\n💡 To use --open-html, you must also specify --html-file\n")
		}

		// Show HTML in console if requested
		if showHTML {
			fmt.Fprintln(out, "\n=== HTML OUTPUT ===")
			fmt.Fprintln(out, htmlContent)
			fmt.Fprintln(out, "=== END HTML OUTPUT ===")
		}

		// Try to copy to clipboard if requested
		if copyHTML {
			err := copyHTMLToClipboard(htmlContent)
			if err != nil {
				fmt.Fprintf(out, "\n⚠️  Failed to copy to clipboard: %v\n", err)
				fmt.Fprintf(out, "💡 Try using --html-file to save to a file instead, or --show-html to display the HTML\n")
			} else {
				fmt.Fprintln(out, "\n✅ HTML report copied to clipboard!")
			}
		}
	}
}

// saveHTMLToFile saves HTML content to a file
func saveHTMLToFile(htmlContent, filename string) error {
	return os.WriteFile(filename, []byte(htmlContent), 0644)
}

// copyHTMLToClipboard attempts to copy HTML to clipboard using system commands
func copyHTMLToClipboard(htmlContent string) error {
	switch runtime.GOOS {
	case "linux":
		return copyHTMLLinux(htmlContent)
	case "darwin":
		return copyHTMLMacOS(htmlContent)
	case "windows":
		return copyHTMLWindows(htmlContent)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func copyHTMLLinux(htmlContent string) error {
	// Try different clipboard tools in order of preference
	htmlTools := [][]string{
		{"wl-copy", "--type", "text/html"},                        // Wayland HTML
		{"xclip", "-selection", "clipboard", "-t", "text/html"},   // X11 HTML
		{"xsel", "--clipboard", "--input", "--type", "text/html"}, // X11 alternative HTML
	}

	for _, tool := range htmlTools {
		if isCommandAvailable(tool[0]) {
			cmd := exec.Command(tool[0], tool[1:]...)
			cmd.Stdin = strings.NewReader(htmlContent)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	// Fallback: try to copy as plain text
	textTools := [][]string{
		{"wl-copy"},                          // Wayland
		{"xclip", "-selection", "clipboard"}, // X11
		{"xsel", "--clipboard", "--input"},   // X11 alternative
	}

	for _, tool := range textTools {
		if isCommandAvailable(tool[0]) {
			cmd := exec.Command(tool[0], tool[1:]...)
			cmd.Stdin = strings.NewReader(htmlContent)
			if err := cmd.Run(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no suitable clipboard tool found (tried: wl-copy, xclip, xsel)")
}

func copyHTMLMacOS(htmlContent string) error {
	// Use osascript to copy HTML with formatting
	script := fmt.Sprintf(`osascript -e 'set the clipboard to "%s" as «class HTML»'`,
		strings.ReplaceAll(htmlContent, `"`, `\"`))

	cmd := exec.Command("sh", "-c", script)
	return cmd.Run()
}

func copyHTMLWindows(htmlContent string) error {
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Clipboard]::SetText(@"
%s
"@, [System.Windows.Forms.TextDataFormat]::Html)`, htmlContent)

	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// openHTMLInBrowser opens the specified HTML file in the default browser
func openHTMLInBrowser(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "linux":
		cmd = exec.Command("xdg-open", filePath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// generateHTMLReport creates an HTML version of the report
func generateHTMLReport(dates []string, completedTasks map[string][]TaskWithDate, nextUpTasks map[string][]TaskWithDate, blockedTasks []Task) string {
	// Process JIRA tickets and fetch summaries
	allTickets := make(map[string][]TaskWithDate)

	// Combine all ticket references
	for ticket, tasks := range completedTasks {
		allTickets[ticket] = tasks
	}
	for ticket, tasks := range nextUpTasks {
		allTickets[ticket] = tasks
	}
	for _, task := range blockedTasks {
		if task.JiraTicket != "" {
			allTickets[task.JiraTicket] = []TaskWithDate{{Task: task}}
		}
	}

	jiraInfo := processJiraTickets(allTickets)

	var htmlBuilder strings.Builder

	// Simplified HTML header for better Slack compatibility
	htmlBuilder.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body>`)

	// Title
	htmlBuilder.WriteString(fmt.Sprintf(`<h1>Work Report (%s to %s)</h1>`, dates[0], dates[len(dates)-1]))
	htmlBuilder.WriteString(`<p><em>Autogenerated by TaskLedger</em></p>`)

	// Completed Tasks Section - using simple list structure for Slack compatibility
	if len(completedTasks) > 0 {
		htmlBuilder.WriteString(`<h2>🦀 Things I've been working on</h2>`)
		htmlBuilder.WriteString(`<ul>`)

		var tickets []string
		for t := range completedTasks {
			tickets = append(tickets, t)
		}
		sort.Strings(tickets)

		for _, ticket := range tickets {
			taskList := completedTasks[ticket]
			sort.Slice(taskList, func(i, j int) bool {
				return taskList[i].Date < taskList[j].Date
			})

			if ticket != "" {
				htmlBuilder.WriteString(fmt.Sprintf(`<li><strong>%s</strong>`, formatJiraTicketHTML(ticket, jiraInfo)))

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

				htmlBuilder.WriteString(`<ul>`)
				for _, desc := range descriptions {
					htmlBuilder.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(desc)))
				}

				if len(prLinks) > 0 {
					var links []string
					for link := range prLinks {
						links = append(links, link)
					}
					sort.Strings(links)

					htmlBuilder.WriteString(`<li>PR(s): `)
					for i, link := range links {
						if i > 0 {
							htmlBuilder.WriteString("; ")
						}
						htmlBuilder.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(link), html.EscapeString(link)))
					}
					htmlBuilder.WriteString(`</li>`)
				}
				htmlBuilder.WriteString(`</ul></li>`)
			} else {
				htmlBuilder.WriteString(`<li>`)
				htmlBuilder.WriteString(`<ul>`)
				for _, taskWithDate := range taskList {
					htmlBuilder.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(taskWithDate.Description)))
					if taskWithDate.GithubPR != "" {
						htmlBuilder.WriteString(fmt.Sprintf(`<li>PR: <a href="%s">%s</a></li>`, html.EscapeString(taskWithDate.GithubPR), html.EscapeString(taskWithDate.GithubPR)))
					}
				}
				htmlBuilder.WriteString(`</ul></li>`)
			}
		}
		htmlBuilder.WriteString(`</ul>`)
	}

	// Next Up Tasks Section
	if len(nextUpTasks) > 0 {
		htmlBuilder.WriteString(`<h2>⭐ Things I plan on working on next</h2>`)
		htmlBuilder.WriteString(`<ul>`)

		var tickets []string
		for ticket := range nextUpTasks {
			tickets = append(tickets, ticket)
		}
		sort.Strings(tickets)

		for _, ticket := range tickets {
			taskList := nextUpTasks[ticket]
			sort.Slice(taskList, func(i, j int) bool {
				return taskList[i].Date < taskList[j].Date
			})

			if ticket != "" {
				htmlBuilder.WriteString(fmt.Sprintf(`<li><strong>%s</strong>`, formatJiraTicketHTML(ticket, jiraInfo)))

				var mostRecentDesc string
				prLinks := make(map[string]bool)

				for i := len(taskList) - 1; i >= 0; i-- {
					taskWithDate := taskList[i]
					if mostRecentDesc == "" {
						if taskWithDate.UpnextDescription != "" {
							mostRecentDesc = taskWithDate.UpnextDescription
						} else if taskWithDate.Description != "" {
							mostRecentDesc = taskWithDate.Description
						}
					}
					if taskWithDate.GithubPR != "" {
						prLinks[taskWithDate.GithubPR] = true
					}
				}

				htmlBuilder.WriteString(`<ul>`)
				if mostRecentDesc != "" {
					htmlBuilder.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(mostRecentDesc)))
				}

				if len(prLinks) > 0 {
					var links []string
					for link := range prLinks {
						links = append(links, link)
					}
					sort.Strings(links)

					htmlBuilder.WriteString(`<li>PR(s): `)
					for i, link := range links {
						if i > 0 {
							htmlBuilder.WriteString("; ")
						}
						htmlBuilder.WriteString(fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(link), html.EscapeString(link)))
					}
					htmlBuilder.WriteString(`</li>`)
				}
				htmlBuilder.WriteString(`</ul></li>`)
			} else {
				if len(taskList) > 0 {
					taskWithDate := taskList[len(taskList)-1]
					var desc string
					if taskWithDate.UpnextDescription != "" {
						desc = taskWithDate.UpnextDescription
					} else {
						desc = taskWithDate.Description
					}

					htmlBuilder.WriteString(`<li>`)
					htmlBuilder.WriteString(`<ul>`)
					htmlBuilder.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(desc)))
					if taskWithDate.GithubPR != "" {
						htmlBuilder.WriteString(fmt.Sprintf(`<li>PR: <a href="%s">%s</a></li>`, html.EscapeString(taskWithDate.GithubPR), html.EscapeString(taskWithDate.GithubPR)))
					}
					htmlBuilder.WriteString(`</ul></li>`)
				}
			}
		}
		htmlBuilder.WriteString(`</ul>`)
	}

	// Blocked Tasks Section
	if len(blockedTasks) > 0 {
		htmlBuilder.WriteString(`<h2>🚫 Things that are blocking me</h2>`)
		htmlBuilder.WriteString(`<ul>`)

		for _, task := range blockedTasks {
			htmlBuilder.WriteString(fmt.Sprintf(`<li><strong>%s</strong>`, formatJiraTicketHTML(task.JiraTicket, jiraInfo)))
			htmlBuilder.WriteString(`<ul>`)
			htmlBuilder.WriteString(fmt.Sprintf(`<li>Blocker: %s</li>`, html.EscapeString(task.Blocker)))
			htmlBuilder.WriteString(`</ul></li>`)
		}
		htmlBuilder.WriteString(`</ul>`)
	}

	htmlBuilder.WriteString(`</body></html>`)
	return htmlBuilder.String()
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
