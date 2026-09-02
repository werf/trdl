Configure the plugin.

## Configure plugin


| Method | Path |
|--------|------|
| `POST` | `/configure` |

### Parameters

* `buildkitd_address` (string, optional) — An address of a running buildkitd (unix://, tcp://, docker-container:// or kube-pod:// scheme) to build release artifacts with the BuildKit client; the docker CLI is used only when neither this nor buildkitd_driver is set. Build secrets are sent to that daemon, and tcp:// is neither encrypted nor authenticated, so securing the channel and isolating the daemon is the administrator's responsibility.
* `buildkitd_driver` (string, optional) — Provision an ephemeral buildkitd per build instead of using the docker CLI: kubernetes runs it as a pod and needs no docker binary next to the plugin. Cannot be combined with buildkitd_address, buildx_driver or buildx_driver_opts. A TRDL_BUILDKITD_ADDRESS set on the process wins over a stored driver, and the build reports the driver as unused.
* `buildkitd_driver_opts` (array, optional) — The buildkitd driver options, one name=value pair per element (e.g. namespace=trdl-build); they require buildkitd_driver to be set. The kubernetes driver accepts annotations, deadline, image, labels, limits.cpu, limits.ephemeral-storage, limits.memory, namespace, nodeselector, requests.cpu, requests.ephemeral-storage, requests.memory, rootless, serviceaccount and timeout; anything else is rejected. When deadline is not set, it defaults to the release task's remaining time at pod creation plus a five-minute margin, so a plugin crash cannot leave the builder pod running indefinitely.
* `buildx_driver` (string, optional) — The buildx driver to build release artifacts with: docker-container (used by default) or kubernetes. Takes precedence over the TRDL_BUILDX_DRIVER environment variable, and cannot be combined with buildkitd_address or buildkitd_driver.
* `buildx_driver_opts` (array, optional) — The buildx driver options, one --driver-opt per element (e.g. namespace=trdl-build), passed through as is. Take precedence over the TRDL_BUILDX_DRIVER_OPTS_* environment variables, and cannot be combined with buildkitd_address or buildkitd_driver.
* `git_repo_url` (string, required) — URL of the Git repository.
* `git_trdl_channels_branch` (string, optional) — A special Git branch to store the trdl channels configuration file.
* `git_trdl_channels_path` (string, optional) — A path in the Git repository to the trdl channels configuration file (trdl_channels.yaml is used by default).
* `git_trdl_path` (string, optional) — A path in the Git repository to the release trdl configuration file (trdl.yaml is used by default).
* `initial_last_published_git_commit` (string, optional) — The initial commit for the last successful publication.
* `required_number_of_verified_signatures_on_commit` (integer, required) — The required number of verified signatures for a commit.
* `s3_access_key_id` (string, required) — The S3 storage access key id.
* `s3_bucket_name` (string, required) — The S3 storage bucket name.
* `s3_endpoint` (string, required) — The S3 storage endpoint.
* `s3_region` (string, required) — The S3 storage region.
* `s3_secret_access_key` (string, required) — The S3 storage secret access key.

### Responses

* 200 — OK. 


## Read the plugin configuration


| Method | Path |
|--------|------|
| `GET` | `/configure` |


### Responses

* 200 — OK. 


## Reset the plugin configuration


| Method | Path |
|--------|------|
| `DELETE` | `/configure` |


### Responses

* 204 — empty body.
