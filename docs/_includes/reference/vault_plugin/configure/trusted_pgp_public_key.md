Configure trusted PGP public keys.

## Add a trusted PGP public key


| Method | Path |
|--------|------|
| `POST` | `/configure/trusted_pgp_public_key` |

### Parameters

* `name` (string, required) — Key name.
* `public_key` (string, required) — Key data.

### Responses

* 200 — OK. 


## Get the list of trusted PGP public keys


| Method | Path |
|--------|------|
| `GET` | `/configure/trusted_pgp_public_key` |

### Parameters

* `list` (string, optional) — Return a list if `true`.

### Responses

* 200 — OK.
