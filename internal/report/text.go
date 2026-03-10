package report

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bryan-cox/taskledger/internal/model"
)

// Section headers for text output (Slack-compatible emoji codes).
const (
	TextHeaderCompleted      = "\n🦀 Thing I've been working on"
	TextHeaderNextUp         = "\n:starfleet: Thing I plan on working on next"
	TextHeaderBlocked        = "\n:facepalm: Thing that is blocking me or that I could use some help / discussion about"
	textNonFeatureWorkHeader = "Non-feature work"
)

// PrintCompletedTasks prints the completed tasks section to the writer.
func PrintCompletedTasks(out io.Writer, tasks map[string][]model.TaskWithDate) {
	if len(tasks) == 0 {
		return
	}
	fmt.Fprintln(out, TextHeaderCompleted)

	// Separate feature work and non-feature work
	var featureTickets []string
	var nonFeatureTickets []string

	for ticket := range tasks {
		taskList := tasks[ticket]
		// Check if any task in the group has a PR (for NO-JIRA check)
		hasPR := false
		for _, t := range taskList {
			if t.GithubPR != "" {
				hasPR = true
				break
			}
		}

		prArg := ""
		if hasPR {
			prArg = "has-pr"
		}

		if IsNonFeatureWork(ticket, prArg) {
			nonFeatureTickets = append(nonFeatureTickets, ticket)
		} else {
			featureTickets = append(featureTickets, ticket)
		}
	}
	sort.Strings(featureTickets)
	sort.Strings(nonFeatureTickets)

	// Print feature work first
	for _, ticket := range featureTickets {
		printTicketEntry(out, ticket, tasks[ticket])
	}

	// Print non-feature work at the end (grouped under "Non-feature work" with sub-entries)
	if len(nonFeatureTickets) > 0 {
		fmt.Fprintf(out, "    • %s: \n", textNonFeatureWorkHeader)
		for _, ticket := range nonFeatureTickets {
			printNonFeatureSubEntry(out, ticket, tasks[ticket])
		}
	}
}

// printTicketEntry prints a single ticket entry with its descriptions and PRs.
func printTicketEntry(out io.Writer, ticket string, taskList []model.TaskWithDate) {
	// Sort tasks chronologically (oldest to newest)
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Date < taskList[j].Date
	})

	// Print the Jira ticket header
	fmt.Fprintf(out, "    • %s: \n", ticket)

	// Collect all descriptions and unique PR links
	var descriptions []string
	prLinks := make(map[string]bool)

	for _, taskWithDate := range taskList {
		descriptions = append(descriptions, taskWithDate.GetDescriptions()...)
		if taskWithDate.GithubPR != "" {
			prLinks[taskWithDate.GithubPR] = true
		}
	}

	// Print all descriptions (deduplicated)
	descriptions = deduplicateDescriptions(descriptions)
	for _, desc := range descriptions {
		fmt.Fprintf(out, "        ◦ %s\n", desc)
	}

	// Print PR links
	if len(prLinks) > 0 {
		var links []string
		for link := range prLinks {
			links = append(links, link)
		}
		sort.Strings(links)
		fmt.Fprintf(out, "        ◦ PR(s): %s\n", strings.Join(links, "; "))
	}
}

// printNonFeatureSubEntry prints a non-feature work sub-entry with ticket name as header.
func printNonFeatureSubEntry(out io.Writer, ticket string, taskList []model.TaskWithDate) {
	// Sort tasks chronologically (oldest to newest)
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Date < taskList[j].Date
	})

	// Collect all descriptions and unique PR links
	var descriptions []string
	prLinks := make(map[string]bool)

	for _, taskWithDate := range taskList {
		descriptions = append(descriptions, taskWithDate.GetDescriptions()...)
		if taskWithDate.GithubPR != "" {
			prLinks[taskWithDate.GithubPR] = true
		}
	}

	// Determine header: for synthetic keys, use the first description
	header := ticket
	if IsSyntheticKey(ticket) || header == "" {
		if len(descriptions) > 0 {
			header = descriptions[0]
			descriptions = descriptions[1:]
		} else {
			header = "Misc"
		}
	}
	fmt.Fprintf(out, "        ◦ %s\n", header)

	// Print remaining descriptions (third-level indent), deduplicated and sorted
	descriptions = deduplicateDescriptions(descriptions)
	sort.Strings(descriptions)
	for _, desc := range descriptions {
		fmt.Fprintf(out, "            ▪ %s\n", desc)
	}

	// Print PR links
	if len(prLinks) > 0 {
		var links []string
		for link := range prLinks {
			links = append(links, link)
		}
		sort.Strings(links)
		fmt.Fprintf(out, "            ▪ PR(s): %s\n", strings.Join(links, "; "))
	}
}

// PrintNextUpTasks prints the next up tasks section to the writer.
func PrintNextUpTasks(out io.Writer, nextUp map[string][]model.TaskWithDate) {
	if len(nextUp) == 0 {
		return
	}
	fmt.Fprintln(out, TextHeaderNextUp)

	// Separate feature work and non-feature work
	var featureTickets []string
	var nonFeatureTickets []string

	for ticket := range nextUp {
		taskList := nextUp[ticket]
		// Check if any task in the group has a PR (for NO-JIRA check)
		hasPR := false
		for _, t := range taskList {
			if t.GithubPR != "" {
				hasPR = true
				break
			}
		}

		prArg := ""
		if hasPR {
			prArg = "has-pr"
		}

		if IsNonFeatureWork(ticket, prArg) {
			nonFeatureTickets = append(nonFeatureTickets, ticket)
		} else {
			featureTickets = append(featureTickets, ticket)
		}
	}
	sort.Strings(featureTickets)
	sort.Strings(nonFeatureTickets)

	// Print feature work first
	for _, ticket := range featureTickets {
		printNextUpTicketEntry(out, ticket, nextUp[ticket])
	}

	// Print non-feature work at the end (grouped under "Non-feature work" with sub-entries)
	if len(nonFeatureTickets) > 0 {
		fmt.Fprintf(out, "    • %s\n", textNonFeatureWorkHeader)
		for _, ticket := range nonFeatureTickets {
			printNonFeatureNextUpSubEntry(out, ticket, nextUp[ticket])
		}
	}
}

// printNextUpTicketEntry prints a single next up ticket entry.
func printNextUpTicketEntry(out io.Writer, ticket string, taskList []model.TaskWithDate) {
	// Sort tasks chronologically (oldest to newest)
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Date < taskList[j].Date
	})

	fmt.Fprintf(out, "    • %s\n", ticket)

	// For next up tasks, only use the most recent entry per ticket
	var mostRecentDesc string
	prLinks := make(map[string]bool)

	// Work backwards to find the most recent upnext description
	for i := len(taskList) - 1; i >= 0; i-- {
		taskWithDate := taskList[i]
		if mostRecentDesc == "" {
			if taskWithDate.UpnextDescription != "" {
				mostRecentDesc = taskWithDate.UpnextDescription
			} else {
				allDescs := taskWithDate.GetDescriptions()
				if len(allDescs) > 0 {
					mostRecentDesc = allDescs[len(allDescs)-1]
				}
			}
		}
		if taskWithDate.GithubPR != "" {
			prLinks[taskWithDate.GithubPR] = true
		}
	}

	// Print the most recent description
	if mostRecentDesc != "" {
		fmt.Fprintf(out, "        ◦ %s\n", mostRecentDesc)
	}

	// Print PR links
	if len(prLinks) > 0 {
		var links []string
		for link := range prLinks {
			links = append(links, link)
		}
		sort.Strings(links)
		fmt.Fprintf(out, "        ◦ PR(s): %s\n", strings.Join(links, "; "))
	}
}

// printNonFeatureNextUpSubEntry prints a non-feature next up sub-entry.
func printNonFeatureNextUpSubEntry(out io.Writer, ticket string, taskList []model.TaskWithDate) {
	// Sort tasks chronologically (oldest to newest)
	sort.Slice(taskList, func(i, j int) bool {
		return taskList[i].Date < taskList[j].Date
	})

	// For next up tasks, only use the most recent entry per ticket
	var mostRecentDesc string
	prLinks := make(map[string]bool)

	// Work backwards to find the most recent upnext description
	for i := len(taskList) - 1; i >= 0; i-- {
		taskWithDate := taskList[i]
		if mostRecentDesc == "" {
			if taskWithDate.UpnextDescription != "" {
				mostRecentDesc = taskWithDate.UpnextDescription
			} else {
				allDescs := taskWithDate.GetDescriptions()
				if len(allDescs) > 0 {
					mostRecentDesc = allDescs[len(allDescs)-1]
				}
			}
		}
		if taskWithDate.GithubPR != "" {
			prLinks[taskWithDate.GithubPR] = true
		}
	}

	// Determine header: for synthetic keys, use the upnext description or first task description
	header := ticket
	if IsSyntheticKey(ticket) || header == "" {
		if mostRecentDesc != "" {
			header = mostRecentDesc
			mostRecentDesc = "" // Don't duplicate as a nested bullet
		} else {
			header = "Misc"
		}
	}
	fmt.Fprintf(out, "        ◦ %s\n", header)

	// Print the most recent description (third-level indent)
	if mostRecentDesc != "" {
		fmt.Fprintf(out, "            ▪ %s\n", mostRecentDesc)
	}

	// Print PR links
	if len(prLinks) > 0 {
		var links []string
		for link := range prLinks {
			links = append(links, link)
		}
		sort.Strings(links)
		fmt.Fprintf(out, "            ▪ PR(s): %s\n", strings.Join(links, "; "))
	}
}

// PrintBlockedTasks prints the blocked tasks section to the writer.
func PrintBlockedTasks(out io.Writer, blocked []model.Task) {
	if len(blocked) == 0 {
		return
	}

	// Separate feature work and non-feature work
	var featureTasks []model.Task
	var nonFeatureTasks []model.Task

	for _, task := range blocked {
		if IsNonFeatureWork(task.JiraTicket, task.GithubPR) {
			nonFeatureTasks = append(nonFeatureTasks, task)
		} else {
			featureTasks = append(featureTasks, task)
		}
	}

	fmt.Fprintln(out, TextHeaderBlocked)

	// Print feature work first
	for _, task := range featureTasks {
		fmt.Fprintf(out, "    • %s \n", task.JiraTicket)
		fmt.Fprintf(out, "        ◦ Blocker: %s\n", task.Blocker)
	}

	// Print non-feature work at the end (grouped under "Non-feature work" with sub-entries)
	if len(nonFeatureTasks) > 0 {
		fmt.Fprintf(out, "    • %s \n", textNonFeatureWorkHeader)
		for _, task := range nonFeatureTasks {
			header := task.JiraTicket
			if header == "" {
				header = "Misc"
			}
			fmt.Fprintf(out, "        ◦ %s\n", header)
			fmt.Fprintf(out, "            ▪ Blocker: %s\n", task.Blocker)
		}
	}
}
