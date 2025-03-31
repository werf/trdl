package server

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/djherbis/buffer"
	"github.com/djherbis/nio/v3"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/config"
	"github.com/werf/trdl/server/pkg/docker"
	trdlGit "github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/pgp"
	"github.com/werf/trdl/server/pkg/tasks_manager"
	"github.com/werf/trdl/server/pkg/util"
)

const (
	fieldNameGitTag      = "git_tag"
	fieldNameGitUsername = "git_username"
	fieldNameGitPassword = "git_password"
)

func releasePath(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern: `release$`,
		Fields: map[string]*framework.FieldSchema{
			fieldNameGitTag: {
				Type:        framework.TypeString,
				Description: "Git tag",
				Required:    true,
			},
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
				Callback: b.pathRelease,
				Summary:  pathReleaseHelpSyn,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathRelease,
				Summary:  pathReleaseHelpSyn,
			},
		},

		HelpSynopsis:    pathReleaseHelpSyn,
		HelpDescription: pathReleaseHelpDesc,
	}
}

func ValidateReleaseVersion(releaseVersion string) error {
	_, err := semver.NewVersion(releaseVersion)
	if err != nil {
		return fmt.Errorf("expected semver release version got %q: %w", releaseVersion, err)
	}
	return nil
}

func (b *Backend) pathRelease(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	if errResp := util.CheckRequiredFields(req, fields); errResp != nil {
		return errResp, nil
	}

	cfg, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get configuration from storage: %w", err)
	}

	if cfg == nil {
		return errorResponseConfigurationNotFound, nil
	}

	gitCredentialFromStorage, err := trdlGit.GetGitCredential(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get git credential from storage: %w", err)
	}

	gitTag := fields.Get(fieldNameGitTag).(string)
	if err := ValidateReleaseVersion(gitTag); err != nil {
		return logical.ErrorResponse("%s validation failed: %s", fieldNameGitTag, err), nil
	}
	releaseName := strings.TrimPrefix(gitTag, "v")

	gitUsername := fields.Get(fieldNameGitUsername).(string)
	gitPassword := fields.Get(fieldNameGitPassword).(string)
	if gitCredentialFromStorage != nil && gitUsername == "" && gitPassword == "" {
		gitUsername = gitCredentialFromStorage.Username
		gitPassword = gitCredentialFromStorage.Password
	}

	opts := cfg.RepositoryOptions()
	opts.InitializeTUFKeys = true
	opts.InitializePGPSigningKey = true
	publisherRepository, err := b.Publisher.GetRepository(ctx, req.Storage, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting publisher repository: %w", err)
	}

	taskUUID, err := b.TasksManager.RunTask(context.Background(), req.Storage, func(ctx context.Context, storage logical.Storage) error {
		logboek.Context(ctx).Default().LogF("Started task\n")
		b.Logger().Debug("Started task")

		logboek.Context(ctx).Default().LogF("Cloning git repo\n")
		b.Logger().Debug("Cloning git repo")

		gitRepo, err := cloneGitRepositoryTag(cfg.GitRepoUrl, gitTag, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %w", err)
		}

		logboek.Context(ctx).Default().LogF("Verifying tag PGP signatures of the git tag %q\n", gitTag)
		b.Logger().Debug(fmt.Sprintf("Verifying tag PGP signatures of the git tag %q", gitTag))

		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, req.Storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted PGP public keys: %w", err)
		}

		b.Logger().Debug(fmt.Sprintf("[DEBUG-SIGNATURES] trustedPGPPublicKeys >%v<", trustedPGPPublicKeys))
		if err := trdlGit.VerifyTagSignatures(gitRepo, gitTag, trustedPGPPublicKeys, cfg.RequiredNumberOfVerifiedSignaturesOnCommit, b.Logger()); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}

		logboek.Context(ctx).Default().LogF("Getting trdl.yaml configuration from the git tag %q\n", gitTag)
		b.Logger().Debug(fmt.Sprintf("Getting trdl.yaml configuration from the git tag %q\n", gitTag))

		trdlCfg, err := getTrdlConfig(gitRepo, gitTag, cfg.GitTrdlPath)
		if err != nil {
			return fmt.Errorf("unable to get trdl configuration: %w", err)
		}

		logboek.Context(ctx).Default().LogF("Starting release artifacts tar archive build\n")
		b.Logger().Debug("Starting release artifacts tar archive build")

		tarBuf := buffer.New(64 * 1024 * 1024)
		tarReader, tarWriter := nio.Pipe(tarBuf)

		errCh := make(chan error, 1)
		go func() {
			err := docker.BuildReleaseArtifacts(ctx,
				docker.BuildReleaseArtifactsOpts{
					TarWriter:    tarWriter,
					GitRepo:      gitRepo,
					FromImage:    trdlCfg.GetDockerImage(),
					RunCommands:  trdlCfg.Commands,
					Storage:      req.Storage,
					BuilderImage: trdlCfg.BuilderImage,
				}, b.Logger())
			if err != nil {
				errCh <- err
				tarWriter.CloseWithError(err)
				return
			}
			errCh <- nil
		}()

		{
			logboek.Context(ctx).Default().LogF("Starting to read tar artifacts...\n")
			b.Logger().Debug("Starting to read tar artifacts...")
			twArtifacts := tar.NewReader(tarReader)
			for {
				hdr, err := twArtifacts.Next()

				if err == io.EOF {
					break
				}

				if err != nil {
					return fmt.Errorf("error reading next tar artifact header: %w", err)
				}

				if strings.HasPrefix(hdr.Name, docker.ContainerArtifactsDir+"/") && hdr.Typeflag != tar.TypeDir {
					name := strings.TrimPrefix(hdr.Name, docker.ContainerArtifactsDir+"/")
					logboek.Context(ctx).Default().LogF("Publishing %q into the tuf repo ...\n", name)
					b.Logger().Debug(fmt.Sprintf("Publishing %q into the tuf repo ...", name))

					if err := b.Publisher.StageReleaseTarget(ctx, publisherRepository, releaseName, name, twArtifacts); err != nil {
						return fmt.Errorf("unable to publish release target %q: %w", name, err)
					}
				}
			}

			if err := <-errCh; err != nil {
				return fmt.Errorf("unable to build release artifacts: %w", err)
			}
		}

		logboek.Context(ctx).Default().LogF("Committing TUF repository state\n")
		b.Logger().Debug("Committing TUF repository state")

		if err := publisherRepository.CommitStaged(ctx); err != nil {
			return fmt.Errorf("unable to commit new tuf repository state: %w", err)
		}

		logboek.Context(ctx).Default().LogF("Task finished\n")
		b.Logger().Debug("Task finished")

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

func cloneGitRepositoryTag(url, gitTag, username, password string) (*git.Repository, error) {
	cloneGitOptions := trdlGit.CloneOptions{
		TagName:           gitTag,
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

func getTrdlConfig(gitRepo *git.Repository, gitTag, trdlPath string) (*config.Trdl, error) {
	if trdlPath == "" {
		trdlPath = config.DefaultTrdlPath
	}

	data, err := trdlGit.ReadWorktreeFile(gitRepo, trdlPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read worktree file %q: %w", trdlPath, err)
	}

	values := map[string]interface{}{
		"Tag": gitTag,
	}

	cfg, err := config.ParseTrdl(data, values)
	if err != nil {
		return nil, fmt.Errorf("error parsing %q configuration file: %w", trdlPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("error validation %q configuration file: %w", trdlPath, err)
	}

	return cfg, nil
}

const (
	pathReleaseHelpSyn  = "Perform a release"
	pathReleaseHelpDesc = "Perform a release for the specified git tag"
)
