{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Generate script to update software binaries in the background and use local ones within a shell session

{{ header }} Syntax

```shell
trdl use REPO GROUP [CHANNEL] [options]
```

{{ header }} Examples

```shell
  # Source script in a shell
  $ . $(trdl use repo_name 1.2 ea)

  # Force script generation for a Unix shell on Windows
  $ trdl use repo_name 1.2 ea --shell unix

```

{{ header }} Options

```shell
      --no-self-update=false
            Do not perform self-update (default $TRDL_NO_SELF_UPDATE or false)
      --shell='pwsh'
            Select the shell for which to prepare the script. 
            Supports `pwsh` and `unix` shells (default $TRDL_SHELL, `pwsh` for Windows or `unix`)
```

