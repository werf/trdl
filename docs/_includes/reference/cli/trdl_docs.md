{% if include.header %}
{% assign header = include.header %}
{% else %}
{% assign header = "###" %}
{% endif %}
Generate documentation as markdown

{{ header }} Syntax

```shell
trdl docs JEKYLL_SITE_DIR [options]
```

{{ header }} Options

```shell
  -h, --help=false
            help for docs
```

