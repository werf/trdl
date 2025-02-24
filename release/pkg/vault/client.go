package vault

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"
)

type TaskLogger interface {
	Log(taskID, msg string)
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
}

// NewTrdlClient initializes the Vault client using DefaultConfig
func NewTrdlClient(vaultAddr string, vaultToken string, logger TaskLogger, enableRetry bool, maxAttempts int, delay time.Duration) (*TrdlClient, error) {
	config := api.DefaultConfig()
	config.Address = vaultAddr
	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	client.SetToken(vaultToken)

	return &TrdlClient{
		vaultClient: client,
		logger:      logger,
		enableRetry: enableRetry,
		maxAttempts: maxAttempts,
		delay:       delay,
	}, nil
}

func (c *TrdlClient) longRunningWrite(path string, data map[string]interface{}) (*api.Secret, error) {
	for {
		resp, err := c.vaultClient.Logical().Write(path, data)
		if err != nil {
			if err.Error() == "busy" {
				c.logger.Log("INFO", fmt.Sprintf("Vault is busy. Retrying request to %s in 5s...", path))
				time.Sleep(c.delay)
				continue
			}
			return nil, fmt.Errorf("failed to write to Vault at %s: %w", path, err)
		}
		return resp, nil
	}
}

func (c *TrdlClient) withRetryRequest(
	path string,
	data map[string]interface{},
	action func(taskID string, logger TaskLogger) error,
) error {
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		resp, err := c.longRunningWrite(path, data)
		if err != nil {
			c.logger.Log("ERROR", fmt.Sprintf("Attempt %d/%d failed: %v", attempt, c.maxAttempts, err))

			if !c.enableRetry || attempt == c.maxAttempts {
				return fmt.Errorf("request to %s failed after %d attempts: %w", path, c.maxAttempts, err)
			}

			time.Sleep(c.delay)
			continue
		}

		taskID, ok := resp.Data["task_uuid"].(string)
		if !ok {
			return fmt.Errorf("invalid response from Vault: missing task_uuid")
		}

		return action(taskID, c.logger)
	}

	return fmt.Errorf("unexpected error: reached unreachable code in withRetryRequest")
}

// Publish sends a publish request to Vault
func (c *TrdlClient) Publish(projectName string) error {
	return c.withRetryRequest(
		fmt.Sprintf("%s/publish", projectName),
		nil,
		func(taskID string, logger TaskLogger) error {
			return c.watchTask(projectName, taskID, logger)
		},
	)
}

// Release sends a release request to Vault
func (c *TrdlClient) Release(projectName, gitTag string) error {
	return c.withRetryRequest(
		fmt.Sprintf("%s/release", projectName),
		map[string]interface{}{"git_tag": gitTag},
		func(taskID string, logger TaskLogger) error {
			return c.watchTask(projectName, taskID, logger)
		},
	)
}

// watchTask waits for the task to finish and handles status changes
func (c *TrdlClient) watchTask(projectName, taskID string, logger TaskLogger) error {
	logger.Log(taskID, "Started task")

	for {
		status, reason, err := c.getTaskStatus(projectName, taskID)
		if err != nil {
			return fmt.Errorf("failed to fetch task status for %s: %w", taskID, err)
		}

		switch status {
		case "FAILED":
			logger.Log(taskID, fmt.Sprintf("Task failed: %s", reason))
			return fmt.Errorf("task %s failed: %s", taskID, reason)
		case "SUCCEEDED":
			return nil
		default:
			logger.Log(taskID, fmt.Sprintf("Task %s still running", taskID))
			time.Sleep(2 * time.Second)
		}
	}
}

// getTaskStatus retrieves the status of the task
func (c *TrdlClient) getTaskStatus(projectName, taskID string) (string, string, error) {
	resp, err := c.vaultClient.Logical().Read(fmt.Sprintf("%s/task/%s", projectName, taskID))
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch task status: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return "", "", nil
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal resp.Data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal task status: %w", err)
	}

	return task.Status, task.Reason, nil
}
