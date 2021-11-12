Configure git credentials.

## Configure git credential


| Method | Path |
|--------|------|
| `POST` | `/configure/git_credential` |

### Parameters

* `password` (string, optional) — Git password. Required for CREATE, UPDATE..
* `username` (string, optional) — Git username. Required for CREATE, UPDATE..

### Responses

* 200 — OK. 


## Reset git credential


| Method | Path |
|--------|------|
| `DELETE` | `/configure/git_credential` |


### Responses

* 204 — empty body.
