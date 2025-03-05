package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/vault/api"
	"golang.org/x/sync/errgroup"
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

type NewTrdlClientOpts struct {
	VaultAddress string
	VaultToken   string
	Retry        bool
	MaxAttempts  int
	Delay        time.Duration
	Logger       TaskLogger
}

// NewTrdlClient initializes the Vault client using DefaultConfig
func NewTrdlClient(opts NewTrdlClientOpts) (*TrdlClient, error) {
	config := api.DefaultConfig()
	config.Address = opts.VaultAddress
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(opts.VaultToken)

	return &TrdlClient{
		vaultClient: client,
		logger:      opts.Logger,
		enableRetry: opts.Retry,
		maxAttempts: opts.MaxAttempts,
		delay:       opts.Delay,
	}, nil
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

func (c *TrdlClient) withBackoffRequest(path string, data map[string]interface{}, action func(taskID string, logger TaskLogger) error) error {
	if !c.enableRetry {
		c.maxAttempts = 0
	}
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

func (c *TrdlClient) Publish(projectName string) error {
	err := c.withBackoffRequest(
		fmt.Sprintf("%s/publish", projectName),
		nil,
		func(taskID string, logger TaskLogger) error {
			return c.watchTask(projectName, taskID)
		},
	)
	if err != nil {
		c.logger.Error("", fmt.Sprintf("failed to publish project %s: %s", projectName, err.Error()))
		return fmt.Errorf("failed to publish project %s: %w", projectName, err)
	}
	return nil
}

func (c *TrdlClient) Release(projectName, gitTag string) error {
	err := c.withBackoffRequest(
		fmt.Sprintf("%s/release", projectName),
		map[string]interface{}{"git_tag": gitTag},
		func(taskID string, logger TaskLogger) error {
			return c.watchTask(projectName, taskID)
		},
	)
	if err != nil {
		c.logger.Error("", fmt.Sprintf("failed to release project %s: %s", projectName, err.Error()))
		return fmt.Errorf("failed to release project %s: %w", projectName, err)
	}
	return nil
}

func (c *TrdlClient) watchTask(projectName, taskID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return c.watchTaskLog(ctx, projectName, taskID)
	})

	g.Go(func() error {
		for {
			status, reason, err := c.getTaskStatus(projectName, taskID)
			if err != nil {
				c.logger.Error("", fmt.Sprintf("failed to get task status: %v", err))
				return ErrTaskStatusUnavailable
			}

			switch status {
			case "FAILED":
				c.logger.Error("", fmt.Sprintf("Task %s failed: %s", taskID, reason))
				cancel()
				return ErrTaskFailed
			case "SUCCEEDED":
				cancel()
				return nil
			default:
				time.Sleep(2 * time.Second)
			}
		}
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("watchTask failed: %w", err)
	}

	return nil
}

func (c *TrdlClient) getTaskStatus(projectName, taskID string) (string, string, error) {
	resp, err := c.vaultClient.Logical().Read(fmt.Sprintf("%s/task/%s", projectName, taskID))
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to fetch task status: %v", err))
		return "", "", fmt.Errorf("failed to fetch task status: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return "", "", fmt.Errorf("no response data")
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to marshal resp.Data: %v", err))
		return "", "", fmt.Errorf("failed to marshal resp.Data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to unmarshal task status: %v", err))
		return "", "", fmt.Errorf("failed to unmarshal task status: %w", err)
	}

	return task.Status, task.Reason, nil
}

func (c *TrdlClient) getTaskLogs(projectName, taskID string) (string, error) {
	resp, err := c.vaultClient.Logical().Read(fmt.Sprintf("%s/task/%s/log", projectName, taskID))
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to fetch task logs: %v", err))
		return "", nil
	}
	if resp == nil || resp.Data == nil {
		return "", nil
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to marshal resp.Data: %v", err))
		return "", fmt.Errorf("failed to marshal resp.Data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		c.logger.Error(taskID, fmt.Sprintf("failed to unmarshal task logs: %v", err))
		return "", fmt.Errorf("failed to unmarshal task logs: %w", err)
	}

	return task.Log, nil
}

func (c *TrdlClient) watchTaskLog(ctx context.Context, projectName, taskID string) error {
	var lastLines []string

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			logData, err := c.getTaskLogs(projectName, taskID)
			if err != nil {
				return fmt.Errorf("failed to get task logs: %w", err)
			}
			if logData == "" {
				time.Sleep(2 * time.Second)
				continue
			}

			lines := cleanAndSplitLog(logData)

			newLines := diffLines(lastLines, lines)
			for _, line := range newLines {
				c.logger.Info(taskID, line)
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
