// Package jira provides JIRA integration for fetching ticket information.
package jira

import (
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/bryan-cox/taskledger/internal/model"
)

// BaseURL is the JIRA instance base URL.
const BaseURL = "https://issues.redhat.com"

// TicketInfo holds information about a JIRA ticket.
type TicketInfo struct {
	Key     string
	Summary string
	URL     string
}

// apiResponse represents the response from JIRA API.
type apiResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

// Regex patterns for extracting JIRA ticket IDs.
var (
	ticketRegex = regexp.MustCompile(`\b([A-Z]+-\d+)\b`)
	urlRegex    = regexp.MustCompile(`https://issues\.redhat\.com/browse/([A-Z]+-\d+)`)
)

// ExtractTicketID extracts a JIRA ticket ID from a URL or text.
func ExtractTicketID(input string) string {
	// First try to extract from URL
	if matches := urlRegex.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1]
	}

	// Then try to extract from plain text
	if matches := ticketRegex.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// FetchTicketSummary fetches the summary of a JIRA ticket using the API.
func FetchTicketSummary(ticketID string) (TicketInfo, error) {
	ticket := TicketInfo{
		Key: ticketID,
		URL: fmt.Sprintf("%s/browse/%s", BaseURL, ticketID),
	}

	// Check if JIRA Personal Access Token is available
	jiraPAT := os.Getenv("JIRA_PAT")
	if jiraPAT == "" {
		// Return ticket info without summary if no PAT is available
		return ticket, nil
	}

	// Make API request to fetch ticket summary
	apiURL := fmt.Sprintf("%s/rest/api/2/issue/%s?fields=summary", BaseURL, ticketID)

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

	var jiraResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&jiraResp); err != nil {
		return ticket, fmt.Errorf("failed to decode response: %w", err)
	}

	ticket.Summary = jiraResp.Fields.Summary
	return ticket, nil
}

// ProcessTickets processes a map of JIRA tickets and fetches their summaries.
func ProcessTickets(tickets map[string][]model.TaskWithDate) map[string]TicketInfo {
	jiraInfo := make(map[string]TicketInfo)

	for ticketReference := range tickets {
		if ticketReference == "" {
			continue
		}

		ticketID := ExtractTicketID(ticketReference)
		if ticketID == "" {
			continue
		}

		if _, exists := jiraInfo[ticketID]; !exists {
			// Fetch ticket info (will include summary only if JIRA_PAT is available)
			if info, err := FetchTicketSummary(ticketID); err == nil {
				jiraInfo[ticketID] = info
			} else {
				// If fetch fails, still create basic info
				jiraInfo[ticketID] = TicketInfo{
					Key: ticketID,
					URL: fmt.Sprintf("%s/browse/%s", BaseURL, ticketID),
				}
				slog.Warn("failed to fetch JIRA ticket summary", "ticket", ticketID, "error", err)
			}
		}
	}

	return jiraInfo
}

// LoadSummariesFromFile loads JIRA ticket summaries from a JSON file.
// The file should contain a map of ticket IDs to TicketInfo objects.
func LoadSummariesFromFile(filePath string) (map[string]TicketInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JIRA summaries file: %w", err)
	}

	var summaries map[string]TicketInfo
	if err := json.Unmarshal(data, &summaries); err != nil {
		return nil, fmt.Errorf("failed to parse JIRA summaries JSON: %w", err)
	}

	// Ensure URLs are set for all tickets
	for key, info := range summaries {
		if info.URL == "" {
			info.URL = fmt.Sprintf("%s/browse/%s", BaseURL, info.Key)
		}
		if info.Key == "" {
			info.Key = key
		}
		summaries[key] = info
	}

	return summaries, nil
}

// FormatTicketHTML formats a JIRA ticket reference as HTML with optional summary.
func FormatTicketHTML(ticketReference string, jiraInfo map[string]TicketInfo) string {
	ticketID := ExtractTicketID(ticketReference)
	if ticketID == "" {
		// No JIRA ticket found, return escaped original text
		return html.EscapeString(ticketReference)
	}

	info, exists := jiraInfo[ticketID]
	if !exists {
		// Fallback: create basic link
		url := fmt.Sprintf("%s/browse/%s", BaseURL, ticketID)
		return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, url, html.EscapeString(ticketID))
	}

	// Create link with summary if available
	linkText := info.Key
	if info.Summary != "" {
		linkText = fmt.Sprintf("%s: %s", info.Key, info.Summary)
	}

	return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, info.URL, html.EscapeString(linkText))
}
