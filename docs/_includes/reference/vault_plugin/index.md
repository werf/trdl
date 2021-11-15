The TRDL backend plugin allows publishing of project's releases into the TUF compatible repository.

## Paths

* [`/configure`]({{ "/reference/vault_plugin/configure.html" | true_relative_url }}) — configure trdl plugin.

* [`/configure/git_credential`]({{ "/reference/vault_plugin/configure/git_credential.html" | true_relative_url }}) — configure git credentials.

* [`/configure/pgp_signing_key`]({{ "/reference/vault_plugin/configure/pgp_signing_key.html" | true_relative_url }}) — configure server pgp signing keys.

* [`/configure/trusted_pgp_public_key`]({{ "/reference/vault_plugin/configure/trusted_pgp_public_key.html" | true_relative_url }}) — configure trusted pgp public keys.

* [`/configure/trusted_pgp_public_key/:name`]({{ "/reference/vault_plugin/configure/trusted_pgp_public_key/name.html" | true_relative_url }}) — read or delete configured trusted pgp public key.

* [`/publish`]({{ "/reference/vault_plugin/publish.html" | true_relative_url }}) — publish release channels.

* [`/release`]({{ "/reference/vault_plugin/release.html" | true_relative_url }}) — perform a release.

* [`/task`]({{ "/reference/vault_plugin/task.html" | true_relative_url }}) — get tasks.

* [`/task/configure`]({{ "/reference/vault_plugin/task/configure.html" | true_relative_url }}) — configure task manager.

* [`/task/:uuid`]({{ "/reference/vault_plugin/task/uuid.html" | true_relative_url }}) — get task status.

* [`/task/:uuid/cancel`]({{ "/reference/vault_plugin/task/uuid/cancel.html" | true_relative_url }}) — cancel running task.

* [`/task/:uuid/log`]({{ "/reference/vault_plugin/task/uuid/log.html" | true_relative_url }}) — get task log.
