{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Update software

{{ header }} Syntax

```shell
trdl update REPO GROUP [CHANNEL] [options]
```

{{ header }} Options

```shell
      --background-stderr-file=''
            Redirect the stderr of the background update to a file
      --background-stdout-file=''
            Redirect the stdout of the background update to a file
      --in-background=false
            Perform update in background
      --no-self-update=false
            Do not perform self-update (default $TRDL_NO_SELF_UPDATE or false)
```

