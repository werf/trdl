## `/configure/trusted_pgp_public_key`

Configure trusted PGP public keys

### Get list of trusted PGP public key



| Method | Path |
|--------|------|
| `GET` | `/configure/trusted_pgp_public_key` |

#### Parameters

* `list` (`string: <optional>`) — Return a list if `true`.

#### Responses

* 200 — OK. 


### Add trusted PGP public key



| Method | Path |
|--------|------|
| `POST` | `/configure/trusted_pgp_public_key` |

#### Parameters

* `name` (`string: <required>`) — Key name.
* `public_key` (`string: <required>`) — Key data.

#### Responses

* 200 — OK.
