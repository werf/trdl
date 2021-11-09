{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Add a software repository

{{ header }} Syntax

```shell
trdl add REPO URL ROOT_VERSION ROOT_SHA512
```

