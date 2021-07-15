package trdl

import (
	"context"
	"fmt"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/logboek"
	"gopkg.in/yaml.v2"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	storageKeyLastPublishedGitCommit = "last_published_git_commit"

	defaultGitTrdlChannelsBranch = "trdl"
)

func publishPath(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: `publish$`,
		Fields: map[string]*framework.FieldSchema{
			fieldNameGitUsername: {
				Type:        framework.TypeString,
				Description: "Git username",
			},
			fieldNameGitPassword: {
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
	if err := util.CheckRequiredFields(req, fields); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	cfg, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get configuration from storage: %s", err)
	}

	if cfg == nil {
		return errorResponseConfigurationNotFound, nil
	}

	gitCredentialFromStorage, err := getGitCredential(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get git credential from storage: %s", err)
	}

	gitUsername := fields.Get(fieldNameGitUsername).(string)
	gitPassword := fields.Get(fieldNameGitPassword).(string)
	if gitCredentialFromStorage != nil && gitUsername == "" && gitPassword == "" {
		gitUsername = gitCredentialFromStorage.Username
		gitPassword = gitCredentialFromStorage.Password
	}

	gitBranch := cfg.GitTrdlChannelsBranch
	if gitBranch != "" {
		gitBranch = defaultGitTrdlChannelsBranch
	}

	lastPublishedGitCommit := cfg.InitialLastPublishedGitCommit
	{
		entry, err := req.Storage.Get(ctx, storageKeyLastPublishedGitCommit)
		if err != nil {
			return nil, fmt.Errorf("unable to get %q from storage: %s", storageKeyLastPublishedGitCommit, err)
		}

		if entry != nil {
			lastPublishedGitCommit = string(entry.Value)
		}
	}

	publisherRepository, err := b.Publisher.GetRepository(ctx, req.Storage, cfg.RepositoryOptions())
	if err != nil {
		return nil, fmt.Errorf("error getting publisher repository: %s", err)
	}

	taskUUID, err := b.TasksManager.RunTask(context.Background(), req.Storage, func(ctx context.Context, storage logical.Storage) error {
		logboek.Context(ctx).Default().LogF("Started task\n")
		hclog.L().Debug("Started task")

		gitRepo, err := cloneGitRepositoryBranch(cfg.GitRepoUrl, gitBranch, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Cloned git repo\n")
		hclog.L().Debug("Cloned git repo")

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

		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, req.Storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted pgp public keys: %s", err)
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headRef.Hash().String(), trustedPGPPublicKeys, cfg.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return fmt.Errorf("signature verification failed: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Verified commit signatures\n")
		hclog.L().Debug("Verified commit signatures")

		cfg, err := GetTrdlChannelsConfig(gitRepo)
		if err != nil {
			return fmt.Errorf("error getting trdl channels config: %s", err)
		}

		cfgDump, _ := yaml.Marshal(cfg)
		logboek.Context(ctx).Default().LogF("Got trdl channels config:\n%s\n---\n", cfgDump)
		hclog.L().Debug(fmt.Sprintf("Got trdl channels config:\n%s\n---", cfgDump))

		if err := b.Publisher.PublishChannelsConfig(ctx, publisherRepository, cfg); err != nil {
			return fmt.Errorf("error publishing trdl channels into the repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Published trdl channels config into the repository\n")
		hclog.L().Debug("Published trdl channels config into the repository")

		if err := publisherRepository.Commit(ctx); err != nil {
			return fmt.Errorf("unable to commit new tuf repository state: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Tuf repo commit done\n")
		hclog.L().Debug("Tuf repo commit done")

		if err := storage.Put(ctx, &logical.StorageEntry{Key: storageKeyLastPublishedGitCommit, Value: []byte(headRef.Hash().String())}); err != nil {
			return fmt.Errorf("unable to put %q into storage: %s", storageKeyLastPublishedGitCommit, err)
		}

		logboek.Context(ctx).Default().LogF("Put published commit record %q\n", headRef.Hash().String())
		hclog.L().Debug(fmt.Sprintf("Put published commit record %q", headRef.Hash().String()))

		return nil
	})
	if err != nil {
		if err == tasks_manager.ErrBusy {
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

func cloneGitRepositoryBranch(url, gitBranch, username, password string) (*git.Repository, error) {
	cloneGitOptions := trdlGit.CloneOptions{
		BranchName:        gitBranch,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	if username != "" && password != "" {
		cloneGitOptions.Auth = &http.BasicAuth{
			Username: username,
			Password: password,
		}
	}

	gitRepo, err := trdlGit.CloneInMemory(url, cloneGitOptions)
	if err != nil {
		return nil, err
	}

	return gitRepo, nil
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
