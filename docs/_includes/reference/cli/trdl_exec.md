{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Exec a software binary

{{ header }} Syntax

```shell
trdl exec REPO GROUP [CHANNEL] [BINARY_NAME] [--] [ARGS]
```

