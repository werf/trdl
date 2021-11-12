Configure server PGP signing keys.

## Get public part of PGP signing key


| Method | Path |
|--------|------|
| `GET` | `/configure/pgp_signing_key` |


### Responses

* 200 — OK. 


## Delete current PGP signing key

Delete current PGP signing key (new key will be generated automatically on demand)


| Method | Path |
|--------|------|
| `DELETE` | `/configure/pgp_signing_key` |


### Responses

* 204 — empty body.
