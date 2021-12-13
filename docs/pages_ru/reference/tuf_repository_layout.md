---
title: Организация TUF-репозитория
title: Организация TUF-репозитория
permalink: reference/tuf_repository_layout.html
toc: true
---

Информацию про TUF-репозиторий, его назначение и стандартный набор файлов можно найти [в документации TUF](https://theupdateframework.github.io/specification/latest/#the-repository). Эта статья — про организацию [target files](https://theupdateframework.github.io/specification/latest/#target-files), способ хранения релиза, GPG-подписей артефактов релиза, а также каналов обновлений. В нашем случае target files — это релизы, подписи и каналы обновлений.

```
targets
├── channels/
├── releases/
└── signatures/
```

## Хранение релиза

### Хранение артефактов релиза

При релизе trdl использует путь, который соответствует версии релиза `targets/releases/<semver>/`, и сохраняет результат сборки без изменений.

```
targets
└── releases
    └── <semver>
        ├── ...
        └── <os>-<arch>
            ├── ...
            └── <release artifact>
```

Здесь:

- `semver` — [semver](https://semver.org/lang/ru) версия релиза;
- `os` — операционная система (`darwin`, `linux`, `windows` или `any`, если артефакты релиза не зависят от системы);
- `arch` — архитектура (`amd64`, `arm64` или `any`, если артефакты релиза не зависят от платформы);
- `release artifact` — произвольный файл.

#### Пример

```
targets
└── releases
    ├── ...
    └── 1.2.20
        ├── darwin-amd64
        │   └── bin
        │       └── werf
        ├── darwin-arm64
        │   └── bin
        │       └── werf
        ├── linux-amd64
        │   └── bin
        │       └── werf
        ├── linux-amd64
        │   └── bin
        │       └── werf
        └── windows-amd64
            └── bin
                └── werf.exe
```

### Хранение GPG-подписей артефактов релиза

При релизе trdl:
* подписывает все артефакты релиза: `targets/releases/<semver>/<os>-<arch>/<release artifact>`;
* сохраняет все подписи в `targets/signatures/` по идентичному пути с расширением `.sig`: `targets/signatures/<semver>/<os>-<arch>/<release artifact>.sig`.

```
targets
└── signatures
    └── <semver>
        ├── ...
        └── <os>-<arch>
            ├── ...
            └── <release artifact>.sig
```

Здесь:

- `semver` — [semver-версия](https://semver.org/lang/ru) релиза;
- `os` — операционная система (`darwin`, `linux`, `windows` или `any`, если артефакты релиза не зависят от системы);
- `arch` — архитектура (`amd64`, `arm64` или `any`, если артефакты релиза не зависят от платформы);
- `release artifact` — произвольный файл.

#### Пример

```
targets
└── signatures
    ├── ...
    └── 1.2.20
        ├── darwin-amd64
        │   └── bin
        │       └── werf.sig
        ├── darwin-arm64
        │   └── bin
        │       └── werf.sig
        ├── linux-amd64
        │   └── bin
        │       └── werf.sig
        ├── linux-amd64
        │   └── bin
        │       └── werf.sig
        └── windows-amd64
            └── bin
                └── werf.exe.sig
```

## Хранение каналов обновлений

При публикации trdl сохраняет каналы обновлений в соответствии с конфигурацией `trdl_channels.yaml`.

```
targets
└── channels
    ├── ...
    └── <semver part>
        ├── ...
        └── <channel>
```

Здесь:

- `semver part` — произвольная часть [semver](https://semver.org/lang/ru);
- `channel` — канал обновлений `alpha`, `beta`, `ea`, `stable` или `rock-solid`.

### Пример

```
targets
└── channels
    ├── ...
    └── 1.2
        ├── alpha
        ├── beta
        ├── ea
        ├── stable
        └── rock-solid
```
