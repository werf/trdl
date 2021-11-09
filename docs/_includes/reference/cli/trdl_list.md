{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
List managed repositories

{{ header }} Syntax

```shell
trdl list
```

