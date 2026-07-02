Configure ELF binary signing via Delivery Kit. Requires `objcopy` with multi-architecture support on the Vault server host (`binutils-multiarch` on Debian/Ubuntu).

## Configure ELF signing


| Method | Path |
|--------|------|
| `POST` | `/configure/delivery_kit_elf_signing` |

### Parameters

* `certificate` (string, required) — Certificate data base64 encoded.
* `intermediates` (string, optional) — Certificate chain (intermediates and root) base64 encoded, as a single PEM bundle.
* `key` (string, required) — Private key data base64 encoded or a Vault key reference in the form hashivault://<key>. When a hashivault:// reference is used, configure the vault_* parameters.
* `password` (string, optional) — Private key password. Ignored when key is a hashivault:// reference.
* `vault_addr` (string, optional) — Vault server address. Applies only when key is a hashivault:// reference.
* `vault_auth_path` (string, optional, default: `ar`) — Mount path of Vault auth method. Applies only when key is a hashivault:// reference.
* `vault_auth_role_id` (string, optional) — AppRole RoleID used to authenticate to Vault. Applies only when key is a hashivault:// reference.
* `vault_auth_secret_id` (string, optional) — AppRole SecretID used to authenticate to Vault. Applies only when key is a hashivault:// reference.
* `vault_transit_path` (string, optional) — Mount path of Vault transit engine. Applies only when key is a hashivault:// reference.

### Responses

* 200 — OK. 


## Reset ELF signing


| Method | Path |
|--------|------|
| `DELETE` | `/configure/delivery_kit_elf_signing` |


### Responses

* 204 — empty body.
