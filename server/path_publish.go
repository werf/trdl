package server

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

	"github.com/werf/trdl/server/pkg/config"
	trdlGit "github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/pgp"
	"github.com/werf/trdl/server/pkg/tasks_manager"
	"github.com/werf/trdl/server/pkg/util"
)

const (
	storageKeyLastPublishedGitCommit = "last_published_git_commit"

	defaultGitTrdlChannelsBranch = "trdl"
)

func publishPath(b *Backend) *framework.Path {
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
			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathPublish,
				Summary:  pathPublishHelpSyn,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathPublish,
				Summary:  pathPublishHelpSyn,
			},
		},

		HelpSynopsis:    pathPublishHelpSyn,
		HelpDescription: pathPublishHelpDesc,
	}
}

func (b *Backend) pathPublish(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	cfg, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get configuration from storage: %s", err)
	}

	if cfg == nil {
		return errorResponseConfigurationNotFound, nil
	}

	gitCredentialFromStorage, err := trdlGit.GetGitCredential(ctx, req.Storage)
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
	if gitBranch == "" {
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

	opts := cfg.RepositoryOptions()
	opts.InitializeKeys = true
	publisherRepository, err := b.Publisher.GetRepository(ctx, req.Storage, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting publisher repository: %s", err)
	}

	taskUUID, err := b.TasksManager.RunTask(context.Background(), req.Storage, func(ctx context.Context, storage logical.Storage) error {
		logboek.Context(ctx).Default().LogF("Started task\n")
		hclog.L().Debug("Started task")

		logboek.Context(ctx).Default().LogF("Cloning git repo\n")
		hclog.L().Debug("Cloning git repo")

		gitRepo, err := cloneGitRepositoryBranch(cfg.GitRepoUrl, gitBranch, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %s", err)
		}

		headRef, err := gitRepo.Head()
		if err != nil {
			return fmt.Errorf("error getting git repo branch %q head reference: %s", gitBranch, err)
		}
		headCommit := headRef.Hash().String()

		if lastPublishedGitCommit == headCommit {
			logboek.Context(ctx).Default().LogF("Head commit %q not changed: skipping publish task\n", headCommit)
			hclog.L().Debug(fmt.Sprintf("Head commit %q not changed: skipping publish task", headCommit))

			return nil
		}

		if lastPublishedGitCommit != "" {
			logboek.Context(ctx).Default().LogF("Checking previously published commit %q is ancestor to the current head commit %q\n", lastPublishedGitCommit, headCommit)
			hclog.L().Debug(fmt.Sprintf("Checking previously published commit %q is ancestor to the current head commit %q", lastPublishedGitCommit, headCommit))

			isAncestor, err := trdlGit.IsAncestor(gitRepo, lastPublishedGitCommit, headRef.Hash().String())
			if err != nil {
				return err
			}

			if !isAncestor {
				return fmt.Errorf("cannot publish git commit %q which is not desdendant of previously published git commit %q", headRef.Hash().String(), lastPublishedGitCommit)
			}
		}

		logboek.Context(ctx).Default().LogF("Verifying tag PGP signatures of the commit %q\n", headCommit)
		hclog.L().Debug("Verifying tag PGP signatures of the commit %q", headCommit)

		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, req.Storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted pgp public keys: %s", err)
		}

		if err := trdlGit.VerifyCommitSignatures(gitRepo, headRef.Hash().String(), trustedPGPPublicKeys, cfg.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return fmt.Errorf("signature verification failed: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Verified commit signatures\n")
		hclog.L().Debug("Verified commit signatures")

		logboek.Context(ctx).Default().LogF("Getting trdl_channels.yaml configuration from the commit %q\n", headCommit)
		hclog.L().Debug(fmt.Sprintf("Getting trdl_channels.yaml configuration from the commit %q\n", headCommit))

		cfg, err := GetTrdlChannelsConfig(gitRepo)
		if err != nil {
			return fmt.Errorf("error getting trdl channels config: %s", err)
		}

		cfgDump, _ := yaml.Marshal(cfg)
		logboek.Context(ctx).Default().LogF("Got trdl channels config:\n%s\n---\n", cfgDump)
		hclog.L().Debug(fmt.Sprintf("Got trdl channels config:\n%s\n---", cfgDump))

		logboek.Context(ctx).Default().LogF("Publishing trdl channels config into the TUF repository\n")
		hclog.L().Debug("Publishing trdl channels config into the TUF repository")

		if err := b.Publisher.StageChannelsConfig(ctx, publisherRepository, cfg); err != nil {
			return fmt.Errorf("error publishing trdl channels into the repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Committing TUF repository state\n")
		hclog.L().Debug("Committing TUF repository state")

		if err := publisherRepository.CommitStaged(ctx); err != nil {
			return fmt.Errorf("unable to commit new tuf repository state: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Storing published commit record %q into the storage\n", headCommit)
		hclog.L().Debug(fmt.Sprintf("Storing published commit record %q into the storage", headCommit))

		if err := storage.Put(ctx, &logical.StorageEntry{Key: storageKeyLastPublishedGitCommit, Value: []byte(headCommit)}); err != nil {
			return fmt.Errorf("unable to put %q into storage: %s", storageKeyLastPublishedGitCommit, err)
		}

		logboek.Context(ctx).Default().LogF("Task finished\n")
		hclog.L().Debug("Task finished")

		return nil
	})
	if err != nil {
		if err == tasks_manager.ErrBusy {
			return logical.ErrorResponse("busy"), nil
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
	pathPublishHelpSyn  = "Publish release channels"
	pathPublishHelpDesc = "Publish release channels based on trdl_channels.yaml configuration in the git repository"
)
