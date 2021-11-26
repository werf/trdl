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

### trdl.yaml

{% include reference/trdl_yaml/example_trdl_yaml.md.liquid %}

### build.sh

{% include reference/trdl_yaml/example_build_sh.md.liquid %}