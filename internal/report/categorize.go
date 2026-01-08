// Package report provides report generation and task categorization.
package report

import (
	"strings"

	"github.com/bryan-cox/taskledger/internal/jira"
	"github.com/bryan-cox/taskledger/internal/model"
)

// IsNonFeatureWork returns true if the task should be grouped under "Non-feature work".
// A task is non-feature work if:
// - jira_ticket is empty, OR
// - jira_ticket contains "NO-JIRA" AND github_pr is empty, OR
// - jira_ticket does NOT contain a valid JIRA ticket pattern (PROJ-123) AND does NOT contain "NO-JIRA"
//
// Note: NO-JIRA with a PR is considered feature work (shown as its own entry, not under non-feature)
func IsNonFeatureWork(ticket string, githubPR string) bool {
	if ticket == "" {
		return true
	}
	// NO-JIRA items: non-feature only if no PR, otherwise feature work
	if strings.Contains(strings.ToUpper(ticket), "NO-JIRA") {
		return githubPR == ""
	}
	// Reuse existing jira.ExtractTicketID to check for valid JIRA pattern
	if jira.ExtractTicketID(ticket) == "" {
		return true
	}
	return false
}

// emptyTicketKey is a placeholder key for tasks without JIRA tickets.
const emptyTicketKey = "__empty__"

// CategorizeTasks groups tasks from the work data into completed, next up, and blocked categories.
func CategorizeTasks(workData model.WorkData, dates []string) model.CategorizedTasks {
	completedTasks := make(map[string][]model.TaskWithDate)
	allNextUpTasks := make(map[string][]model.TaskWithDate)
	mostRecentTasks := make(map[string]model.TaskWithDate)

	for _, date := range dates {
		dailyLog, exists := workData[date]
		if !exists {
			continue
		}
		for _, task := range dailyLog.Tasks {
			taskWithDate := model.TaskWithDate{Task: task, Date: date}
			jiraTicket := task.JiraTicket

			// Track completed tasks - include both completed and in-progress tasks with descriptions
			if strings.EqualFold(task.Status, model.StatusCompleted) ||
				(strings.EqualFold(task.Status, model.StatusInProgress) && len(task.GetDescriptions()) > 0) {
				completedTasks[jiraTicket] = append(completedTasks[jiraTicket], taskWithDate)
			}

			// Collect all tasks with upnext descriptions
			if task.UpnextDescription != "" {
				allNextUpTasks[jiraTicket] = append(allNextUpTasks[jiraTicket], taskWithDate)
			}

			// Track most recent task per Jira ticket
			taskKey := jiraTicket
			if taskKey == "" {
				taskKey = emptyTicketKey
			}
			if existing, exists := mostRecentTasks[taskKey]; !exists || date > existing.Date {
				mostRecentTasks[taskKey] = taskWithDate
			}
		}
	}

	// Filter next up tasks: only include tickets where the most recent task is still in progress or not started
	nextUpTasks := make(map[string][]model.TaskWithDate)
	for jiraTicket, taskList := range allNextUpTasks {
		taskKey := jiraTicket
		if taskKey == "" {
			taskKey = emptyTicketKey
		}
		if mostRecent, exists := mostRecentTasks[taskKey]; exists {
			if strings.EqualFold(mostRecent.Status, model.StatusInProgress) || strings.EqualFold(mostRecent.Status, model.StatusNotStarted) {
				nextUpTasks[jiraTicket] = taskList
			}
		}
	}

	// Filter blocked tasks: only include tickets where the most recent task has a blocker
	var blockedTasks []model.Task
	for _, taskWithDate := range mostRecentTasks {
		if taskWithDate.Blocker != "" {
			blockedTasks = append(blockedTasks, taskWithDate.Task)
		}
	}

	return model.CategorizedTasks{
		Completed: completedTasks,
		NextUp:    nextUpTasks,
		Blocked:   blockedTasks,
	}
}
