Configure task manager.

## Configure task manager

| Method | Path |
|--------|------|
| `POST` | `/task/configure` |

### Parameters

* `task_history_limit` (integer, optional, default: `10`) — Task history limit.
* `task_timeout` (integer, optional, default: `30m`) — Task timeout.

### Responses

* 200 — OK. 


## Get task manager configuration

| Method | Path |
|--------|------|
| `GET` | `/task/configure` |


### Responses

* 200 — OK.
