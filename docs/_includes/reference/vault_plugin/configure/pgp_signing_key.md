Configure a PGP key for signing release artifacts.

## Get the public part of the current PGP signing key


| Method | Path |
|--------|------|
| `GET` | `/configure/pgp_signing_key` |


### Responses

* 200 — OK. 


## Delete the current PGP signing key

Delete the current PGP signing key (new key will be generated automatically on demand)


| Method | Path |
|--------|------|
| `DELETE` | `/configure/pgp_signing_key` |


### Responses

* 204 — empty body.
