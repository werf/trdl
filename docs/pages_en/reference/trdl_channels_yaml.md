---
title: trdl_channels.yaml
permalink: reference/trdl_channels_yaml.html
toc: false
---

The `trdl_channels.yaml` configuration contains groups, update channels and versions.

On publish, trdl reads `trdl_channels.yaml` from the default git repository branch, unless explicitly overridden when configuring the vault plugin, and then applies the changes - updates come to users.

{% include reference/trdl_channels_yaml/table.html %}

## Example

```yaml
groups:
  - name: 1.1
    channels:
      - name: alpha
        version: 1.1.25
      - name: beta
        version: 1.1.24
      - name: ea
        version: 1.1.23
      - name: stable
        version: 1.1.20
      - name: rock-solid
        version: 1.1.12
  - name: 1.2
    channels:
      - name: alpha
        version: 1.2.39
      - name: beta
        version: 1.2.38
      - name: ea
        version: 1.2.30
```