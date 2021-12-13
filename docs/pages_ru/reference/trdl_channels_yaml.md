---
title: trdl_channels.yaml
permalink: reference/trdl_channels_yaml.html
toc: false
---

Конфигурация `trdl_channels.yaml` содержит группы, каналы обновлений и версии.

При публикации trdl читает `trdl_channels.yaml` из ветки Git-репозитория по умолчанию, если это явно не переопределено при конфигурации vault-плагина, а затем применяет изменения. Обновления становятся доступными пользователям.

{% include reference/trdl_channels_yaml/table.html %}

## Пример

```yaml
groups:
  - name: 1.1
    channels:
      - name: alpha
        version: 1.1.25
      - name: beta
        version: 1.1.24
      - name: ea
        version: 1.1.23
      - name: stable
        version: 1.1.20
      - name: rock-solid
        version: 1.1.12
  - name: 1.2
    channels:
      - name: alpha
        version: 1.2.39
      - name: beta
        version: 1.2.38
      - name: ea
        version: 1.2.30
```
