Generate a script to update the software binaries in the background and use local ones within a shell session

## Syntax

```shell
trdl use REPO GROUP [CHANNEL] [options]
```

## Examples

```shell
  # Source script in a shell
  $ . $(trdl use repo_name 1.2 ea)

  # Force script generation for a Unix shell on Windows
  $ trdl use repo_name 1.2 ea --shell unix

```

## Options

```shell
      --no-self-update=false
            Do not perform self-update (default $TRDL_NO_SELF_UPDATE or false)
      --shell='unix'
            Select the shell for which to prepare the script. 
            Supports `pwsh` and `unix` shells (default $TRDL_SHELL, `pwsh` for Windows or `unix`)
```

## Options inherited from parent commands

```shell
      --home-dir='~/.trdl'
            Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)
```

