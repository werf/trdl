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

### Build backend

During a release, the plugin builds release artifacts with `docker buildx` in an ephemeral per-build builder. By default the `docker-container` driver is used, which requires access to the Docker daemon (see above).

On Kubernetes-native installations the build can instead be delegated to in-cluster BuildKit Pods using the buildx `kubernetes` driver. The driver is configured with environment variables of the Vault process:

* `TRDL_BUILDX_DRIVER` — the buildx driver to use: `docker-container` (default) or `kubernetes`;
* `TRDL_BUILDX_DRIVER_OPTS_<ANY_SUFFIX>` — additional `--driver-opt` values for `docker buildx create`. Any number of such variables can be set, one option each, applied in the lexicographic order of the variable names. The value is passed through as-is, so options containing commas, such as `nodeselector=disktype=ssd,zone=a`, need no escaping;
* `TRDL_BUILDX_DRIVER_OPTS_SEPARATOR` — when set, splits the value of each variable into several options by this separator. Unset by default, meaning one variable carries exactly one option.

An example configuration for the `kubernetes` driver:

```shell
TRDL_BUILDX_DRIVER=kubernetes
TRDL_BUILDX_DRIVER_OPTS_NAMESPACE=namespace=trdl-build
TRDL_BUILDX_DRIVER_OPTS_ROOTLESS=rootless=true
```

or the same configuration packed into a single variable with an explicit separator:

```shell
TRDL_BUILDX_DRIVER=kubernetes
TRDL_BUILDX_DRIVER_OPTS_SEPARATOR=;
TRDL_BUILDX_DRIVER_OPTS_KUBE='namespace=trdl-build;rootless=true'
```

The same two settings are also available per project in the plugin configuration, which is the only way to reach them when the plugin is compiled into a host process whose environment the administrator cannot set:

* `buildx_driver` — the same values as `TRDL_BUILDX_DRIVER`;
* `buildx_driver_opts` — a list of `--driver-opt` values, one per element, each passed through as is.

```json
{
  "buildx_driver": "kubernetes",
  "buildx_driver_opts": ["namespace=trdl-build", "rootless=true"]
}
```

Each of the two settings is resolved on its own: the plugin configuration takes precedence over the environment, and the environment takes precedence over the default `docker-container` driver with no options. A field left out of `configure`, or set to an empty value, means "not configured" and falls back to the environment — it does not override it with an empty value. To build with no driver options at all while the environment defines some, unset those variables.

Neither field can be combined with `buildkitd_address` or `buildkitd_driver` (see below): those settings replace the buildx path entirely, no builder is created through the docker CLI, and the driver settings would have no effect — so `configure` rejects the combination instead of ignoring them. The check covers one `configure` call only. A `TRDL_BUILDKITD_ADDRESS` set on the Vault process also wins over these settings, and since it is process-wide and can be changed after a project is configured, it cannot be rejected at that point: the build reports the settings as unused in the plugin log instead.

Notes on the `kubernetes` driver:

* the target namespace must exist, and the Vault process needs permissions to manage Deployments and Pods in it: the builder runs as a BuildKit Deployment and is removed after the build;
* the cluster is targeted via the standard kubeconfig or in-cluster ServiceAccount resolution;
* rootless BuildKit (`rootless=true`) does not fit the `baseline` PodSecurity level either: buildx gives the builder pod `seccompProfile: Unconfined` and the `unconfined` AppArmor annotation, and both are already forbidden at `baseline`. The builder namespace has to be labelled `privileged` (or be exempt from PodSecurity admission);
* see the [buildx kubernetes driver documentation](https://docs.docker.com/build/builders/drivers/kubernetes/) for the available driver options.

#### Building without the buildx drivers

Both buildx drivers above shell out to the `docker` CLI, so they require the binary to be present next to the plugin. Two settings replace that path, and both talk to BuildKit with the Go client:

* `buildkitd_driver` — the plugin provisions an ephemeral `buildkitd` itself, one per build, and removes it afterwards. It executes no external binary at all, which is what makes it usable where no `docker` exists — for example when the plugin is embedded into another process shipped in a distroless image;
* `buildkitd_address` — the plugin connects to a `buildkitd` somebody else runs, and provisions nothing. Whether it needs a binary depends on the scheme: `unix://` and `tcp://` do not, while `docker-container://` and `kube-pod://` still shell out to `docker` and `kubectl` respectively (see below).

Only one of them can be set, and `configure` refuses either of them next to the buildx *fields*. The buildx *environment* variables are a different matter: `TRDL_BUILDX_DRIVER` and `TRDL_BUILDX_DRIVER_OPTS_*` are a fallback for the fields, and a configured `buildkitd_driver` or `buildkitd_address` simply wins over them, because a process-wide variable cannot be rejected when a project is configured.

##### Provisioning an ephemeral buildkitd

`buildkitd_driver=kubernetes` runs the builder as a Pod for the duration of one build:

```shell
vault write trdl-test-project/configure ... \
  buildkitd_driver=kubernetes \
  buildkitd_driver_opts=namespace=trdl-build \
  buildkitd_driver_opts=serviceaccount=trdl-buildkit
```

or, as a JSON payload:

```json
{
  "buildkitd_driver": "kubernetes",
  "buildkitd_driver_opts": ["namespace=trdl-build", "serviceaccount=trdl-buildkit"]
}
```

The options are `name=value` pairs, one per list element and passed through as is, so a value containing commas such as `nodeselector=disktype=ssd,zone=a` needs no escaping. They are validated when the configuration is written — an option the driver cannot honor is rejected there rather than by a release that fails later. The kubernetes driver accepts:

| Option | Meaning |
|---|---|
| `namespace` | the namespace to run the builder in; defaults to the namespace named by the current kubeconfig context, then to the namespace of the plugin's own ServiceAccount, and to `default` when neither names one |
| `image` | the buildkitd image; defaults to `moby/buildkit:buildx-stable-1`, or its `-rootless` variant when `rootless=true` |
| `rootless` | run rootless BuildKit |
| `serviceaccount` | the ServiceAccount for the builder pod |
| `nodeselector`, `labels`, `annotations` | comma-separated `name=value` pairs |
| `requests.cpu`, `requests.memory`, `requests.ephemeral-storage` | pod resource requests |
| `limits.cpu`, `limits.memory`, `limits.ephemeral-storage` | pod resource limits |
| `timeout` | how long to wait for the builder to become ready, e.g. `5m`; `2m` by default |
| `deadline` | a hard lifetime cap for the builder pod (`activeDeadlineSeconds`), e.g. `2h`; defaults to the release task's timeout ([`task_timeout`](/reference/vault_plugin/task/configure.html), `30m` unless configured) plus a five-minute margin; a whole number of seconds of at least `1s`. It is not a grace period: a build still running when it expires is killed too, so set it above the longest release this project takes |

The option names are the buildx kubernetes driver's own wherever the two overlap, but the vocabulary is this driver's, not buildx's: options buildx accepts and this driver does not — `replicas`, `loadbalance`, `tolerations`, `schedulername`, `qemu.*`, the persistent-volume options — are rejected, and `deadline` has no buildx counterpart.

What the plugin needs in the target namespace is `create`, `get` and `delete` on `pods`, plus `create` on `pods/exec`; a namespaced Role is enough. No `apps` API group is used. The build stream rides the API server's exec channel, so the plugin needs no network route to the builder pod. The cluster is targeted via the standard kubeconfig or in-cluster ServiceAccount resolution.

Notes on the pod:

* the namespace must exist and must admit the builder pod. By default the container runs `privileged`; with `rootless=true` it runs unprivileged but needs seccomp `Unconfined` and the `unconfined` AppArmor annotation instead. Either way the `baseline` PodSecurity level forbids it, so the namespace has to be labelled `privileged` or be exempt from PodSecurity admission — the same requirement the buildx `kubernetes` driver has. Unlike the buildx path, the rejection arrives directly from the `create` call rather than as a readiness timeout;
* the pod is removed when the build ends, including when it fails or is canceled, and when the builder never becomes ready. It is not removed if the plugin's own process dies outright, and not if the delete itself fails — a lost API connection, a withdrawn `delete` permission. That failure is reported in the release log and the plugin log, but it does not fail the release, so a privileged pod can outlive a build that reported success. Both cases are bounded by `activeDeadlineSeconds`, which is always set — `deadline` overrides the default. Note it terminates the pod but does not delete the object: a Failed pod remains visible until removed by hand or by the cluster's pod garbage collection. It is also counted from the moment the pod starts running, so a pod that was never scheduled — no node, a quota rejection — is not capped by it;
* the builder is a bare Pod with `restartPolicy: Never`, deliberately: nothing may replace it mid-build, because the replacement would be a builder the release is not connected to.

Two things follow from the plugin creating the pod with its own credentials, and both are the administrator's to weigh:

* **`configure` write access becomes pod-create access.** Whoever can write a project's configuration chooses the `namespace` the builder runs in, the `serviceaccount` it runs as and the `image` it runs — anywhere the plugin's own Role reaches. Restrict `<project>/configure` to the same people who are trusted with the release keys, and keep the plugin's Role to namespaces that hold nothing else worth taking. Give the builder a ServiceAccount whose privileges you have audited in full: creating a workload lets it run as any ServiceAccount of that namespace, and a ServiceAccount carries whatever is bound to it directly, through a namespaced RoleBinding, through a ClusterRoleBinding, and through group bindings such as `system:serviceaccounts`. The absence of a RoleBinding in the namespace is not by itself an isolation guarantee, and when a `serviceaccount` is configured the builder pod does mount its token; without one the token is not mounted at all (`automountServiceAccountToken: false`).
* **The builder container is privileged by default.** BuildKit needs it; `rootless=true` trades it for seccomp `Unconfined` and the `unconfined` AppArmor annotation. Either way the build executes project-supplied instructions in a container the `baseline` PodSecurity level would refuse, so give it a namespace and nodes you are willing to lose, not the ones the signing keys live on.

##### Connecting to an existing buildkitd

The build can instead be pointed at an already running `buildkitd`: the plugin then talks to it directly with the BuildKit client, and no builder is provisioned or removed per build.

The buildkitd address is set per project in the plugin configuration:

```shell
vault write trdl-test-project/configure ... buildkitd_address=tcp://buildkitd.trdl-build.svc:1234
```

or, as a fallback for all projects, with the `TRDL_BUILDKITD_ADDRESS` environment variable of the Vault process. The per-project setting takes precedence. The supported address schemes are:

* `unix://` and `tcp://` — direct gRPC connection to buildkitd, no external binaries required;
* `docker-container://` and `kube-pod://` — connection through `docker exec`/`kubectl exec`, requiring the corresponding CLI.

When neither `buildkitd_address` nor `buildkitd_driver` is set, builds go through `docker buildx` exactly as described above. Deploying `buildkitd` itself is out of trdl's scope: on Kubernetes it is typically a Deployment or StatefulSet in a dedicated namespace whose PodSecurity labels are managed by the cluster owner, since BuildKit requires a relaxed seccomp/AppArmor profile even in rootless mode.

Securing the connection and isolating the daemon is the administrator's responsibility. The plugin sends the entire build context and every build secret over this connection — the project build secrets and, when mac signing is configured, the signing certificate, its password and the notary key. What that requires:

* **`tcp://` is plaintext and unauthenticated.** The plugin neither encrypts the traffic nor verifies the identity of the daemon it connects to, so anyone able to intercept the connection or take over the endpoint's address receives those secrets. Use `tcp://` only over a channel made confidential and authenticated by other means — a network segment no other workload can reach, a service mesh with mTLS, or an equivalent tunnel. When that cannot be guaranteed, run buildkitd alongside the plugin and use `unix://` to a socket shared between them.
* **The address is a trust boundary.** Whoever can write the project configuration, or set `TRDL_BUILDKITD_ADDRESS` for the Vault process, decides which daemon receives the release secrets. Write access to `<project>/configure` has to be restricted to the same people who are trusted with the release keys.
* **The daemon is shared and unrestricted.** buildkitd executes the project's build instructions, and no builder is created or removed per build, so concurrent releases and every project pointed at the same address share one instance, its cache and its privileges. Dedicate an instance per trust domain, and treat access to it as access to the release artifacts it produces.

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

**Adding a key**

```shell
vault write trdl-test-project/configure/trusted_pgp_public_key name=developer public_key=@developer.pgp
```

where `developer.pgp` is the file with the public PGP key generated by the `gpg --armor --output developer.pgp --export developer@trdl.dev` command.

{% offtopic title="Contents of the developer.pgp file" %}
```shell
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
XaOlJrPDM5E9zw==
=bIYD
-----END PGP PUBLIC KEY BLOCK-----
```
{% endofftopic %}

Please refer to the [gpg documentation](https://www.gnupg.org/gph/en/manual.html#AEN65) for more information about exporting.

**Getting a list of keys**

```shell
vault read trdl-test-project/configure/trusted_pgp_public_key
````

```shell
Key     Value
---     -----
keys    [developer]
```

**Displaying key contents**

```shell
vault read trdl-test-project/configure/trusted_pgp_public_key/developer
```

{% offtopic title="Command output" %}
```shell
Key           Value
---           -----
name          developer

public_key    -----BEGIN PGP PUBLIC KEY BLOCK-----

mQGNBGH8PiQBDAClie5jZHKIEDUw14+UJB+knS+X5SQg8lOlZqdiizMcYBdhnEEM
OLhtvvMfTTY+ikREuvEVUBVXYMrAGSCA+291ngbKIlU5YyC75mHxV6IDvEX91UEc
5o2OXnNFlTHj3jXAJytUd6IXfv6Wx06aHI8xeFzhYxW8CHD/NaJd+XfX3gr5pmUp
U2N8T0dTIM9QZ4o8fdrpWfMcp6Q8LwO1ConFJnEPIvR0etdqNiIu+6/33ImWrYuu
09XHUQ+LZAkjP9YJS8ITK38qboEtFsflO06NMeaPH+TgLFmBi4Ov42aSJCJ5x1HS
5qB18V99oEVFE82DVjy7Eflw4oCJayue405X1mgW0uc/225n+9JwoV2ZyRG5s/aE
gQjxqaVIDr7a6RtfqRK8AAPHkSOhaP2l0PhO9voZ/y2sFqtuWq8y+I+O78Gxq85O
ejuf0U/KYcQKjg4CE1eAVxakBz24VWkSHuBvdhjvzQydSe0KEKV/uE4g5ihk8olD
tf+cAf2jFLrlBDEAEQEAAbQZVGVhbSBMZWFkZXIgPHRsQHRyZGwuZGV2PokBzgQT
AQoAOBYhBCulX9gVgDTuvpKqntnXm2Ovwwx6BQJh/D4kAhsDBQsJCAcCBhUKCQgL
AgQWAgMBAh4BAheAAAoJENnXm2Ovwwx6Ng0L+wWkj/P5QINyids8iLoNnYGdKx46
ayLzi7HquOC2ckQiazcli5KSq9/4uJn9ff2Ri4wQmwNMOuLBUSFxyfibR73ZAFtS
xHfbYFgUoQHWOH//y5QzEkHSNZXFhsSKuy3Xgmr7o3BtVtmR33qYUpbVrRVCYIdN
qKVlpBxQnObq995993eIUUKTheUfFF9Bh91mdbU4usZf1uQH0I5vhTS7Xd45U9Wd
m2g7NoMQVgM8lAmwaDWlKzv+P4XiQFUUbSGbXt7yQtqXUhVXOQ5xLh/i0mDVrSlt
tZD+F6tFYgEJphlgWkEFXpcWI9xxpGv6UCuCnhm5B9SbV83pJUp1Dr/Btw/OASUW
PzcvN54LwXX2SwTP83qxS2qpvHK4SNtHrn7+icgBi2ZLqCv+8iWNPvl3G9pF/Zzs
E8bQh0lmdvHIoJd2ZeBKfBOOMLqHEPae9DYcaW9VUkLr+GRFHJzh1WHF9f1Dd+A+
INJqsb1KawfsJwDXcZM8si1PUhoxI+YbFXgn8rkBjQRh/D4kAQwAqudoseQ/O6WU
NdE9XSCvJAhnUYKhLadTyN8pd70ibWONav4M+B71rg+BFNTTB5eRHEgGzPDJmxex
ba4Zhvt+2TAbmnF1SAcSciCEIx57239L1ERkLXpHwNLmCEjbiR3k9xOZ4wMDQEHC
1qswbf0XvO1UjYsw/L6uL253anqP8IxMSPuCG9TkZuZ4A1qrCxQ2Y8JO+XEtM764
5OqWGU90I+6PXl0hgPgg+VeFpkAXr67fwaa94aISJq/rIzfxf76N8YcJeldMlFyp
vytz7BqsdGYmVigKSjWCllIVTCyFV3oggnDJn6Gmbwhp8+lj9MuZRyBn3nFzZbZT
Mo9TAgIFy6UQ80yW2M9MnIOPMHmtRzoSjUlEgTzjwT8L/YQGQ9GnFxIINk4PUFPj
fFEvmP+y+8cb+EhrgQ770LtQEd+E6zXexrh9mvGxIj87XP5Jl6Kz8goMcPp3+jTR
vggepxU/6pmFonRMcbmwjZ1M9JpibjPX49Pb1nAkUvE6szgwMItPABEBAAGJAbYE
GAEKACAWIQQrpV/YFYA07r6Sqp7Z15tjr8MMegUCYfw+JAIbDAAKCRDZ15tjr8MM
eiJ5DACCga9PnpyVHIltDXb5UC3OEsfNLI8PCVnBnMMco2Iedea0E3pyKniMHxHS
TW/+4RT9KzdOqOEQBzIdmsL/Vq0dnh3j+UDrVhp6ppVi5dBXgrgYx1RL+4EoipOS
pVKJdmOqA/b5O8LNnN761MP3n5gJWURr5k2seKhxgjTQ27qRPi3Gq6mtj0xWRkXZ
ivia1mefDpIif0TjSCrEMy4y8Zj+4fyy6AbMGYvSkUDaCwzk0shiAwAhW+9w8V6f
2fDuY18OXvTNwW8anU7XMM12mdyNdzvVPTfe23HdboJ5dDwKH8p8E+f1B+ozXosb
qvdhnCCdTNCww95K+Nq5zy0CQ2+mGB929dmOIJCo7BTM4/vxQT100P/FnShCu/Ji
UVlFWU0M1u8czX5la8AXimkAdmO9HIiPD6Qs/X+VaLuqvgIO0OrmytC1jVXzn9HH
1GYIrC8WdSo7ATE/gI5BftJq+WXDzXwLCA1Ze2QP8GffQKkuRjHRiv3spnFAXjiZ
TJ9EZRY=
=vkcq
-----END PGP PUBLIC KEY BLOCK-----
```
{% endofftopic %}

**Deleting a key**

```shell
vault delete trdl-test-project/configure/trusted_pgp_public_key/developer 
````

```
Success! Data deleted (if it existed) at: trdl-test-project/configure/trusted_pgp_public_key/developer
```

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
ROOT_SHA512=$(curl -Ls ${URL}/${ROOT_VERSION}.root.json | sha512sum | cut -c 1-128)
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
