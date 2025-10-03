Add or update build signing credentials.

## Add or update mac signing credentials


| Method | Path |
|--------|------|
| `POST` | `/configure/build/mac_signing_identity` |

### Parameters

* `data` (string, required) — Certificate data base64 encoded.
* `notary_issuer` (string, required) — Notary issuer.
* `notary_key` (string, required) — Notary key.
* `notary_key_id` (string, required) — Notary key ID.
* `password` (string, required) — Certificate password.

### Responses

* 200 — OK. 


## Delete mac signing credentials


| Method | Path |
|--------|------|
| `DELETE` | `/configure/build/mac_signing_identity` |


### Responses

* 204 — empty body.
