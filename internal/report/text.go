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
	var nonFeatureDescriptions []string
	nonFeaturePRLinks := make(map[string]bool)

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

		if IsNonFeatureWork(ticket, func() string {
			if hasPR {
				return "has-pr"
			}
			return ""
		}()) {
			// Collect non-feature work descriptions and PRs
			for _, taskWithDate := range taskList {
				nonFeatureDescriptions = append(nonFeatureDescriptions, taskWithDate.GetDescriptions()...)
				if taskWithDate.GithubPR != "" {
					nonFeaturePRLinks[taskWithDate.GithubPR] = true
				}
			}
		} else {
			featureTickets = append(featureTickets, ticket)
		}
	}
	sort.Strings(featureTickets)

	// Print feature work first
	for _, ticket := range featureTickets {
		taskList := tasks[ticket]

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
	}

	// Print non-feature work at the end
	if len(nonFeatureDescriptions) > 0 {
		fmt.Fprintf(out, "    • %s: \n", textNonFeatureWorkHeader)
		for i, desc := range nonFeatureDescriptions {
			fmt.Fprintf(out, "        ◦ %s", desc)
			// If there are PR links, add them after the last description
			if i == len(nonFeatureDescriptions)-1 && len(nonFeaturePRLinks) > 0 {
				var links []string
				for link := range nonFeaturePRLinks {
					links = append(links, link)
				}
				sort.Strings(links)
				output := fmt.Sprintf("\n        ◦ PR(s): %s", strings.Join(links, "; "))
				fmt.Fprint(out, output)
			}
			fmt.Fprintln(out)
		}
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
	var nonFeatureDescriptions []string
	nonFeaturePRLinks := make(map[string]bool)

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

		if IsNonFeatureWork(ticket, func() string {
			if hasPR {
				return "has-pr"
			}
			return ""
		}()) {
			// Collect non-feature work descriptions (most recent per ticket group)
			sort.Slice(taskList, func(i, j int) bool {
				return taskList[i].Date < taskList[j].Date
			})
			for i := len(taskList) - 1; i >= 0; i-- {
				taskWithDate := taskList[i]
				var desc string
				if taskWithDate.UpnextDescription != "" {
					desc = taskWithDate.UpnextDescription
				} else {
					allDescs := taskWithDate.GetDescriptions()
					if len(allDescs) > 0 {
						desc = allDescs[len(allDescs)-1]
					}
				}
				if desc != "" {
					nonFeatureDescriptions = append(nonFeatureDescriptions, desc)
					break
				}
				if taskWithDate.GithubPR != "" {
					nonFeaturePRLinks[taskWithDate.GithubPR] = true
				}
			}
		} else {
			featureTickets = append(featureTickets, ticket)
		}
	}
	sort.Strings(featureTickets)

	// Print feature work first
	for _, ticket := range featureTickets {
		taskList := nextUp[ticket]

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
			fmt.Fprintf(out, "        ◦ %s", mostRecentDesc)
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
	}

	// Print non-feature work at the end
	if len(nonFeatureDescriptions) > 0 {
		fmt.Fprintf(out, "    • %s\n", textNonFeatureWorkHeader)
		for i, desc := range nonFeatureDescriptions {
			fmt.Fprintf(out, "        ◦ %s", desc)
			if i == len(nonFeatureDescriptions)-1 && len(nonFeaturePRLinks) > 0 {
				var links []string
				for link := range nonFeaturePRLinks {
					links = append(links, link)
				}
				sort.Strings(links)
				output := fmt.Sprintf("\n        ◦ PR(s): %s", strings.Join(links, "; "))
				fmt.Fprint(out, output)
			}
			fmt.Fprintln(out)
		}
	}
}

// PrintBlockedTasks prints the blocked tasks section to the writer.
func PrintBlockedTasks(out io.Writer, blocked []model.Task) {
	if len(blocked) == 0 {
		return
	}

	// Separate feature work and non-feature work
	var featureTasks []model.Task
	var nonFeatureBlockers []string

	for _, task := range blocked {
		if IsNonFeatureWork(task.JiraTicket, task.GithubPR) {
			nonFeatureBlockers = append(nonFeatureBlockers, task.Blocker)
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

	// Print non-feature work at the end
	if len(nonFeatureBlockers) > 0 {
		fmt.Fprintf(out, "    • %s \n", textNonFeatureWorkHeader)
		for _, blocker := range nonFeatureBlockers {
			fmt.Fprintf(out, "        ◦ Blocker: %s\n", blocker)
		}
	}
}
