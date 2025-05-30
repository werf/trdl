package client

import (
	"fmt"
	"time"

	"github.com/werf/trdl/client/pkg/logger"
	"github.com/werf/trdl/release/pkg/vault"
)

type Interface interface {
	Publish(projectName string) error
	Release(projectName, gitTag string) error
}

type Client struct {
	client Interface
}

func newClient(client Interface) *Client {
	return &Client{client: client}
}

func (c *Client) Publish(projectName string) error {
	return c.client.Publish(projectName)
}

func (c *Client) Release(projectName, gitTag string) error {
	return c.client.Release(projectName, gitTag)
}

type NewTrdlVaultClientOpts struct {
	Address     string
	Token       string
	Retry       bool
	MaxAttempts int
	Delay       time.Duration
	Logger      *logger.Logger
}

func NewTrdlVaultClient(opts NewTrdlVaultClientOpts) (*Client, error) {
	log := opts.Logger
	trdlClient, err := vault.NewTrdlClient(vault.NewTrdlClientOpts{
		Address:     opts.Address,
		Token:       opts.Token,
		Retry:       opts.Retry,
		MaxAttempts: opts.MaxAttempts,
		Delay:       opts.Delay,
		Logger:      log,
	})
	if err != nil {
		log.Error(fmt.Sprintf("Unable to create Vault client: %s", err.Error()))
		return nil, fmt.Errorf("new Vault client error: %w", err)
	}
	return newClient(trdlClient), nil
}
