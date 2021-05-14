package trdl

import (
	"context"
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v2"

	log "github.com/hashicorp/go-hclog"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager"
)

const (
	PublishedCommitKey = "published_commit"
)

func pathPublish(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: `publish$`,
		Fields: map[string]*framework.FieldSchema{
			"reset_commit": {
				Type:        framework.TypeBool,
				Description: "Reset previously published commit even if current commit is not descendant of previous (optional)",
			},
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

func (b *backend) pathPublish(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	resetCommit := fields.Get("reset_commit").(bool)

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

	gitBranch := "trdl" // TODO: get branch from vault storage

	publisherRepository, err := GetPublisherRepository(req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting publisher repository: %s", err)
	}

	taskUUID, err := b.TaskQueueManager.RunTask(context.Background(), req.Storage, func(ctx context.Context, storage logical.Storage) error {
		logboek.Context(ctx).Default().LogF("Started task\n")
		log.L().Debug("Started task\n") // Remove this debug when tasks log debugged

		gitRepo, err := cloneGitRepositoryBranch(c.GitRepoUrl, gitBranch, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Cloned git repo\n")
		log.L().Debug("Cloned git repo\n") // Remove this debug when tasks log debugged

		headRef, err := gitRepo.Head()
		if err != nil {
			return fmt.Errorf("error getting git repo branch %q head reference: %s", gitBranch, err)
		}

		headCommitObj, err := gitRepo.CommitObject(headRef.Hash())
		if err != nil {
			return fmt.Errorf("error getting git repo commit object by hash %q: %s", headRef.Hash().String(), err)
		}

		if !resetCommit {
			prevCommitEntry, err := storage.Get(ctx, PublishedCommitKey)
			if err != nil {
				return fmt.Errorf("error getting published commit by key %q from storage: %s", PublishedCommitKey, err)
			}

			if prevCommitEntry != nil {
				prevCommit := string(prevCommitEntry.Value)

				logboek.Context(ctx).Default().LogF("Got previously published commit record %q\n", prevCommit)
				log.L().Debug("Got previously published commit record %q\n", prevCommit) // Remove this debug when tasks log debugged

				prevCommitObj, err := gitRepo.CommitObject(plumbing.NewHash(prevCommit))
				if err != nil {
					return fmt.Errorf("error getting git repo commit object by hash %q: %s", prevCommit, err)
				}

				isAncestor, err := prevCommitObj.IsAncestor(headCommitObj)
				if err != nil {
					return fmt.Errorf("error checking ancestry of commit %q to commit %q: %s", prevCommit, headRef.Hash().String(), err)
				}

				if !isAncestor {
					return fmt.Errorf("cannot publish commit %q which is not desdendant of previously published commit %q", headRef.Hash().String(), prevCommit)
				}
			}
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headRef.Hash().String(), c.TrustedGPGPublicKeys, c.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return fmt.Errorf("signature verification failed: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Verified commit signatures\n")
		log.L().Debug("Verified commit signatures\n") // Remove this debug when tasks log debugged

		cfg, err := GetTrdlChannelsConfig(gitRepo)
		if err != nil {
			return fmt.Errorf("error getting trdl channels config: %s", err)
		}

		cfgDump, _ := yaml.Marshal(cfg)
		logboek.Context(ctx).Default().LogF("Got trdl channels config:\n%s\n---\n", cfgDump)
		log.L().Debug("Got trdl channels config:\n%s\n---\n", cfgDump) // Remove this debug when tasks log debugged

		if err := publisher.PublishChannelsConfig(ctx, publisherRepository, cfg); err != nil {
			return fmt.Errorf("error publishing trdl channels into the repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Published trdl channels config into the repository\n")
		log.L().Debug("Published trdl channels config into the repository\n") // Remove this debug when tasks log debugged

		if err := publisherRepository.Commit(ctx); err != nil {
			return fmt.Errorf("unable to commit new tuf repository state: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Tuf repo commit done\n")
		log.L().Debug("Tuf repo commit done\n") // Remove this debug when tasks log debugged

		if err := storage.Put(ctx, &logical.StorageEntry{Key: PublishedCommitKey, Value: []byte(headRef.Hash().String())}); err != nil {
			return fmt.Errorf("error putting published commit record by key %q: %s", PublishedCommitKey, err)
		}

		logboek.Context(ctx).Default().LogF("Put published commit record %q\n", headRef.Hash().String())
		log.L().Debug("Put published commit record %q\n", headRef.Hash().String()) // Remove this debug when tasks log debugged

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
