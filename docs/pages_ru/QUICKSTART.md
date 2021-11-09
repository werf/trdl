---
title: Быстрый старт
permalink: quickstart.html
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

### Установка плагина

Скачайте [плагин trdl](https://github.com/werf/trdl/releases). Скопируйте его в `/etc/vault.d/plugins` или в другой каталог, где вы обычно храните плагины.

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

Теперь настроим сам плагин trdl. К подключенному плагину нужно обращаться по соответствующему пути, который мы указали в параметре `-path` при регистрации:

```shell
vault write trdl-test-project/configure s3_secret_access_key=FOO s3_access_key_id=BAR s3_bucket_name=trdl-test-project-tuf s3_region=europe-west1 s3_endpoint="https://storage.googleapis.com" required_number_of_verified_signatures_on_commit=0 git_repo_url=https://github.com/werf/trdl-test-project
```

Здесь мы указываем все необходимые настройки для работы плагина:
* `s3_secret_access_key` и `s3_access_key_id` — данные для доступа к бакету;
* `s3_bucket_name`, `s3_region` и `s3_endpoint` — имя бакета, его регион и адрес для подключения;
* `required_number_of_verified_signatures_on_commit` — количество необходимых подписей Git-коммита;
* `git_repo_url` — адрес репозитория.

## Для разработчика

### Создание сборочных инструкций

Рассмотрим простой пример: shell-скрипт, который выполняет `echo foobar`.

trdl.yaml:

{% raw %}
```shell
docker_image: alpine:3.13.6@sha256:e15947432b813e8ffa90165da919953e2ce850bef511a0ad1287d7cb86de84b5
commands:
- ./build.sh {{ .Tag }} && cp -a release-build/{{ .Tag }}/* /result
```
{% endraw %}

build.sh:

```shell
#!/bin/sh -e
VERSION=$1
if [ -z "$VERSION" ] ; then
    echo "Required version argument!" 1>&2
    echo 1>&2
    echo "Usage: $0 VERSION" 1>&2
    exit 1
fi
mkdir -p release-build/${VERSION}/any-any/bin && echo "echo foobar" > release-build/${VERSION}/any-any/bin/trdl-example.sh
```

Оба файла добавляем и коммитим в Git.

### Релиз новой версии

После того, как `trdl.yaml` и `build.sh` закоммичены в Git, создайте Git-тег, например `v0.0.1`. Тег будет определять версию артефакта.

Для создания релиза можно воспользоваться нашим готовым скриптом `release.sh`, который находится в каталоге [server/examples](https://github.com/werf/trdl/tree/main/server/examples).

Перед запуском скрипта необходимо установить четыре переменных окружения:
* `VAULT_ADDR` — адрес, по которому доступен Vault;
* `VAULT_TOKEN` — токен Vault с правами на обращение к endpoint’у, по которому зарегистрирован плагин;
* `PROJECT_NAME` — имя проекта. В нашем случае это путь, по которому зарегистрирован плагин (см. параметр -path в разделе «Настройка плагина»);
* `GIT_TAG` — git тег.

### Организация каналов обновлений

Чтобы сделать релиз доступным для клиента, релиз нужно опубликовать. Для этого переключитесь на основную ветку, добавьте в репозиторий файл с описанием каналов доставки и их соответствия релизам.

trdl_channels.yaml:

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

После добавления `trdl_channels.yaml` в репозиторий можно опубликовать релиз при помощи скрипта [publish.sh](https://github.com/werf/trdl/tree/main/server/examples).

Так же, как и скрипту `release.sh`, релизу необходимо передать несколько переменных окружения:
* `VAULT_ADDR` — адрес, по которому доступен Vault;
* `VAULT_TOKEN` — токен Vault с правами на обращение к endpoint’у, по которому зарегистрирован плагин;
* `PROJECT_NAME` — имя проекта. В нашем случае это путь, по которому зарегистрирован плагин (см. параметр `-path` в разделе «Настройка плагина»).

*Результатом работы скрипта будет…*

## Для клиента

### Установка клиента

Достаточно скачать готовый бинарный файл и положить его в каталог, доступный в `PATH` пользователя:

```shell
curl -L "https://tuf.trdl.dev/targets/releases/0.1.3/linux-amd64/bin/trdl" -o /tmp/trdl
mkdir -p ~/bin
install /tmp/trdl ~/bin/trdl
rm /tmp/trdl
echo 'export PATH=$HOME/bin:$PATH' >> ~/.bash_profile
export PATH="$HOME/bin:$PATH"
```

### Использование клиента

Чтобы подключить репозиторий к клиенту trdl, нужно получить хеш-сумму файла `1.root.json`, который находится в бакете:

```shell
curl -Ls http://127.0.0.1:9000/trdl-test-project-tuf/1.root.json | sha512sum
```

Далее (*этот пункт нужно доработать*):

```shell
trdl add test http://127.0.0.1:9000/trdl-test-project-tuf 10 bc74122561f18d2bad3fc7ae96cdd5673f1e0dd98bdb12d3717fe44d02f091b257d4c9b85deb591b0b342531d847a66edbe92824edbc93e9077d88cc45184d68
```

Здесь test — это имя репозитория, 0 — имя группы из trdl_channels.yaml.

Далее:

```shell
source $(trdl use test 0 stable)
```

Теперь наш скрипт `trdl-example.sh` доступен в `$PATH`:

```shell
$ trdl-example.sh
foobar
```
