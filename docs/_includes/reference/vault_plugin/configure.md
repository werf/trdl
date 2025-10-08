Configure the plugin.

## Configure plugin


| Method | Path |
|--------|------|
| `POST` | `/configure` |

### Parameters

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
