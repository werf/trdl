---
title: Быстрый старт
permalink: quickstart.html
layout: page-nosidebar
toc: true
toc_headers: h2
---

## Для администратора

> Мы предполагаем, что вы уже знакомы с Vault и умеете его использовать, поэтому опустим подробности его настройки. Будем считать, что Vault настроен согласно [официальной документации](https://learn.hashicorp.com/tutorials/vault/deployment-guide).

### Vault

Установить Vault и trdl-плагин можно несколькими способами. Рассмотрим самый простой: использование уже готового бинарника Vault (например, скачанного с официального сайта или установленного пакетным менеджером дистрибутива) и готового бинарного файла trdl-плагина.

### Docker
Установите Docker. Добавьте в группу Docker пользователя, из-под которого запускается Vault:

```shell
usermod -a -G docker vault
```

### Подготовка проекта

#### Git-репозиторий

Создайте обычный публичный Git-репозиторий.

#### Бакет

Подойдет любой S3-совместимый бакет. Он должен быть публично доступен для чтения.

{% offtopic title="Особенности GCS (Google Cloud Storage)" %}
При появлении ошибки `An error occurred (AccessDenied) when calling the CreateMultipartUpload operation: Access denied` необходимо убедиться, что у Service Account'а, который используется для доступа к бакету, имеется роль `Storage Admin`
{% endofftopic %}

### Установка плагина

Скачайте плагин trdl, следуя инструкциям в сообщении [выбранного релиза](https://github.com/werf/trdl/releases). Скопируйте его в `/etc/vault.d/plugins` или в другой каталог, где вы обычно храните плагины.

### Настройка плагина

Для настройки со стороны Vault нужно указать каталог, в котором хранятся плагины:

```shell
plugin_directory = "/etc/vault.d/plugins"
```

Перезапустите Vault.

Зарегистрируйте плагин в Vault:

```shell
vault plugin register -sha256=$(sha256sum /etc/vault.d/plugins/vault-plugin-secrets-trdl | awk '{print $1}') secret vault-plugin-secrets-trdl
```

В нашем случае файл плагина называется `vault-plugin-secrets-trdl`, и в Vault мы его регистрируем с таким же именем. Подробно про регистрацию плагинов можно прочитать в [официальной документации](https://www.vaultproject.io/docs/commands/plugin/register).

Подключите плагин как `secrets engine` с определенным путем:

```shell
vault secrets enable -path=trdl-test-project vault-plugin-secrets-trdl
```

Один и тот же плагин можно подключать множество раз, но каждый раз с уникальным путем. Подробнее об этом — в [официальной документации](https://www.vaultproject.io/docs/commands/secrets/enable).

Теперь настроим сам плагин trdl. Для конфигурации необходимо использовать метод API [/configure](/reference/vault_plugin/configure.html#configure-plugin):

```shell
vault write trdl-test-project/configure @configuration.json
```

где `configuration.json`:
```json
{
  "s3_secret_access_key": "FOO",
  "s3_access_key_id": "BAR",
  "s3_bucket_name": "trdl-test-project-tuf",
  "s3_region": "europe-west1",
  "s3_endpoint": "https://storage.googleapis.com",
  "git_repo_url": "https://github.com/werf/trdl-test-project",
  "required_number_of_verified_signatures_on_commit": 0
}
```

> В текущей версии инструкции организация кворума GPG-подписей и сопутствующий процесс не рассматривается, при конфигурации плагина явно игнорируется (`"required_number_of_verified_signatures_on_commit": 0`).
> 
> При конфигурации плагина **крайне важно** указывать минимальное количество требуемых GPG-подписей у коммита (`required_number_of_verified_signatures_on_commit`) и загружать допустимый набор публичных GPG-ключей, используя метод API [/configure/trusted_pgp_public_key](/reference/vault_plugin/configure/trusted_pgp_public_key.html). В противном случае система обновлений становится уязвимой, так как выполнение операций контролируется не кворумом, определённым при конфигурации плагина, а любым пользователем получившим доступ.

## Для разработчика

### Конфигурация сборки

Рассмотрим простой пример создания и организации артефактов релиза для нескольких платформ — организуем доставку скрипта, который при запуске будет выводить тег релиза.

Вся конфигурация сборки, окружение и сборочные инструкции, описываются в файле [trdl.yaml](/reference/trdl_yaml.html).

Стоит отдельно отметить, что артефакты релиза должны иметь определённую организацию директорий для доставки на различные платформы и эффективной работы с исполняемыми файлами при использовании trdl-клиента пользователями (подробнее в [соответствующем разделе документации](/reference/trdl_yaml.html#организация-артефактов-релиза)).

#### trdl.yaml

{% include reference/trdl_yaml/example_trdl_yaml.md.liquid %}

#### build.sh

{% include reference/trdl_yaml/example_build_sh.md.liquid %}

Оба файла добавляем и коммитим в Git.

### Релиз новой версии

После того как `trdl.yaml` и `build.sh` закоммичены в Git, создайте Git-тег с произвольным [semver](https://semver.org/lang/ru) и префиксом `v` (например, `v0.0.1`). Тег будет определять версию артефактов релиза.

Для создания релиза необходимо использовать метод API [/release](/reference/vault_plugin/release.html#perform-a-release), а проверка, контроль и логирование может организовываться с помощью методов API [/task/:uuid](/reference/vault_plugin/task/uuid.html), [/task/:uuid/cancel](/reference/vault_plugin/task/uuid/cancel.html) и [/task/:uuid/log](/reference/vault_plugin/task/uuid/log.html).

Упрощённая версия релизного процесса представлена в скрипте `release.sh`, который находится в каталоге [server/examples](https://github.com/werf/trdl/tree/main/server/examples) репозитория проекта.

Перед запуском скрипта необходимо установить четыре переменных окружения:
* `VAULT_ADDR` — адрес, по которому доступен Vault;
* `VAULT_TOKEN` — токен Vault с правами на обращение к endpoint’у, по которому зарегистрирован плагин;
* `PROJECT_NAME` — имя проекта. В нашем случае это путь, по которому зарегистрирован плагин (см. параметр -path в разделе «Настройка плагина»);
* `GIT_TAG` — git тег.

> При использовании GitHub Actions можно воспользоваться [нашим готовым набором actions](https://github.com/werf/trdl-vault-actions).

### Организация каналов обновлений

Чтобы сделать релиз доступным для пользователя, релиз нужно опубликовать. Для этого переключитесь на основную ветку, добавьте в репозиторий файл с описанием каналов обновлений [trdl_channels.yaml](/reference/trdl_channels_yaml.html).

#### trdl_channels.yaml:

```yaml
groups:
- name: "0"
  channels:
  - name: alpha
    version: 0.0.1
  - name: stable
    version: 0.0.1
```

### Публикация каналов обновлений

После того как `trdl_channels.yaml` находится в Git, можно переходить непосредственно к публикации.

При публикации необходимо использовать метод API [/publish](/reference/vault_plugin/publish.html), а проверка, контроль и логирование может организовываться с помощью методов API [/task/:uuid](/reference/vault_plugin/task/uuid.html), [/task/:uuid/cancel](/reference/vault_plugin/task/uuid/cancel.html) и [/task/:uuid/log](/reference/vault_plugin/task/uuid/log.html).

Упрощённая версия процесса публикации представлена в скрипте `publish.sh`, который находится в каталоге [server/examples](https://github.com/werf/trdl/tree/main/server/examples) репозитория проекта.

Так же, как и скрипту `release.sh`, скрипту `publish.sh` требуются переменные окружения:
* `VAULT_ADDR` — адрес, по которому доступен Vault;
* `VAULT_TOKEN` — токен Vault с правами на обращение к endpoint’у, по которому зарегистрирован плагин;
* `PROJECT_NAME` — имя проекта. В нашем случае это путь, по которому зарегистрирован плагин (см. параметр `-path` в разделе «Настройка плагина»).

> При использовании GitHub Actions можно воспользоваться [нашим готовым набором actions](https://github.com/werf/trdl-vault-actions).

## Для пользователя

> Инструкция приведённая далее справедлива для операционных систем Linux, macOS и Windows. Команды можно выполнять в произвольной командной оболочке Unix или в PowerShell для Windows.

### Установка клиента

Скачайте trdl-клиент, следуя инструкциям в сообщении [выбранного релиза](https://github.com/werf/trdl/releases). Положите его в каталог, доступный в `PATH` пользователя.

### Использование клиента

При добавлении репозитория пользователю потребуются: произвольное имя (`REPO`), адрес TUF-репозитория (`URL`), номер доверенной версии (`ROOT_VERSION`) и хеш-сумма соответствующего файла `<VERSION>.root.json` (`ROOT_SHA512`), которые **вендор должен предоставить пользователю** для верификации TUF-репозитория при первичном обращении.

Таким образом, пользователь получает от вендора следующие данные:

```shell
URL=https://storage.googleapis.com/trdl-test-project-tuf
ROOT_VERSION=1
ROOT_SHA512=$(curl -Ls ${URL}/${ROOT_VERSION}.root.json | sha512sum)
```

И добавляет репозиторий, указав произвольное имя:

```shell
REPO=test
trdl add $REPO $URL $ROOT_VERSION $ROOT_SHA512
```

После того как репозиторий добавлен можно начинать использовать артефакты в рамках желаемого канала обновления:

```shell
. $(trdl use test 0 stable)
```

Теперь скрипт доступен в `PATH` текущей shell-сессии.

<div class="tabs">
  <a href="javascript:void(0)" class="tabs__btn active" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'linux_or_darwin')">Linux / macOS</a>
  <a href="javascript:void(0)" class="tabs__btn" onclick="openTab(event, 'tabs__btn', 'tabs__content', 'windows')">Windows</a>
</div>


<div id="linux_or_darwin" class="tabs__content active" markdown="1">

```shell
$ trdl-example.sh
v0.0.1
```
</div>

<div id="windows" class="tabs__content" markdown="1">

```shell
$ trdl-example.ps1
v0.0.1
```
</div>
