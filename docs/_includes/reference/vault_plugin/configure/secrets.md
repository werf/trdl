Configure build secrets.

## Add build secret

| Method | Path |
|--------|------|
| `POST` | `/configure/secrets` |

### Parameters

* `id` (string, required) — id of build secret
* `data` (string, required) — secret data

### Responses

* 200 — OK. 


## Delete build secret

| Method | Path |
|--------|------|
| `DELETE` | `/configure/secrets` |

### Parameters

* `id` (string, required) — id of build secret

### Responses

* 200 — OK. 