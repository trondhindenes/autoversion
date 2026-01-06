package ghactions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	HeadBranch   string `json:"headBranch"`
	HeadSHA      string `json:"headSha"`
	URL          string `json:"url"`
	DatabaseID   int64  `json:"databaseId"`
	Number       int    `json:"number"`
	WorkflowID   int64  `json:"workflowDatabaseId"`
	WorkflowName string `json:"workflowName"`
	Event        string `json:"event"`
	CreatedAt    string `json:"createdAt"`
	Title        string `json:"displayTitle"`
}

// VersionInfo represents the version info extracted from logs
type VersionInfo struct {
	Branch     string `json:"branch"`
	CommitSHA  string `json:"commitSha"`
	Version    string `json:"version"`
	Workflow   string `json:"workflow"`
	Job        string `json:"job"`
	Step       string `json:"step"`
	RunURL     string `json:"runUrl"`
	RunNumber  int    `json:"runNumber"`
	Conclusion string `json:"conclusion"`
}

// FinalVersionOutput represents the JSON structure in the log output
type FinalVersionOutput struct {
	Semver           string `json:"semver"`
	SemverWithPrefix string `json:"semverWithPrefix"`
	PEP440           string `json:"pep440"`
	PEP440WithPrefix string `json:"pep440WithPrefix"`
	Major            int    `json:"major"`
	Minor            int    `json:"minor"`
	Patch            int    `json:"patch"`
	IsRelease        bool   `json:"isRelease"`
}

// logVerbose prints a message to stderr if verbose mode is enabled
func logVerbose(verbose bool, format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// ListWorkflowRuns fetches recent workflow runs using gh CLI
func ListWorkflowRuns(workflow string, limit int, verbose bool) ([]WorkflowRun, error) {
	args := []string{"run", "list", "--json", "status,conclusion,headBranch,headSha,url,databaseId,number,workflowDatabaseId,workflowName,event,createdAt,displayTitle"}
	if workflow != "" {
		args = append(args, "-w", workflow)
	}
	if limit > 0 {
		args = append(args, "-L", fmt.Sprintf("%d", limit))
	}

	logVerbose(verbose, "Executing: gh %s", strings.Join(args, " "))

	cmd := exec.Command("gh", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute gh: %w", err)
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}

	logVerbose(verbose, "Found %d workflow runs", len(runs))

	return runs, nil
}

// GetJobLogs fetches logs for a specific run and filters by job name
func GetJobLogs(runID int64, jobName string, stepName string, verbose bool) (string, error) {
	logVerbose(verbose, "  Fetching logs for run %d...", runID)

	cmd := exec.Command("gh", "run", "view", fmt.Sprintf("%d", runID), "--log")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("gh command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to execute gh: %w", err)
	}

	logVerbose(verbose, "  Received %d bytes of logs", len(output))

	// Filter by job name if specified
	if jobName != "" {
		logVerbose(verbose, "  Filtering logs by job: %s", jobName)
	}
	return filterLogsByJob(string(output), jobName)
}

// filterLogsByJob filters log lines to only include those from a specific job
func filterLogsByJob(logs string, jobName string) (string, error) {
	if jobName == "" {
		return logs, nil
	}

	var filteredLogs strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(logs))
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		// Log lines look like: "JobName\tStepName\tTimestamp Message"
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) >= 1 {
			currentJob := parts[0]
			if strings.EqualFold(currentJob, jobName) {
				filteredLogs.WriteString(line)
				filteredLogs.WriteString("\n")
				found = true
			}
		}
	}

	if !found {
		return "", fmt.Errorf("job '%s' not found in logs", jobName)
	}

	return filteredLogs.String(), nil
}

// VersionExtractResult contains the extracted version and metadata about where it was found
type VersionExtractResult struct {
	Version *FinalVersionOutput
	Job     string
	Step    string
}

// ExtractFinalVersion extracts the "Final version:" JSON from logs
// It specifically looks for JSON-formatted output (containing {) to distinguish
// from test output that may also contain "Final version:" without JSON
// Returns the version along with the job and step names where it was found
func ExtractFinalVersion(logs string) (*VersionExtractResult, error) {
	scanner := bufio.NewScanner(strings.NewReader(logs))
	for scanner.Scan() {
		line := scanner.Text()
		// Look for "Final version:" followed by JSON (containing {)
		if idx := strings.Index(line, "Final version:"); idx != -1 {
			// Extract the part after "Final version:"
			afterMarker := line[idx+len("Final version:"):]

			// Only process if it contains JSON (starts with { after trimming)
			if braceIdx := strings.Index(afterMarker, "{"); braceIdx != -1 {
				jsonPart := afterMarker[braceIdx:]
				// Find the closing brace
				depth := 0
				endIdx := 0
				for i, ch := range jsonPart {
					if ch == '{' {
						depth++
					} else if ch == '}' {
						depth--
						if depth == 0 {
							endIdx = i + 1
							break
						}
					}
				}
				if endIdx > 0 {
					jsonPart = jsonPart[:endIdx]
				}

				var version FinalVersionOutput
				if err := json.Unmarshal([]byte(jsonPart), &version); err != nil {
					// This line had Final version: with { but wasn't valid JSON, continue looking
					continue
				}

				// Extract job and step from the log line
				// Log lines look like: "JobName\tStepName\tTimestamp Message"
				job := ""
				step := ""
				parts := strings.SplitN(line, "\t", 3)
				if len(parts) >= 1 {
					job = parts[0]
				}
				if len(parts) >= 2 {
					step = parts[1]
				}

				return &VersionExtractResult{
					Version: &version,
					Job:     job,
					Step:    step,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("'Final version:' with JSON output not found in logs")
}

// GetVersionsFromRuns fetches version info from multiple workflow runs
func GetVersionsFromRuns(workflow string, jobName string, stepName string, limit int, verbose bool) ([]VersionInfo, error) {
	logVerbose(verbose, "Listing workflow runs (workflow=%q, limit=%d)...", workflow, limit)

	runs, err := ListWorkflowRuns(workflow, limit, verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	var versions []VersionInfo
	for i, run := range runs {
		logVerbose(verbose, "Processing run #%d (%d/%d): branch=%s, sha=%s, conclusion=%s",
			run.Number, i+1, len(runs), run.HeadBranch, run.HeadSHA[:7], run.Conclusion)

		// Skip incomplete runs
		if run.Conclusion == "" || run.Conclusion == "cancelled" || run.Conclusion == "skipped" {
			logVerbose(verbose, "  Skipping run (conclusion=%s)", run.Conclusion)
			continue
		}

		logs, err := GetJobLogs(run.DatabaseID, jobName, stepName, verbose)
		if err != nil {
			// Log the error to stderr but continue with other runs
			fmt.Fprintf(os.Stderr, "Warning: failed to get logs for run %d: %v\n", run.DatabaseID, err)
			continue
		}

		result, err := ExtractFinalVersion(logs)
		if err != nil {
			// Log the error to stderr but continue with other runs
			fmt.Fprintf(os.Stderr, "Warning: failed to extract version from run %d: %v\n", run.DatabaseID, err)
			continue
		}

		logVerbose(verbose, "  Extracted version: %s (job=%s, step=%s)", result.Version.Semver, result.Job, result.Step)

		versions = append(versions, VersionInfo{
			Branch:     run.HeadBranch,
			CommitSHA:  run.HeadSHA[:7], // Short SHA
			Version:    result.Version.SemverWithPrefix,
			Workflow:   run.WorkflowName,
			Job:        result.Job,
			Step:       result.Step,
			RunURL:     run.URL,
			RunNumber:  run.Number,
			Conclusion: run.Conclusion,
		})
	}

	logVerbose(verbose, "Successfully extracted %d versions from %d runs", len(versions), len(runs))

	return versions, nil
}
