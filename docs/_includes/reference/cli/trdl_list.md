{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
List registered repositories

{{ header }} Syntax

```shell
trdl list
```

