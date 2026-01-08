// Package model defines the core data structures for TaskLedger.
package model

// Task status constants.
const (
	StatusCompleted  = "completed"
	StatusInProgress = "in progress"
	StatusNotStarted = "not started"
)

// WorkLog represents a single time entry (start and end).
type WorkLog struct {
	StartTime string `yaml:"start_time"`
	EndTime   string `yaml:"end_time"`
}

// Task represents a single work item.
type Task struct {
	Status            string   `yaml:"status"`
	Description       string   `yaml:"description"`
	Descriptions      []string `yaml:"descriptions"`
	JiraTicket        string   `yaml:"jira_ticket"`
	QCGoal            string   `yaml:"qc_goal"`
	UpnextDescription string   `yaml:"upnext_description"`
	GithubPR          string   `yaml:"github_pr"`
	Blocker           string   `yaml:"blocker"`
}

// GetDescriptions returns all descriptions for a task, combining both
// the singular Description field and the Descriptions array.
// This provides backward compatibility while supporting multiple descriptions.
func (t *Task) GetDescriptions() []string {
	var descs []string
	if t.Description != "" {
		descs = append(descs, t.Description)
	}
	descs = append(descs, t.Descriptions...)
	return descs
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

// CategorizedTasks holds tasks organized by their report section.
type CategorizedTasks struct {
	Completed map[string][]TaskWithDate // Jira ticket -> list of completed/in-progress tasks
	NextUp    map[string][]TaskWithDate // Jira ticket -> list of tasks with next up descriptions
	Blocked   []Task                    // Tasks with blockers
}
