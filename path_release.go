package trdl

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	git "github.com/go-git/go-git/v5"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/logboek"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/config"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameGitTag = "git_tag"
)

func releasePath(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: `release$`,
		Fields: map[string]*framework.FieldSchema{
			fieldNameGitTag: {
				Type:        framework.TypeString,
				Description: "Project git repository tag which should be released",
				Required:    true,
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
				Callback: b.pathRelease,
				Summary:  pathReleaseHelpSyn,
			},
		},

		HelpSynopsis:    pathReleaseHelpSyn,
		HelpDescription: pathReleaseHelpDesc,
	}
}

func (b *backend) pathRelease(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	c, resp, err := GetAndValidateConfiguration(ctx, req.Storage)
	if resp != nil || err != nil {
		return resp, err
	}

	gitTag := fields.Get(fieldNameGitTag).(string)

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

	publisherRepository, err := GetPublisherRepository(req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting publisher repository: %s", err)
	}

	taskUUID, err := b.TaskQueueManager.RunTask(context.Background(), req.Storage, func(ctx context.Context, storage logical.Storage) error {
		stderr := os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")

		logboek.Context(ctx).Default().LogF("Started task\n")
		fmt.Fprintf(stderr, "Started task\n") // Remove this debug when tasks log debugged

		gitRepo, err := cloneGitRepositoryTag(c.GitRepoUrl, gitTag, gitUsername, gitPassword)
		if err != nil {
			return fmt.Errorf("unable to clone git repository: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Cloned git repo\n")
		fmt.Fprintf(stderr, "Cloned git repo\n") // Remove this debug when tasks log debugged

		if err := trdlGit.VerifyTagSignatures(gitRepo, gitTag, c.TrustedGPGPublicKeys, c.RequiredNumberOfVerifiedSignaturesOnCommit); err != nil {
			return fmt.Errorf("signature verification failed: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Verified tag signatures\n")
		fmt.Fprintf(stderr, "Verified tag signatures\n") // Remove this debug when tasks log debugged

		trdlCfg, err := getTrdlConfig(gitRepo, gitTag)
		if err != nil {
			return fmt.Errorf("unable to get trdl configuration: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Got trdl.yaml configuration\n")

		tarReader, tarWriter := io.Pipe()
		if err := buildReleaseArtifacts(ctx, tarWriter, gitRepo, trdlCfg.DockerImage, trdlCfg.Commands); err != nil {
			return fmt.Errorf("unable to build release artifacts: %s", err)
		}

		logboek.Context(ctx).Default().LogF("Built release artifacts tar archive\n")
		fmt.Fprintf(stderr, "Built release artifacts tar archive\n") // Remove this debug when tasks log debugged

		var fileNames []string
		{
			twArtifacts := tar.NewReader(tarReader)
			for {
				hdr, err := twArtifacts.Next()

				logboek.Context(ctx).Default().LogF("Next tar entry hdr=%#v err=%v\n", hdr, err)
				fmt.Fprintf(stderr, "Next tar entry hdr=%#v err=%v\n", hdr, err) // Remove this debug when tasks log debugged

				if err == io.EOF {
					break
				}

				if err != nil {
					return fmt.Errorf("error reading next tar artifact header: %s", err)
				}

				if hdr.Typeflag != tar.TypeDir {
					logboek.Context(ctx).Default().LogF("Publishing %q into the tuf repo ...\n", hdr.Name)
					fmt.Fprintf(stderr, "Publishing %q into the tuf repo ...\n", hdr.Name) // Remove this debug when tasks log debugged

					if err := publisher.PublishReleaseTarget(ctx, publisherRepository, gitTag, hdr.Name, twArtifacts); err != nil {
						return fmt.Errorf("unable to publish release target %q: %s", hdr.Name, err)
					}

					logboek.Context(ctx).Default().LogF("Published %q into the tuf repo\n", hdr.Name)
					fmt.Fprintf(stderr, "Published %q into the tuf repo\n", hdr.Name) // Remove this debug when tasks log debugged

					fileNames = append(fileNames, hdr.Name)
				}
			}

			if err := publisherRepository.Commit(ctx); err != nil {
				return fmt.Errorf("unable to commit new tuf repository state: %s", err)
			}

			logboek.Context(ctx).Default().LogF("Tuf repo commit done\n")
			fmt.Fprintf(stderr, "Tuf repo commit done\n") // Remove this debug when tasks log debugged
		}

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

func buildReleaseArtifacts(ctx context.Context, tarWriter *io.PipeWriter, gitRepo *git.Repository, fromImage string, runCommands []string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("unable to create docker client: %s", err)
	}

	serviceDirInContext := ".trdl"
	serviceDockerfilePathInContext := path.Join(serviceDirInContext, "Dockerfile")
	contextReader, contextWriter := io.Pipe()
	go func() {
		if err := func() error {
			tw := tar.NewWriter(contextWriter)

			if err := trdlGit.AddWorktreeFilesToTar(tw, gitRepo); err != nil {
				return fmt.Errorf("unable to add git worktree files to tar: %s", err)
			}

			if err := docker.GenerateAndAddDockerfileToTar(tw, serviceDockerfilePathInContext, serviceDirInContext, fromImage, runCommands, true); err != nil {
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

	response, err := cli.ImageBuild(ctx, contextReader, types.ImageBuildOptions{
		Dockerfile:  serviceDockerfilePathInContext,
		PullParent:  true,
		NoCache:     true,
		Remove:      true,
		ForceRemove: true,
		Version:     types.BuilderV1,
	})
	if err != nil {
		return fmt.Errorf("unable to run docker image build: %s", err)
	}

	handleFromImageBuildResponse(response, tarWriter)

	return nil
}

func handleFromImageBuildResponse(response types.ImageBuildResponse, tarWriter *io.PipeWriter) {
	r, w := io.Pipe()
	go func() {
		if err := docker.ReadTarFromImageBuildResponse(w, response); err != nil {
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
	pathReleaseHelpSyn = `
	Performs release of project.
	`

	pathReleaseHelpDesc = `
	Performs release of project by the specified git tag.
	Provided command should prepare release artifacts in the /result directory, which will be published into the TUF repository.
	`
)
