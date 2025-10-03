The trdl plugin builds and releases software versions, publishes the release channels, and ensures object storage security via saving data signed by keys (no one has access to those keys) and continuously rotating TUF keys and metadata.

## Paths

* [`/configure`]({{ "/reference/vault_plugin/configure.html" | true_relative_url }}) — configure the plugin.

* [`/configure/build/mac_signing_identity`]({{ "/reference/vault_plugin/configure/build/mac_signing_identity.html" | true_relative_url }}) — add or update build signing credentials.

* [`/configure/build/secrets`]({{ "/reference/vault_plugin/configure/build/secrets.html" | true_relative_url }}) — add a build secret.

* [`/configure/build/secrets/:id`]({{ "/reference/vault_plugin/configure/build/secrets/id.html" | true_relative_url }}) — delete a build secret.

* [`/configure/git_credential`]({{ "/reference/vault_plugin/configure/git_credential.html" | true_relative_url }}) — configure git credentials.

* [`/configure/last_published_git_commit`]({{ "/reference/vault_plugin/configure/last_published_git_commit.html" | true_relative_url }}) — read or delete the last published git commit.

* [`/configure/pgp_signing_key`]({{ "/reference/vault_plugin/configure/pgp_signing_key.html" | true_relative_url }}) — configure a pgp key for signing release artifacts.

* [`/configure/trusted_pgp_public_key`]({{ "/reference/vault_plugin/configure/trusted_pgp_public_key.html" | true_relative_url }}) — configure trusted pgp public keys.

* [`/configure/trusted_pgp_public_key/:name`]({{ "/reference/vault_plugin/configure/trusted_pgp_public_key/name.html" | true_relative_url }}) — read or delete the configured trusted pgp public key.

* [`/publish`]({{ "/reference/vault_plugin/publish.html" | true_relative_url }}) — publish release channels.

* [`/release`]({{ "/reference/vault_plugin/release.html" | true_relative_url }}) — perform a release.

* [`/task`]({{ "/reference/vault_plugin/task.html" | true_relative_url }}) — get tasks.

* [`/task/configure`]({{ "/reference/vault_plugin/task/configure.html" | true_relative_url }}) — configure the task manager.

* [`/task/:uuid`]({{ "/reference/vault_plugin/task/uuid.html" | true_relative_url }}) — get task status.

* [`/task/:uuid/cancel`]({{ "/reference/vault_plugin/task/uuid/cancel.html" | true_relative_url }}) — cancel the running task.

* [`/task/:uuid/log`]({{ "/reference/vault_plugin/task/uuid/log.html" | true_relative_url }}) — get the task log.
