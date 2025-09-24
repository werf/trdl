Add or update build signing credentials.

## Add or update mac signing credentials


| Method | Path |
|--------|------|
| `POST` | `/configure/build/mac_signing` |

### Parameters

* `certificate` (string, required) — Certificate data base64 encoded.
* `name` (string, required) — Credentials name.
* `notary_issuer` (string, required) — Notary issuer.
* `notary_key` (string, required) — Notary key ID.
* `notary_key_id` (string, required) — Notary key ID.
* `password` (string, required) — Certificate password.

### Responses

* 200 — OK.
