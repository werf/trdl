{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Get directory with software artifacts

{{ header }} Syntax

```shell
trdl dir-path REPO GROUP [CHANNEL]
```

