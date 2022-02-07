Read or delete the configured trusted PGP public key.

## Get the trusted PGP public key


| Method | Path |
|--------|------|
| `GET` | `/configure/trusted_pgp_public_key/:name` |

### Parameters

* `name` (url pattern, required) — Key name.
* `list` (string, optional) — Return a list if `true`.

### Responses

* 200 — OK. 


## Delete the trusted PGP public key


| Method | Path |
|--------|------|
| `DELETE` | `/configure/trusted_pgp_public_key/:name` |

### Parameters

* `name` (url pattern, required) — Key name.

### Responses

* 204 — empty body.
