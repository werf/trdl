<p align="center">
  <img src="https://trdl.dev/images/logo.svg" alt="trdl" style="max-height:100%;" height="30">
</p>
___

This repository allows you to organize CI/CD with GitHub Actions and
[trdl](https://trdl.dev/).

**Ready-to-use GitHub Actions Workflows** for different CI/CD workflows.

```yaml
- name: Install trdl
  uses: werf/trdl-actions/install@v0
```

## Examples

### Add and use `werf` manually via trdl CLI

```yaml
- name: Install trdl
  uses: werf/trdl-actions/install@v0

- name: Run werf
  run: |
    . $(trdl add werf https://tuf.werf.io 12 e1d3c7bcfdf473fe1466c5e9d9030bea0fed857d0563db1407754d2795256e4d063b099156807346cdcdc21d747326cc43f96fa2cacda5f1c67c8349fe09894d)
    . $(trdl use werf 2 stable)
    . $(werf ci-env github --as-file)
    werf converge
  env:
    WERF_KUBECONFIG_BASE64: ${{ secrets.KUBE_CONFIG_BASE64_DATA }}
    WERF_ENV: production
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Versioning

When using action, select the version corresponding to the required `MAJOR`
version of trdl.

By default, the action installs actual trdl version within _stable_ channel
(more details about channels, trdl release cycle and compatibility promise
[here](https://werf.io/installation.html#all-changes-in-werf-go-through-all-stability-channels)).
Using the `channel` input the user can switch the release channel.

> This is recommended approach to be up-to-date and to use actual trdl version
> without changing configurations.

```yaml
- uses: werf/trdl-actions/install@v0
  with:
    channel: stable
```

Withal, it is not necessary to work within release channels, and the user might
specify certain trdl version with `version` input.

```yaml
- uses: werf/trdl-actions/install@v0
  with:
    version: v2.1.0
```

## License

Apache License 2.0, see [LICENSE](LICENSE)
