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
	TextHeaderCompleted = "\n🦀 Thing I've been working on"
	TextHeaderNextUp    = "\n:starfleet: Thing I plan on working on next"
	TextHeaderBlocked   = "\n:facepalm: Thing that is blocking me or that I could use some help / discussion about"
)

// PrintCompletedTasks prints the completed tasks section to the writer.
func PrintCompletedTasks(out io.Writer, tasks map[string][]model.TaskWithDate) {
	if len(tasks) == 0 {
		return
	}
	fmt.Fprintln(out, TextHeaderCompleted)

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

// PrintNextUpTasks prints the next up tasks section to the writer.
func PrintNextUpTasks(out io.Writer, nextUp map[string][]model.TaskWithDate) {
	if len(nextUp) == 0 {
		return
	}
	fmt.Fprintln(out, TextHeaderNextUp)

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
					} else {
						// Get all descriptions and use the last one if available
						allDescs := taskWithDate.GetDescriptions()
						if len(allDescs) > 0 {
							mostRecentDesc = allDescs[len(allDescs)-1]
						}
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
					// Get all descriptions and use the last one if available
					allDescs := taskWithDate.GetDescriptions()
					if len(allDescs) > 0 {
						desc = allDescs[len(allDescs)-1]
					}
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

// PrintBlockedTasks prints the blocked tasks section to the writer.
func PrintBlockedTasks(out io.Writer, blocked []model.Task) {
	if len(blocked) == 0 {
		return
	}
	fmt.Fprintln(out, TextHeaderBlocked)
	for _, task := range blocked {
		fmt.Fprintf(out, "    • %s \n", task.JiraTicket)
		fmt.Fprintf(out, "        ◦ Blocker: %s\n", task.Blocker)
	}
}
