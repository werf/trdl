package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

type TaskLogger interface {
	Debug(taskID, msg string)
	Info(taskID, msg string)
	Warn(taskID, msg string)
	Error(taskID, msg string)
}

type TrdlClient struct {
	vaultClient *api.Client
	logger      TaskLogger
	enableRetry bool
	maxAttempts int
	delay       time.Duration
}

type Task struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
	Log    string `json:"result"`
}

// NewTrdlClient initializes the Vault client using DefaultConfig
func NewTrdlClient(vaultAddr string, vaultToken string, logger TaskLogger, enableRetry bool, maxAttempts int, delay time.Duration) *TrdlClient {
	config := api.DefaultConfig()
	config.Address = vaultAddr
	client, err := api.NewClient(config)
	if err != nil {
		logger.Error("", fmt.Sprintf("failed to create Vault client: %v", err))
		os.Exit(1)
	}

	client.SetToken(vaultToken)

	return &TrdlClient{
		vaultClient: client,
		logger:      logger,
		enableRetry: enableRetry,
		maxAttempts: maxAttempts,
		delay:       delay,
	}
}

func (c *TrdlClient) longRunningWrite(path string, data map[string]interface{}) *api.Secret {
	for {
		resp, err := c.vaultClient.Logical().Write(path, data)
		if err != nil {
			if err.Error() == "busy" {
				c.logger.Warn("", fmt.Sprintf("Vault is busy. Retrying request to %s in 5s...", path))
				time.Sleep(c.delay)
				continue
			}
			c.logger.Error("", fmt.Sprintf("failed to write to Vault at %s: %v", path, err))
			os.Exit(1)
		}
		return resp
	}
}

func (c *TrdlClient) withRetryRequest(
	path string,
	data map[string]interface{},
	action func(taskID string, logger TaskLogger),
) {
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		resp := c.longRunningWrite(path, data)
		if resp == nil {
			c.logger.Error("", fmt.Sprintf("Attempt %d/%d failed: Vault write failed", attempt, c.maxAttempts))
			if !c.enableRetry || attempt == c.maxAttempts {
				return
			}
			time.Sleep(c.delay)
			continue
		}

		taskID, ok := resp.Data["task_uuid"].(string)
		if !ok {
			c.logger.Error("", fmt.Sprintf("invalid response from Vault: missing task_uuid"))
			return
		}

		action(taskID, c.logger)
		return
	}

	c.logger.Error("", fmt.Sprintf("unexpected error: reached unreachable code in withRetryRequest"))
	os.Exit(1)
}

func (c *TrdlClient) Publish(projectName string) {
	c.withRetryRequest(
		fmt.Sprintf("%s/publish", projectName),
		nil,
		func(taskID string, logger TaskLogger) {
			c.watchTask(projectName, taskID, logger)
		},
	)
}

func (c *TrdlClient) Release(projectName, gitTag string) {
	c.withRetryRequest(
		fmt.Sprintf("%s/release", projectName),
		map[string]interface{}{"git_tag": gitTag},
		func(taskID string, logger TaskLogger) {
			c.watchTask(projectName, taskID, logger)
		},
	)
}

func (c *TrdlClient) watchTask(projectName, taskID string, logger TaskLogger) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.WatchTaskLog(ctx, projectName, taskID, logger)

	for {
		status, reason := c.getTaskStatus(projectName, taskID)
		if status == "" {
			cancel()
			return
		}

		switch status {
		case "FAILED":
			cancel()
			c.logger.Error(taskID, fmt.Sprintf("task %s failed: %s", taskID, reason))
			return
		case "SUCCEEDED":
			cancel()
			return
		default:
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *TrdlClient) getTaskStatus(projectName, taskID string) (string, string) {
	resp, err := c.vaultClient.Logical().Read(fmt.Sprintf("%s/task/%s", projectName, taskID))
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to fetch task status: %v", err))
		return "", ""
	}
	if resp == nil || resp.Data == nil {
		return "", ""
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to marshal resp.Data: %v", err))
		return "", ""
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to unmarshal task status: %v", err))
		return "", ""
	}

	return task.Status, task.Reason
}

func (c *TrdlClient) getTaskLogs(projectName, taskID string) string {
	resp, err := c.vaultClient.Logical().Read(fmt.Sprintf("%s/task/%s/log", projectName, taskID))
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to fetch task logs: %v", err))
		return ""
	}
	if resp == nil || resp.Data == nil {
		return ""
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to marshal resp.Data: %v", err))
		os.Exit(1)
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to unmarshal task logs: %v", err))
		os.Exit(1)
	}

	return task.Log
}

func (c *TrdlClient) WatchTaskLog(ctx context.Context, projectName, taskID string, logger TaskLogger) {
	var lastLines []string

	for {
		select {
		case <-ctx.Done():
			return
		default:
			logData := c.getTaskLogs(projectName, taskID)
			if logData == "" {
				time.Sleep(2 * time.Second)
				continue
			}

			lines := cleanAndSplitLog(logData)

			newLines := diffLines(lastLines, lines)
			for _, line := range newLines {
				logger.Info(taskID, line)
			}

			lastLines = lines
			time.Sleep(2 * time.Second)
		}
	}
}

func cleanAndSplitLog(log string) []string {
	lines := strings.Split(log, "\n")

	re := regexp.MustCompile(`^\d+h?\d+m\d+(\.\d+)?\s+`)

	var cleanedLines []string
	for _, line := range lines {
		cleanedLine := re.ReplaceAllString(line, "")
		if cleanedLine != "" {
			cleanedLines = append(cleanedLines, cleanedLine)
		}
	}

	return cleanedLines
}

func diffLines(oldLines, newLines []string) []string {
	if len(oldLines) == 0 {
		return newLines
	}

	lastIndex := len(oldLines)
	if lastIndex >= len(newLines) {
		return nil
	}

	return newLines[lastIndex:]
}
