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
	"github.com/werf/trdl/client/pkg/logger"
	"golang.org/x/sync/errgroup"
)

type TrdlClient struct {
	vaultClient *api.Client
	logger      *logger.Logger
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
	Address     string
	Token       string
	Retry       bool
	MaxAttempts int
	Delay       time.Duration
	Logger      *logger.Logger
}

// NewTrdlClient initializes the Vault client using DefaultConfig
func NewTrdlClient(opts NewTrdlClientOpts) (*TrdlClient, error) {
	config := api.DefaultConfig()
	config.Address = opts.Address
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(opts.Token)

	return &TrdlClient{
		vaultClient: client,
		logger:      opts.Logger,
		enableRetry: opts.Retry,
		maxAttempts: opts.MaxAttempts,
		delay:       opts.Delay,
	}, nil
}

func (c *TrdlClient) longRunningWrite(path string, data map[string]interface{}) (*api.Secret, error) {
	log := c.logger.With("path", path)
	for {
		resp, err := c.vaultClient.Logical().Write(path, data)
		if err != nil {
			if err.Error() == "busy" {
				log.Warn(fmt.Sprintf("Vault is busy. Retrying request to %s in 5s...", path))
				time.Sleep(c.delay)
				continue
			}
			log.Error(fmt.Sprintf("failed to write to Vault at %s", path), "error", err)
			return nil, err
		}
		return resp, nil
	}
}

func (c *TrdlClient) withBackoffRequest(path string, data map[string]interface{}, action func(taskID string, logger *logger.Logger) error) error {
	log := c.logger.With("path", path)
	if !c.enableRetry {
		c.maxAttempts = 0
	}
	bo := backoff.WithMaxRetries(backoff.NewConstantBackOff(c.delay), uint64(c.maxAttempts))

	operation := func() error {
		resp, err := c.longRunningWrite(path, data)
		if err != nil {
			log.Error(fmt.Sprintf("%v", err))
			return err
		}

		taskID, ok := resp.Data["task_uuid"].(string)
		if !ok {
			log.Error("invalid response from Vault: missing task_uuid")
			return err
		}

		return action(taskID, c.logger)
	}

	err := backoff.RetryNotify(
		operation,
		bo,
		func(err error, duration time.Duration) {
			log.Info(fmt.Sprintf("Retrying after %v...", c.delay))
		},
	)
	if err != nil {
		log.Error(fmt.Sprintf("operation exceeded maximum duration: %v", err))
		return err
	}

	return err
}

func (c *TrdlClient) Publish(projectName string) error {
	err := c.withBackoffRequest(
		fmt.Sprintf("%s/publish", projectName),
		nil,
		func(taskID string, logger *logger.Logger) error {
			return c.watchTask(projectName, taskID)
		},
	)
	if err != nil {
		c.logger.Error(fmt.Sprintf("failed to publish project: %s", err.Error()), "project", projectName)
		return fmt.Errorf("failed to publish project %s: %w", projectName, err)
	}
	return nil
}

func (c *TrdlClient) Release(projectName, gitTag string) error {
	err := c.withBackoffRequest(
		fmt.Sprintf("%s/release", projectName),
		map[string]interface{}{"git_tag": gitTag},
		func(taskID string, logger *logger.Logger) error {
			return c.watchTask(projectName, taskID)
		},
	)
	if err != nil {
		c.logger.Error(fmt.Sprintf("failed to release project: %s", err.Error()), "project", projectName)
		return fmt.Errorf("failed to release project %s: %w", projectName, err)
	}
	return nil
}

func (c *TrdlClient) watchTask(projectName, taskID string) error {
	log := c.logger.With("taskId", taskID, "project", projectName)
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
				log.Error(fmt.Sprintf("failed to get task status: %v", err))
				return ErrTaskStatusUnavailable
			}

			switch status {
			case "FAILED":
				log.Error(fmt.Sprintf("Task failed: %s", reason))
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
	log := c.logger.With("taskID", taskID, "project", projectName)
	resp, err := c.vaultClient.Logical().Read(fmt.Sprintf("%s/task/%s", projectName, taskID))
	if err != nil {
		log.Error(fmt.Sprintf("failed to fetch task status: %v", err))
		return "", "", fmt.Errorf("failed to fetch task status: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return "", "", fmt.Errorf("no response data")
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		log.Error(fmt.Sprintf("failed to marshal resp.Data: %v", err))
		return "", "", fmt.Errorf("failed to marshal resp.Data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		log.Error(fmt.Sprintf("failed to unmarshal task status: %v", err))
		return "", "", fmt.Errorf("failed to unmarshal task status: %w", err)
	}

	return task.Status, task.Reason, nil
}

func (c *TrdlClient) getTaskLogs(projectName, taskID string, offset int) (string, error) {
	log := c.logger.With("taskID", taskID, "project", projectName)

	data := map[string][]string{
		"offset": {fmt.Sprintf("%d", offset)},
		"limit":  {"1000000000"},
	}
	resp, err := c.vaultClient.Logical().ReadWithData(
		fmt.Sprintf("%s/task/%s/log", projectName, taskID),
		data,
	)

	if err != nil {
		log.Error(fmt.Sprintf("failed to fetch task logs: %v", err))
		return "", nil
	}
	if resp == nil || resp.Data == nil {
		return "", nil
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		log.Error(fmt.Sprintf("failed to marshal resp.Data: %v", err))
		return "", fmt.Errorf("failed to marshal resp.Data: %w", err)
	}

	var task Task
	if err := json.Unmarshal(dataBytes, &task); err != nil {
		log.Error(fmt.Sprintf("failed to unmarshal task logs: %v", err))
		return "", fmt.Errorf("failed to unmarshal task logs: %w", err)
	}

	return task.Log, nil
}

func (c *TrdlClient) watchTaskLog(ctx context.Context, projectName, taskID string) error {
	log := c.logger.With("taskId", taskID, "project", projectName)
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			logData, err := c.getTaskLogs(projectName, taskID, offset)
			if err != nil {
				return fmt.Errorf("failed to get task logs: %w", err)
			}
			if logData == "" {
				time.Sleep(2 * time.Second)
				continue
			}

			lines := cleanAndSplitLog(logData)
			for _, line := range lines {
				log.Info(line)
			}
			offset += len(logData)
			time.Sleep(2 * time.Second)
		}
	}
}

func cleanAndSplitLog(log string) []string {
	lines := strings.Split(log, "\n")

	re := regexp.MustCompile(`^\d+h?\d+m\d+(\.\d+)?\s+`)

	var cleanedLines []string
	for _, line := range lines {
		line = re.ReplaceAllString(line, "")
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return cleanedLines
}
