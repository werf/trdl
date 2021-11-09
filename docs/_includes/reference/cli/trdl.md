{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
The universal package manager for delivering your software updates securely from a TUF repository (more details on [https://trdl.dev](https://trdl.dev))

{{ header }} Options

```shell
      --home-dir='~/.trdl'
            Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)
```

