package server

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"
	"github.com/werf/logboek"

	"github.com/werf/vault-plugin-secrets-trdl/server/pkg/config"
	"github.com/werf/vault-plugin-secrets-trdl/server/pkg/docker"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/server/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/server/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/server/pkg/tasks_manager"
	"github.com/werf/vault-plugin-secrets-trdl/server/pkg/util"
)

const (
	fieldNameGitTag      = "git_tag"
	fieldNameGitUsername = "git_username"
	fieldNameGitPassword = "git_password"
)

func releasePath(b *backend) *framework.Path {
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

func ValidateGitTag(gitTag string) error {
	_, err := semver.NewVersion(gitTag)
	if err != nil {
		return fmt.Errorf("expected semver release name got %q: %s", gitTag, err)
	}
	return nil
}

func (b *backend) pathRelease(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
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

	gitTag := fields.Get(fieldNameGitTag).(string)
	if err := ValidateGitTag(gitTag); err != nil {
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

		gitRepo, err := cloneGitRepositoryTag(cfg.GitRepoUrl, gitTag, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Verifying tag PGP signatures of the git tag %q\n", gitTag)
		hclog.L().Debug("Verifying tag PGP signatures of the git tag %q", gitTag)

		trustedPGPPublicKeys, err := pgp.GetTrustedPGPPublicKeys(ctx, req.Storage)
		if err != nil {
			return fmt.Errorf("unable to get trusted pgp public keys: %s", err)
		}

		if err := trdlGit.VerifyTagSignatures(gitRepo, gitTag, trustedPGPPublicKeys, cfg.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return fmt.Errorf("signature verification failed: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Getting trdl.yaml configuration from the git tag %q\n", gitTag)
		hclog.L().Debug(fmt.Sprintf("Getting trdl.yaml configuration from the git tag %q\n", gitTag))

		trdlCfg, err := getTrdlConfig(gitRepo, gitTag)
		if err != nil {
			return fmt.Errorf("unable to get trdl configuration: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Starting release artifacts tar archive build\n")
		hclog.L().Debug("Starting release artifacts tar archive build")

		tarReader, tarWriter := io.Pipe()
		err, cleanupFunc := buildReleaseArtifacts(ctx, tarWriter, gitRepo, trdlCfg.DockerImage, trdlCfg.Commands)
		if err != nil {
			return fmt.Errorf("unable to build release artifacts: %s", err)
		}
		defer func() {
			if err := cleanupFunc(); err != nil {
				hclog.L().Error(fmt.Sprintf("unable to remove service docker image: %s", err))
			}
		}()

		{
			twArtifacts := tar.NewReader(tarReader)
			for {
				hdr, err := twArtifacts.Next()

				if err == io.EOF {
					break
				}

				if err != nil {
					return fmt.Errorf("error reading next tar artifact header: %s", err)
				}

				if hdr.Typeflag != tar.TypeDir {
					logboek.Context(ctx).Default().LogF("Publishing %q into the tuf repo ...\n", hdr.Name)
					hclog.L().Debug(fmt.Sprintf("Publishing %q into the tuf repo ...", hdr.Name))

					if err := b.Publisher.StageReleaseTarget(ctx, publisherRepository, releaseName, hdr.Name, twArtifacts); err != nil {
						return fmt.Errorf("unable to publish release target %q: %s", hdr.Name, err)
					}
				}
			}

			logboek.Context(ctx).Default().LogF("Committing TUF repository state\n")
			hclog.L().Debug("Committing TUF repository state")

			if err := publisherRepository.CommitStaged(ctx); err != nil {
				return fmt.Errorf("unable to commit new tuf repository state: %s", err)
			}
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

func getTrdlConfig(gitRepo *git.Repository, gitTag string) (*config.Trdl, error) {
	data, err := trdlGit.ReadWorktreeFile(gitRepo, config.TrdlFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read worktree file %q: %s", config.TrdlFileName, err)
	}

	values := map[string]interface{}{
		"Tag": gitTag,
	}

	cfg, err := config.ParseTrdl(data, values)
	if err != nil {
		return nil, fmt.Errorf("error parsing %q configuration file: %s", config.TrdlFileName, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("error validation %q configuration file: %s", config.TrdlFileName, err)
	}

	return cfg, nil
}

func buildReleaseArtifacts(ctx context.Context, tarWriter *io.PipeWriter, gitRepo *git.Repository, fromImage string, runCommands []string) (error, func() error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("unable to create docker client: %s", err), nil
	}

	serviceDirInContext := ".trdl"
	serviceDockerfilePathInContext := path.Join(serviceDirInContext, "Dockerfile")
	serviceLabels := map[string]string{
		"vault-trdl-release-uuid": uuid.NewV4().String(),
	}
	contextReader, contextWriter := io.Pipe()
	go func() {
		if err := func() error {
			tw := tar.NewWriter(contextWriter)

			logboek.Context(ctx).Default().LogF("Adding git worktree files to the build context\n")
			hclog.L().Debug("Adding git worktree files to the build context")

			if err := trdlGit.AddWorktreeFilesToTar(tw, gitRepo); err != nil {
				return fmt.Errorf("unable to add git worktree files to tar: %s", err)
			}

			dockerfileOpts := docker.DockerfileOpts{
				WithArtifacts: true,
				Labels:        serviceLabels,
			}
			if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePathInContext, fromImage, runCommands, dockerfileOpts); err != nil {
				return fmt.Errorf("unable to add service dockerfile to tar: %s", err)
			}

			if err := tw.Close(); err != nil {
				return fmt.Errorf("unable to close tar writer: %s", err)
			}

			return nil
		}(); err != nil {
			if closeErr := contextWriter.CloseWithError(err); closeErr != nil {
				panic(closeErr)
			}
			return
		}

		if err := contextWriter.Close(); err != nil {
			panic(err)
		}
	}()

	logboek.Context(ctx).Default().LogF("Building docker image with artifacts\n")
	hclog.L().Debug("Building docker image with artifacts")

	response, err := cli.ImageBuild(ctx, contextReader, types.ImageBuildOptions{
		Dockerfile:  serviceDockerfilePathInContext,
		PullParent:  true,
		NoCache:     true,
		Remove:      true,
		ForceRemove: true,
		Version:     types.BuilderV1,
	})
	if err != nil {
		return fmt.Errorf("unable to run docker image build: %s", err), nil
	}

	handleFromImageBuildResponse(ctx, response, tarWriter)

	cleanupFunc := func() error {
		return docker.RemoveImagesByLabels(ctx, cli, serviceLabels)
	}

	return nil, cleanupFunc
}

func handleFromImageBuildResponse(ctx context.Context, response types.ImageBuildResponse, tarWriter *io.PipeWriter) {
	r, w := io.Pipe()
	go func() {
		if err := docker.ReadTarFromImageBuildResponse(w, logboek.Context(ctx).OutStream(), response); err != nil {
			if closeErr := w.CloseWithError(err); closeErr != nil {
				panic(closeErr)
			}
			return
		}

		if err := w.Close(); err != nil {
			panic(err)
		}
	}()

	go func() {
		decoder := base64.NewDecoder(base64.StdEncoding, r)
		if _, err := io.Copy(tarWriter, decoder); err != nil {
			if closeErr := tarWriter.CloseWithError(err); closeErr != nil {
				panic(closeErr)
			}
			return
		}

		if err := w.Close(); err != nil {
			panic(err)
		}
	}()
}

const (
	pathReleaseHelpSyn  = "Perform a release"
	pathReleaseHelpDesc = "Perform a release for the specified git tag"
)
