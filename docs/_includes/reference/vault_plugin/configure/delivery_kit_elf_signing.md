Add or update Delivery Kit elf signing settings.

## Add or update ELF signing settings


| Method | Path |
|--------|------|
| `POST` | `/configure/delivery_kit_elf_signing` |

### Parameters

* `certificate` (string, required) — Certificate data base64 encoded.
* `intermediates` (string, optional) — Intermediates certificates data base64 encoded.
* `key` (string, required) — Private key data base64 encoded or Vault url.
* `password` (string, optional) — Private key password.
* `vault_addr` (string, optional) — Vault server address.
* `vault_auth_path` (string, optional) — Vault auth path.
* `vault_auth_role_id` (string, optional) — Vault auth role id.
* `vault_auth_secret_id` (string, optional) — Vault auth secret id.
* `vault_transit_path` (string, optional) — Vault transit path.

### Responses

* 200 — OK. 


## Delete ELF signing settings


| Method | Path |
|--------|------|
| `DELETE` | `/configure/delivery_kit_elf_signing` |


### Responses

* 204 — empty body.
