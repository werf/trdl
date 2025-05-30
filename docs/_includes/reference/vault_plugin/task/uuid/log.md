Get the task log.

## Get the task log


| Method | Path              |
| ------ | ----------------- |
| `GET`  | `/task/:uuid/log` |

### Parameters

* `uuid` (url pattern, required) — Task UUID.
* `offset` (query, optional) — Number of characters to skip from the beginning of the log. Defaults to 0.
* `limit` (query, optional) — Maximum number of characters to return. Defaults to 500.

### Responses

* 200 — OK.
