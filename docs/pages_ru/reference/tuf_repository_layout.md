---
title: Организация TUF-репозитория
permalink: reference/tuf_repository_layout.html
---

Информацию про TUF-репозиторий, назначение и стандартный набор файлов, можно прочитать [в документации TUF](https://theupdateframework.github.io/specification/latest/#the-repository), а в этой статье будет уделено внимание организации [_Target files_](https://theupdateframework.github.io/specification/latest/#target-files), способу хранения релиза, GPG-подписей артефактов релиза, а также каналов обновлений. 

```
targets
├── channels/
├── releases/
└── signatures/
```

## Хранение релиза

### Хранение артефактов релиза

При релизе trdl использует путь, соответствующий версии релиза `targets/releases/<semver>/`, и сохраняет результат сборки без изменений.

```
targets
└── releases
    └── <semver>
        ├── ...
        └── <os>-<arch>
            ├── ...
            └── <release artifact>
```

**где:**

- semver — [semver](https://semver.org/lang) версия релиза;
- os — операционная система (`darwin`, `linux`, `windows` или `any`, если артефакты релиза не зависят от системы);
- arch — архитектура (`amd64`, `arm64` или `any`, если артефакты релиза не зависят от платформы);
- release artifact — произвольный файл. 

#### Пример

````
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
````

### Хранение GPG-подписей артефактов релиза

При релизе trdl подписывает все артефакты релиза (`targets/releases/<semver>/<os>-<arch>/<release artifact>`) и сохраняет все подписи в `targets/signatures/` по идентичному пути с расширением `.sig` (`targets/signatures/<semver>/<os>-<arch>/<release artifact>.sig`).

```
targets
└── signatures
    └── <semver>
        ├── ...
        └── <os>-<arch>
            ├── ...
            └── <release artifact>.sig
```

**где:**

- semver — [semver](https://semver.org/lang) версия релиза;
- os — операционная система (`darwin`, `linux`, `windows` или `any`, если артефакты релиза не зависят от системы);
- arch — архитектура (`amd64`, `arm64` или `any`, если артефакты релиза не зависят от платформы);
- release artifact — произвольный файл.

#### Пример

````
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
````

## Хранение каналов обновлений

При публикации trdl сохраняет каналы обновлений в соответствии с конфигурацией trdl_channels.yaml.

```
targets
└── channels
    ├── ...
    └── <semver part>
        ├── ...
        └── <channel>
```

**где:**

- semver part — произвольная часть [semver](https://semver.org/lang);
- channel — канал обновлений `alpha`, `beta`, `ea`, `stable` или `rock-solid`. 

### Пример

````
targets
└── channels
    ├── ...
    └── 1.2
        ├── alpha
        ├── beta
        ├── ea
        ├── stable
        └── rock-solid
````