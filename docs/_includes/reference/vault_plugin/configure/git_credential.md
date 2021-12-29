Configure Git credentials.

## Configure Git credentials


| Method | Path |
|--------|------|
| `POST` | `/configure/git_credential` |

### Parameters

* `password` (string, optional) — A Git password; Required for CREATE, UPDATE..
* `username` (string, optional) — A Git username; Required for CREATE, UPDATE..

### Responses

* 200 — OK. 


## Reset Git credentials


| Method | Path |
|--------|------|
| `DELETE` | `/configure/git_credential` |


### Responses

* 204 — empty body.
