---
title: Quickstart
permalink: quickstart.html
layout: page-nosidebar
toc: true
toc_headers: h2
---

## For an administrator

> We assume that you are already familiar with Vault and know how to use it, so let us skip the details of setting it up. We also assume that Vault is configured according to the [official documentation](https://learn.hashicorp.com/tutorials/vault/deployment-guide).

### Vault

There are a couple of ways to install Vault and the trdl plugin. The easiest one is to use the ready-made Vault binary (you can get one at the Vault website or install it using your distribution's package manager) and the ready-made trdl plugin binary.

### Docker

Install Docker. Add a Vault user to the Docker group:

```shell
usermod -a -G docker vault
```

### Setting up the project

#### Git repository

Create a regular public Git repository.

#### Bucket

Any S3-compatible bucket will do. It should be publicly available for reading.

{% offtopic title="A note about GCS (Google Cloud Storage)" %}
To get rid of the `An error occurred (AccessDenied) when calling the CreateMultipartUpload operation: Access denied` error, make sure that the Service Account used to access the bucket belongs to the `Storage Admin` role.
{% endofftopic %}

### Installing the plugin

Download the trdl plugin by following the instructions in the message of the [selected release](https://github.com/werf/trdl/releases). Copy it to `/etc/vault.d/plugins` or another directory where you normally store plugins.

### Configuring the plugin

Setting up Vault includes specifying the directory where the plugins are stored:

```shell
plugin_directory = "/etc/vault.d/plugins"
```

Restart Vault.

Register the plugin in Vault:

```shell
vault plugin register -sha256=$(sha256sum /etc/vault.d/plugins/vault-plugin-secrets-trdl | awk '{print $1}') secret vault-plugin-secrets-trdl
```

In our case, the plugin file is called `vault-plugin-secrets-trdl`, and we register it under the same name in Vault. Refer to the [official documentation](https://www.vaultproject.io/docs/commands/plugin/register) to learn more about registering plugins.

Enable the plugin as a `secrets engine` at a specific path in Vault:

```shell
vault secrets enable -path=trdl-test-project vault-plugin-secrets-trdl
```

You can enable the same plugin many times; however, you must use a unique path each time. For more information, refer to the [official documentation](https://www.vaultproject.io/docs/commands/secrets/enable).

Now let's configure the trdl plugin itself. We will use the [/configure](/reference/vault_plugin/configure.html#configure-plugin) API method to do this:

```shell
vault write trdl-test-project/configure @configuration.json
```

where `configuration.json` has the following contents:
```json
{
  "s3_secret_access_key": "FOO",
  "s3_access_key_id": "BAR",
  "s3_bucket_name": "trdl-test-project-tuf",
  "s3_region": "europe-west1",
  "s3_endpoint": "https://storage.googleapis.com",
  "git_repo_url": "https://github.com/werf/trdl-test-project",
  "required_number_of_verified_signatures_on_commit": 2
}
```

> When configuring the plugin, you must specify the minimum number of GPG signatures required for a commit (`required_number_of_verified_signatures_on_commit`). Otherwise, the release system becomes vulnerable: any user with access can tamper with it because the system is not protected by a quorum.

The minimum number of GPG signatures required (`required_number_of_verified_signatures_on_commit`) depends on the size and scope of the team, frequency of updates, and other factors.

#### Managing public parts of trusted GPG keys

The [/configure/trusted_pgp_public_key](/reference/vault_plugin/configure/trusted_pgp_public_key.html) group of API methods is used to handle the public parts of trusted GPG keys.

## For a developer

### Setting up a GPG signature in Git

Git has a mechanism for signing new tags (releasing) and individual commits (publishing). As a result, the GPG signature becomes an integral part of the Git tag or Git commit. However, this approach supports only one signature.

The [signatures](https://github.com/werf/third-party-git-signatures) plugin allows you to sign Git tags and Git commits after they are created. In this case, GPG signatures are stored in [Git notes](https://git-scm.com/docs/git-notes). You can use as many signatures as you want, and you can also delete previously used signatures without affecting the linked Git tag or Git commit in any way.

All you need to do is set up GPG and Git correctly to create GPG signatures. This [manual](https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work#_gpg_introduction) can help you.

#### Installing the signatures plugin

To use the plugin, you have to install it to an arbitrary directory in `PATH` (e.g., `~/bin`):
```bash
git clone https://github.com/werf/third-party-git-signatures.git
cd third-party-git-signatures
install bin/git-signatures ~/bin
```

After running the `git signatures` command you should see the plugin description.

```bash
git signatures <command> [<args>]

Git Signatures is a system for adding and verifying one or more PGP
signatures to a given git reference.

Git Signatures works by appending one of more signatures of a given
ref hash to the git notes interface for that ref at 'refs/signatures'.

In addition to built in commit signing that allows -authors- to sign,
Git Signatures allows parties other than the author to issue "approval"
signatures to a ref, allowing for decentralized cryptographic proof of
code review. This is also useful for automation use cases where CI
systems to be able to add a signatures to a repo if a repo if all tests
pass successfully.

In practice Git Signatures allows for tamper evident design and brings
strong code attestations to a deployment process.

Commands
--------

* git signatures init
    Setup git to automatically include signatures on push/pull

* git signatures import
    Import all PGP keys specified in .gitsigners file to local
    GnuPG keychain allowing for verifications.

* git signatures show
    Show signatures for a given ref.

* git signatures add
    Add a signature to a given ref.

* git signatures verify
    Verify signatures for a given ref.

* git signatures pull
    Pull all signatures for all refs from origin.

* git signatures push
    Push all signatures for all refs to origin.

* git signatures version
    Report the version number.
```

### Configuring the build process

As a basic example of creating and arranging release artifacts for multiple platforms, let's deliver the script that outputs a release tag when run.

All build parameters, such as environment and build instructions, are defined in the [trdl.yaml](/reference/trdl_yaml.html) file.

**Caution.** Release artifacts must have a specific directory structure to deliver to different platforms and handle the executable files efficiently when using trdl-client ([learn more about using artifacts](/reference/trdl_yaml.html#release-artifacts-layout)).

#### trdl.yaml

{% include reference/trdl_yaml/example_trdl_yaml.md.liquid %}

#### build.sh

{% include reference/trdl_yaml/example_build_sh.md.liquid %}

Add both files and commit them to Git.

### Releasing a new version

Create and publish a new GPG-signed Git tag:

```shell
git tag -s v0.0.1 -m 'Signed v0.0.1 tag'
git push origin v0.0.1
```

> The tag defines the version of the release artifacts and has a predefined format: an arbitrary [semver](https://semver.org/lang/ru) number with the `v` prefix.

Once a Git tag is published, it needs to be signed by a sufficient number of trusted GPG keys. Each quorum member specified in the [plugin configuration](#configuring-the-plugin) must sign the Git tag and publish their GPG signature using the Git [signatures](#installing-the-signatures-plugin) plugin:

```shell
git fetch --tags
git signatures pull
git signatures add v0.0.1
git signatures push
```

> You can also use the following shortened command to sign Git tags: `git signatures add --push v0.0.1`.

Now that the tag has been created and signed by the necessary number of GPG keys, you can proceed to the release.

Use the [/release](/reference/vault_plugin/release.html#perform-a-release) API method to create a release. You can also use the following API methods for checking, controlling, and logging: [/task/:uuid](/reference/vault_plugin/task/uuid.html), [/task/:uuid/cancel](/reference/vault_plugin/task/uuid/cancel.html), and [/task/:uuid/log](/reference/vault_plugin/task/uuid/log.html).

A simplified version of the release process is available in the `release.sh` script in the [server/examples](https://github.com/werf/trdl/tree/main/server/examples) directory of the project repository.

Four environment variables must be set before running the script:
* `VAULT_ADDR` — Vault address;
* `VAULT_TOKEN` — Vault token with permissions to access the endpoint at which the plugin is registered;
* `PROJECT_NAME` — project name. In our case, this is the path at which the plugin is registered (see the `-path` parameter in the "Configuring the plugin" section);
* `GIT_TAG` — Git tag.

> Note that you can use our ready-made [set of Vault actions](https://github.com/werf/trdl-vault-actions) for GitHub Actions.

### Publishing the release channels

You must publish the release for the user to access it. To do this, switch to the main branch and add to the repository the [trdl_channels.yaml](/reference/trdl_channels_yaml.html) file that describes the release channels.

#### trdl_channels.yaml:

```yaml
groups:
- name: "0"
  channels:
  - name: alpha
    version: 0.0.1
  - name: stable
    version: 0.0.1
```

Next, add this file to Git, sign it with a GPG key, and commit it to the repository:

```shell
git add trdl_channels.yaml
git commit -S -m 'Signed release channels'
git push
```

Once a Git commit is published, it needs to be signed by a sufficient number of trusted GPG keys. Each quorum member specified in the [plugin configuration](#configuring-the-plugin) must sign the Git commit and publish their GPG signature using the Git [signatures](#installing-the-signatures-plugin) plugin:

```shell
git fetch
git signatures pull
git signatures add origin/main
git signatures push
```

> You can also use the following shortened command: `git signatures add --push origin/main`.

Now that the commit has the required number of GPG signatures, you can publish the release channels.

Use the [/publish](/reference/vault_plugin/publish.html) API method to do this. You can also use the following API methods for checking, controlling, and logging: [/task/:uuid](/reference/vault_plugin/task/uuid.html), [/task/:uuid/cancel](/reference/vault_plugin/task/uuid/cancel.html), and [/task/:uuid/log](/reference/vault_plugin/task/uuid/log.html).

A streamlined version of the publishing process is available in the `publish.sh` script in the [server/examples](https://github.com/werf/trdl/tree/main/server/examples) directory of the project repository.

As with `release.sh`, the `publish.sh` script requires setting some environment variables:
* `VAULT_ADDR` — Vault address;
* `VAULT_TOKEN` — Vault token with permissions to access the endpoint at which the plugin is registered;
* `PROJECT_NAME` — project name. In our case, this is the path at which the plugin is registered (see the `-path` parameter in the "Configuring the plugin" section).

> Note that you can use our ready-made set of [actions](https://github.com/werf/trdl-vault-actions) for GitHub Actions.

## For a user

> The instructions below are valid for Linux, macOS, and Windows. Commands can be executed in any Unix shell or in Windows PowerShell.

### Installing the client

Download the trdl client by following the instructions in the message for the [chosen release](https://github.com/werf/trdl/releases). Copy it to the directory in the user's `PATH`.

### Using the client

When adding the repository, the user has to provide details to verify the TUF repository during initial access: the TUF repository address (`URL`), the trusted version number (`ROOT_VERSION`), and the hash of the corresponding `<VERSION>.root.json` (`ROOT_SHA512`) file. The user receives these from the vendor.

In our case, the user gets the following data from the vendor:

```shell
URL=https://storage.googleapis.com/trdl-test-project-tuf
ROOT_VERSION=1
ROOT_SHA512=$(curl -Ls ${URL}/${ROOT_VERSION}.root.json | sha512sum)
```

Next, the user adds a repository with an arbitrary name:

```shell
REPO=test
trdl add $REPO $URL $ROOT_VERSION $ROOT_SHA512
```

You can then use artifacts within the desired update channel:

```shell
. $(trdl use test 0 stable)
```

The script is now available in the `PATH` of the current shell session.

<div class="tabs">
  <a href="javascript:void(0)" class="tabs__btn active" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'linux_or_darwin')">Linux / macOS</a>
  <a href="javascript:void(0)" class="tabs__btn" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'windows')">Windows</a>
</div>


<div id="linux_or_darwin" class="tabs__content active" markdown="1">

```shell
trdl-example.sh
v0.0.1
```
</div>

<div id="windows" class="tabs__content" markdown="1">

```shell
trdl-example.ps1
v0.0.1
```
</div>
