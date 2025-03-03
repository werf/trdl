package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
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

var (
	ErrTaskStatusUnavailable = errors.New("task status not available")
	ErrTaskFailed            = errors.New("task failed")
)

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

func (c *TrdlClient) longRunningWrite(path string, data map[string]interface{}) (*api.Secret, error) {
	for {
		resp, err := c.vaultClient.Logical().Write(path, data)
		if err != nil {
			if err.Error() == "busy" {
				c.logger.Warn("", fmt.Sprintf("Vault is busy. Retrying request to %s in 5s...", path))
				time.Sleep(c.delay)
				continue
			}
			c.logger.Error("", fmt.Sprintf("failed to write to Vault at %s: %v", path, err))
			return nil, err
		}
		return resp, nil
	}
}

func (c *TrdlClient) withBackoffRequest(
	path string,
	data map[string]interface{},
	action func(taskID string, logger TaskLogger) error,
) error {
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(c.delay), uint64(c.maxAttempts))

	operation := func() error {
		resp, err := c.longRunningWrite(path, data)
		if err != nil {
			c.logger.Error("", fmt.Sprintf("%v", err))
			return err
		}

		taskID, ok := resp.Data["task_uuid"].(string)
		if !ok {
			c.logger.Error("", "invalid response from Vault: missing task_uuid")
			return err
		}

		return action(taskID, c.logger)
	}

	err := backoff.RetryNotify(
		operation,
		bo,
		func(err error, duration time.Duration) {
			c.logger.Info("", fmt.Sprintf("Retrying %s after %v...", path, c.delay))
		},
	)

	if err != nil {
		c.logger.Error("", fmt.Sprintf("operation exceeded maximum duration: %v", err))
		return err
	}

	return err
}

func (c *TrdlClient) Publish(projectName string) {
	err := c.withBackoffRequest(
		fmt.Sprintf("%s/publish", projectName),
		nil,
		func(taskID string, logger TaskLogger) error {
			return c.watchTask(projectName, taskID, logger)
		},
	)
	if err != nil {
		c.logger.Error("", fmt.Sprintf("Failed to publish project %s: %v", projectName, err))
		os.Exit(1)
	}
}

func (c *TrdlClient) Release(projectName, gitTag string) {
	err := c.withBackoffRequest(
		fmt.Sprintf("%s/release", projectName),
		map[string]interface{}{"git_tag": gitTag},
		func(taskID string, logger TaskLogger) error {
			return c.watchTask(projectName, taskID, logger)
		},
	)
	if err != nil {
		c.logger.Error("", fmt.Sprintf("Failed to release project %s: %v", projectName, err))
		os.Exit(1)
	}
}

func (c *TrdlClient) watchTask(projectName, taskID string, logger TaskLogger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go c.WatchTaskLog(ctx, projectName, taskID, logger)
	for {
		status, reason := c.getTaskStatus(projectName, taskID)
		if status == "" {
			cancel()
			c.logger.Error("", "task status not available")
			return ErrTaskStatusUnavailable
		}
		switch status {
		case "FAILED":
			cancel()
			c.logger.Error("", fmt.Sprintf("Task %s failed: %s", taskID, reason))
			return ErrTaskFailed
		case "SUCCEEDED":
			cancel()
			return nil
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
