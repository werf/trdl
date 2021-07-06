package trdl

import (
	"context"
	"fmt"

	git "github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager"
)

const (
	DefaultGitTrdlChannelsBranch = "trdl"
	LastPublishedGitCommitKey    = "last_published_git_commit"
)

func pathPublish(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: `publish$`,
		Fields: map[string]*framework.FieldSchema{
			fieldNameGitCredentialUsername: {
				Type:        framework.TypeString,
				Description: "Git username",
			},
			fieldNameGitCredentialPassword: {
				Type:        framework.TypeString,
				Description: "Git password",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathPublish,
				Summary:  pathPublishHelpSyn,
			},
		},

		HelpSynopsis:    pathPublishHelpSyn,
		HelpDescription: pathPublishHelpDesc,
	}
}

func GetGitTrdlChannelsBranch(cfg *Configuration) string {
	if cfg.GitTrdlChannelsBranch != "" {
		return cfg.GitTrdlChannelsBranch
	}
	return DefaultGitTrdlChannelsBranch
}

func (b *backend) pathPublish(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	c, resp, err := GetAndValidateConfiguration(ctx, req.Storage)
	if resp != nil || err != nil {
		return resp, err
	}

	var gitUsername string
	val, ok := fields.GetOk(fieldNameGitCredentialUsername)
	if ok {
		gitUsername = val.(string)
	} else {
		gitUsername = c.GitCredential.Username
	}

	var gitPassword string
	val, ok = fields.GetOk(fieldNameGitCredentialPassword)
	if ok {
		gitPassword = val.(string)
	} else {
		gitPassword = c.GitCredential.Password
	}

	gitBranch := GetGitTrdlChannelsBranch(c)

	var lastPublishedGitCommit string
	if lastCommitEntry, err := req.Storage.Get(ctx, LastPublishedGitCommitKey); err != nil {
		return nil, fmt.Errorf("error getting last published git commit by key %q from storage: %s", LastPublishedGitCommitKey, err)
	} else if lastCommitEntry != nil {
		lastPublishedGitCommit = string(lastCommitEntry.Value)
	}

	publisherRepository, err := GetPublisherRepository(ctx, c, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting publisher repository: %s", err)
	}

	taskUUID, err := b.TaskQueueManager.RunTask(context.Background(), req.Storage, func(ctx context.Context, storage logical.Storage) error {
		logboek.Context(ctx).Default().LogF("Started task\n")
		hclog.L().Debug(fmt.Sprintf("Started task"))

		gitRepo, err := cloneGitRepositoryBranch(c.GitRepoUrl, gitBranch, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Cloned git repo\n")
		hclog.L().Debug(fmt.Sprintf("Cloned git repo"))

		headRef, err := gitRepo.Head()
		if err != nil {
			return fmt.Errorf("error getting git repo branch %q head reference: %s", gitBranch, err)
		}

		if lastPublishedGitCommit != "" {
			logboek.Context(ctx).Default().LogF("Got previously published commit record %q\n", lastPublishedGitCommit)
			hclog.L().Debug(fmt.Sprintf("Got previously published commit record %q", lastPublishedGitCommit))

			isAncestor, err := trdlGit.IsAncestor(gitRepo, lastPublishedGitCommit, headRef.Hash().String())
			if err != nil {
				return err
			}

			if !isAncestor {
				return fmt.Errorf("cannot publish git commit %q which is not desdendant of previously published git commit %q", headRef.Hash().String(), lastPublishedGitCommit)
			}
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headRef.Hash().String(), c.TrustedPGPPublicKeys, c.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return fmt.Errorf("signature verification failed: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Verified commit signatures\n")
		hclog.L().Debug(fmt.Sprintf("Verified commit signatures"))

		cfg, err := GetTrdlChannelsConfig(gitRepo)
		if err != nil {
			return fmt.Errorf("error getting trdl channels config: %s", err)
		}

		cfgDump, _ := yaml.Marshal(cfg)
		logboek.Context(ctx).Default().LogF("Got trdl channels config:\n%s\n---\n", cfgDump)
		hclog.L().Debug(fmt.Sprintf("Got trdl channels config:\n%s\n---", cfgDump))

		if err := publisher.PublishChannelsConfig(ctx, publisherRepository, cfg); err != nil {
			return fmt.Errorf("error publishing trdl channels into the repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Published trdl channels config into the repository\n")
		hclog.L().Debug(fmt.Sprintf("Published trdl channels config into the repository"))

		if err := publisherRepository.Commit(ctx); err != nil {
			return fmt.Errorf("unable to commit new tuf repository state: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Tuf repo commit done\n")
		hclog.L().Debug(fmt.Sprintf("Tuf repo commit done"))

		if err := storage.Put(ctx, &logical.StorageEntry{Key: LastPublishedGitCommitKey, Value: []byte(headRef.Hash().String())}); err != nil {
			return fmt.Errorf("error putting published commit record by key %q: %s", LastPublishedGitCommitKey, err)
		}

		logboek.Context(ctx).Default().LogF("Put published commit record %q\n", headRef.Hash().String())
		hclog.L().Debug(fmt.Sprintf("Put published commit record %q", headRef.Hash().String()))

		return nil
	})
	if err != nil {
		if err == queue_manager.QueueBusyError {
			return logical.ErrorResponse(err.Error()), nil
		}

		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"task_uuid": taskUUID,
		},
	}, nil
}

func GetTrdlChannelsConfig(gitRepo *git.Repository) (*config.TrdlChannels, error) {
	data, err := trdlGit.ReadWorktreeFile(gitRepo, config.TrdlChannelsFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read worktree file %s: %s", config.TrdlChannelsFileName, err)
	}

	cfg, err := config.ParseTrdlChannels(data)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s configuration file: %s", config.TrdlChannelsFileName, err)
	}

	return cfg, nil
}

const (
	pathPublishHelpSyn = `
	Publishes release channels mapping of the project.
	`

	pathPublishHelpDesc = `
	Publishes release channels mapping of the project using trdl-channels.yaml configuration file.
	`
)
