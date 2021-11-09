{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Set default channel for a managed repository.
The new channel will be used by default instead of stable

{{ header }} Syntax

```shell
trdl set-default-channel REPO CHANNEL
```

