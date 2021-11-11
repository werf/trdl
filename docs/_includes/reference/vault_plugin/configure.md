## `/configure`

Configure TRDL plugin

### Read plugin configuration



| Method | Path |
|--------|------|
| `GET` | `/configure` |


#### Responses

* 200 — OK. 


### Configure plugin



| Method | Path |
|--------|------|
| `POST` | `/configure` |

#### Parameters

* `initial_last_published_git_commit` (`string: <optional>`) — Initial last published git commit.
* `s3_region` (`string: <required>`) — S3 storage region.
* `git_trdl_channels_branch` (`string: <optional>`) — Special git branch to store trdl_channels.yaml configuration.
* `s3_bucket_name` (`string: <required>`) — S3 storage bucket name.
* `s3_endpoint` (`string: <required>`) — S3 storage endpoint.
* `s3_secret_access_key` (`string: <required>`) — S3 storage access key id.
* `required_number_of_verified_signatures_on_commit` (`integer: <required>`) — Required number of verified signatures on commit.
* `s3_access_key_id` (`string: <required>`) — S3 storage access key id.
* `git_repo_url` (`string: <required>`) — Git repository url.

#### Responses

* 200 — OK. 


### Reset plugin configuration



| Method | Path |
|--------|------|
| `DELETE` | `/configure` |


#### Responses

* 204 — empty body.
