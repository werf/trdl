---
title: trdl.yaml
permalink: reference/trdl_yaml.html
toc: false
---

Конфигурация `trdl.yaml` содержит инструкции, которые определяют окружение и набор команд необходимый для сборки артефактов релиза. 

При релизе trdl читает `trdl.yaml` из git-тега и выполняет сборку:
- Запускает контейнер на базе выбранного docker-образа.
- Монтирует исходный код git-тега в директорию `/git`.
- Выполняет сборочные инструкции в директории `/git`.
- Сохраняет артефакты релиза из директории `/result`.

<br />

{% include reference/trdl_yaml/table.html %}

## Пример

{% raw %}
```yaml
dockerImage: golang:1.17-alpine@sha256:13919fb9091f6667cb375d5fdf016ecd6d3a5d5995603000d422b04583de4ef9
commands:
  - ./scripts/build.sh {{ .Tag }} 
  - cp -a release/* /result
```
{% endraw %}