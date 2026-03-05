// Package report provides report generation and task categorization.
package report

import (
	"fmt"
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
	// Synthetic keys from categorization are always non-feature work
	if IsSyntheticKey(ticket) {
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

// IsSyntheticKey returns true if the key is a generated grouping key (not a real ticket name).
// Synthetic keys are used for tasks without JIRA tickets, grouped by PR URL or unique counter.
func IsSyntheticKey(key string) bool {
	return strings.HasPrefix(key, "__noticket_") || strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://")
}

// CategorizeTasks groups tasks from the work data into completed, next up, and blocked categories.
func CategorizeTasks(workData model.WorkData, dates []string) model.CategorizedTasks {
	completedTasks := make(map[string][]model.TaskWithDate)
	allNextUpTasks := make(map[string][]model.TaskWithDate)
	mostRecentTasks := make(map[string]model.TaskWithDate)

	emptyCounter := 0
	for _, date := range dates {
		dailyLog, exists := workData[date]
		if !exists {
			continue
		}
		for _, task := range dailyLog.Tasks {
			taskWithDate := model.TaskWithDate{Task: task, Date: date}
			jiraTicket := task.JiraTicket

			// Compute grouping key: for tasks without a JIRA ticket, group by PR URL
			// or assign a unique key so they don't all merge under one entry
			groupKey := jiraTicket
			if jiraTicket == "" {
				if task.GithubPR != "" {
					groupKey = task.GithubPR
				} else {
					groupKey = fmt.Sprintf("__noticket_%d__", emptyCounter)
					emptyCounter++
				}
			}

			// Track completed tasks - include both completed and in-progress tasks with descriptions
			if strings.EqualFold(task.Status, model.StatusCompleted) ||
				(strings.EqualFold(task.Status, model.StatusInProgress) && len(task.GetDescriptions()) > 0) {
				completedTasks[groupKey] = append(completedTasks[groupKey], taskWithDate)
			}

			// Collect all tasks with upnext descriptions
			if task.UpnextDescription != "" {
				allNextUpTasks[groupKey] = append(allNextUpTasks[groupKey], taskWithDate)
			}

			// Track most recent task per group
			if existing, exists := mostRecentTasks[groupKey]; !exists || date > existing.Date {
				mostRecentTasks[groupKey] = taskWithDate
			}
		}
	}

	// Filter next up tasks: only include tickets where the most recent task is still in progress or not started
	nextUpTasks := make(map[string][]model.TaskWithDate)
	for groupKey, taskList := range allNextUpTasks {
		if mostRecent, exists := mostRecentTasks[groupKey]; exists {
			if strings.EqualFold(mostRecent.Status, model.StatusInProgress) || strings.EqualFold(mostRecent.Status, model.StatusNotStarted) {
				nextUpTasks[groupKey] = taskList
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
