Update the software

## Syntax

```shell
trdl update REPO GROUP [CHANNEL] [options]
```

## Options

```shell
      --autoclean=true
            Erase old downloaded releases (default $TRDL_AUTOCLEAN or true)
      --background-stderr-file=''
            Redirect the stderr of the background update to a file (default $TRDL_BACKGROUND_STDERR_FILE or none)
      --background-stdout-file=''
            Redirect the stdout of the background update to a file (default $TRDL_BACKGROUND_STDOUT_FILE or none)
      --in-background=false
            Perform update in background (default $TRDL_IN_BACKGROUND or false)
      --no-self-update=false
            Do not perform self-update (default $TRDL_NO_SELF_UPDATE or false)
```

## Options inherited from parent commands

```shell
      --home-dir='~/.trdl'
            Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)
```

