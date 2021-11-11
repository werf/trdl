## `/task/configure`

Configure task manager

### Get task manager configuration



| Method | Path |
|--------|------|
| `GET` | `/task/configure` |


#### Responses

* 200 — OK. 


### Configure task manager



| Method | Path |
|--------|------|
| `POST` | `/task/configure` |

#### Parameters

* `task_timeout` (`integer: 30m`) — Task timeout.
* `task_history_limit` (`integer: 10`) — Task history limit.

#### Responses

* 200 — OK.
