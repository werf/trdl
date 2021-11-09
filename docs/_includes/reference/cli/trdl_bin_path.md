{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Get directory with software binaries

{{ header }} Syntax

```shell
trdl bin-path REPO GROUP [CHANNEL]
```

