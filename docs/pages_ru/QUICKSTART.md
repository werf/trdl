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
При появлении ошибки `An error occurred (AccessDenied) when calling the CreateMultipartUpload operation: Access denied`  убедитесь, что у Service Account'а, который используется для доступа к бакету, роль `Storage Admin`
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

В нашем случае файл плагина называется `vault-plugin-secrets-trdl`, и в Vault мы его регистрируем с таким же именем. Подробно про регистрацию плагинов можно прочитать [в официальной документации](https://www.vaultproject.io/docs/commands/plugin/register).

Подключите плагин как `secrets engine` с определенным путем:

```shell
vault secrets enable -path=trdl-test-project vault-plugin-secrets-trdl
```

Один и тот же плагин можно подключать множество раз, но каждый раз с уникальным путем. Подробнее об этом — [в официальной документации](https://www.vaultproject.io/docs/commands/secrets/enable).

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
  "required_number_of_verified_signatures_on_commit": 2
}
```

> При настройке плагина важно указывать минимальное количество требуемых GPG-подписей коммита (`required_number_of_verified_signatures_on_commit`). В противном случае система обновлений становится уязвимой, так как операции контролируются не кворумом, определённым при настройке плагина, а любым пользователем с доступом.

Минимальное количество требуемых GPG-подписей (`required_number_of_verified_signatures_on_commit`) зависит от размера и особенности команды, частоты операций и других факторов.

#### Управление публичными частями доверенных GPG-ключей

Для работы с публичными частями доверенных GPG-ключей используется группа методов API [/configure/trusted_pgp_public_key](/reference/vault_plugin/configure/trusted_pgp_public_key.html).

## Для разработчика

### Настройка GPG-подписи в Git

Стандартный механизм подписи Git позволяет подписывать Git-теги (релиз) и Git-коммиты (публикация) при их создании. В результате GPG-подпись становится неразрывной частью Git-тега или Git-коммита. При использовании этого подхода можно создать только одну подпись.

Плагин [signatures](https://github.com/werf/third-party-git-signatures) позволяет подписывать Git-теги и Git-коммиты, но уже после их создания. В таком случае GPG-подписи сохраняются в [git-notes](https://git-scm.com/docs/git-notes). При этом можно создавать произвольное количество подписей, а также удалять раннее созданные без какого-либо влияния на связанный Git-тег или Git-коммит.

Обе процедуры требуют настроенного gpg и Git для создания GPG-подписей. Выполните необходимые шаги, следуя [инструкции](https://git-scm.com/book/ru/v2/%D0%98%D0%BD%D1%81%D1%82%D1%80%D1%83%D0%BC%D0%B5%D0%BD%D1%82%D1%8B-Git-%D0%9F%D0%BE%D0%B4%D0%BF%D0%B8%D1%81%D1%8C#_%D0%B2%D0%B2%D0%B5%D0%B4%D0%B5%D0%BD%D0%B8%D0%B5_%D0%B2_gpg).

#### Установка плагина signatures

Для использования плагина необходимо установить его в произвольную директорию `PATH` (например, в `~/bin`):
```bash
git clone https://github.com/werf/third-party-git-signatures.git
cd third-party-git-signatures
install bin/git-signatures ~/bin
```

При выполнении команды `git signatures` должно появиться описание плагина:

```bash
git signatures <command> [<args>]

Git Signatures is a system for adding and verifying one or more PGP
signatures to a given git reference.

Git Signatures works by appending one of more signatures of a given
ref hash to the git notes interface for that ref at 'refs/signatures'.

In addition to built in commit signing that allows -authors- to sign,
Git Signatures allows parties other than the author to issue "approval"
signatures to a ref, allowing for decentralized cryptographic proof of
code review. This is also useful for automation use cases where CI
systems to be able to add a signatures to a repo if a repo if all tests
pass successfully.

In practice Git Signatures allows for tamper evident design and brings
strong code attestations to a deployment process.

Commands
--------

* git signatures init
    Setup git to automatically include signatures on push/pull

* git signatures import
    Import all PGP keys specified in .gitsigners file to local
    GnuPG keychain allowing for verifications.

* git signatures show
    Show signatures for a given ref.

* git signatures add
    Add a signature to a given ref.

* git signatures verify
    Verify signatures for a given ref.

* git signatures pull
    Pull all signatures for all refs from origin.

* git signatures push
    Push all signatures for all refs to origin.

* git signatures version
    Report the version number.
```

### Конфигурация сборки

Рассмотрим простой пример создания и организации артефактов релиза для нескольких платформ: организуем доставку скрипта, который при запуске будет выводить тег релиза.

Вся конфигурация сборки, окружение и сборочные инструкции описываются в файле [trdl.yaml](/reference/trdl_yaml.html).

**Важно.** Артефакты релиза должны иметь определённую организацию директорий для доставки на различные платформы и эффективной работы с исполняемыми файлами при использовании trdl-клиента ([подробнее об организации артефактов](/reference/trdl_yaml.html#организация-артефактов-релиза)).

#### trdl.yaml

{% include reference/trdl_yaml/example_trdl_yaml.md.liquid %}

#### build.sh

{% include reference/trdl_yaml/example_build_sh.md.liquid %}

Оба файла добавляем и коммитим в Git.

### Релиз новой версии

Создадим и опубликуем новый Git-тег с GPG-подписью:

```shell
git tag -s v0.0.1 -m 'Signed v0.0.1 tag'
git push origin v0.0.1
```

> Тег определяет версию артефактов релиза и должен соответствовать определённому формату: произвольный [semver](https://semver.org/lang/ru) с префиксом `v`.

После того как Git-тег опубликован, необходимо подписать его достаточным количеством доверенных GPG-ключей. Каждый участник кворума, определённого при [конфигурации плагина](#настройка-плагина), должен подписать Git-тег и опубликовать свою GPG-подпись с помощью Git-плагина [signatures](#установка-плагина-signatures):

```shell
git fetch --tags
git signatures pull
git signatures add v0.0.1
git signatures push
```

> Процесс подписывания может выполняться в один шаг `git signatures add --push v0.0.1`.

Тег создан, необходимое количество GPG-подписей есть — можно переходить непосредственно к релизу.

Для создания релиза используйте метод API [/release](/reference/vault_plugin/release.html#perform-a-release). Проверка, контроль и логирование можно организовывать с помощью методов API [/task/:uuid](/reference/vault_plugin/task/uuid.html), [/task/:uuid/cancel](/reference/vault_plugin/task/uuid/cancel.html) и [/task/:uuid/log](/reference/vault_plugin/task/uuid/log.html).

Упрощённая версия релизного процесса представлена в скрипте `release.sh`, который находится в каталоге [server/examples](https://github.com/werf/trdl/tree/main/server/examples) репозитория проекта.

Перед запуском скрипта необходимо установить четыре переменных окружения:
* `VAULT_ADDR` — адрес, по которому доступен Vault;
* `VAULT_TOKEN` — токен Vault с правами на обращение к endpoint’у, по которому зарегистрирован плагин;
* `PROJECT_NAME` — имя проекта. В нашем случае это путь, по которому зарегистрирован плагин (см. параметр `-path` в разделе «Настройка плагина»);
* `GIT_TAG` — Git-тег.

> При использовании GitHub Actions можно воспользоваться [нашим готовым набором actions](https://github.com/werf/trdl-vault-actions).

### Публикация каналов обновлений

Чтобы у пользователя был доступ к релизу, его нужно опубликовать. Для этого переключитесь на основную ветку, добавьте в репозиторий файл с описанием каналов обновлений [trdl_channels.yaml](/reference/trdl_channels_yaml.html).

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

Добавьте конфигурацию в Git и опубликуйте Git-коммит c GPG-подписью:

```shell
git add trdl_channels.yaml
git commit -S -m 'Signed release channels'
git push
```

После того как Git-коммит опубликован, необходимо подписать его достаточным количеством доверенных GPG-ключей. Каждый участник кворума, определённого при [конфигурации плагина](#настройка-плагина), должен подписать Git-коммит и опубликовать свою GPG-подпись с помощью Git-плагина [signatures](#установка-плагина-signatures):

```shell
git fetch
git signatures pull
git signatures add origin/main
git signatures push
```

> Процесс подписывания может выполняться в один шаг `git signatures add --push origin/main`.

Необходимое количество GPG-подписей добавлено — можно переходить непосредственно к публикации каналов обновлений.

При публикации используйте метод API [/publish](/reference/vault_plugin/publish.html). Проверку, контроль и логирование можно организовать с помощью методов API [/task/:uuid](/reference/vault_plugin/task/uuid.html), [/task/:uuid/cancel](/reference/vault_plugin/task/uuid/cancel.html) и [/task/:uuid/log](/reference/vault_plugin/task/uuid/log.html).

Упрощённая версия процесса публикации представлена в скрипте `publish.sh`, который находится в каталоге [server/examples](https://github.com/werf/trdl/tree/main/server/examples) репозитория проекта.

Так же, как и скрипту `release.sh`, скрипту `publish.sh` требуются переменные окружения:
* `VAULT_ADDR` — адрес, по которому доступен Vault;
* `VAULT_TOKEN` — токен Vault с правами на обращение к endpoint’у, по которому зарегистрирован плагин;
* `PROJECT_NAME` — имя проекта. В нашем случае это путь, по которому зарегистрирован плагин (см. параметр `-path` в разделе «Настройка плагина»).

> При использовании GitHub Actions можно воспользоваться [нашим готовым набором actions](https://github.com/werf/trdl-vault-actions).

## Для пользователя

> Инструкция актуальна для операционных систем Linux, macOS и Windows. Команды можно выполнять в произвольной командной оболочке Unix или в PowerShell для Windows.

### Установка клиента

Скачайте trdl-клиент, следуя инструкциям в сообщении [выбранного релиза](https://github.com/werf/trdl/releases). Загрузите его в каталог, доступный в `PATH` пользователя.

### Использование клиента

При добавлении репозитория пользователю потребуются:
* произвольное имя (`REPO`);
* адрес TUF-репозитория (`URL`);
* номер доверенной версии (`ROOT_VERSION`) и хеш-сумма соответствующего файла `<VERSION>.root.json` (`ROOT_SHA512`), которые вендор должен предоставить пользователю для верификации TUF-репозитория при первичном обращении.

Данные, которые пользователь получает у вендора:

```shell
URL=https://storage.googleapis.com/trdl-test-project-tuf
ROOT_VERSION=1
ROOT_SHA512=$(curl -Ls ${URL}/${ROOT_VERSION}.root.json | sha512sum)
```

Далее пользователь добавляет репозиторий, указав произвольное имя:

```shell
REPO=test
trdl add $REPO $URL $ROOT_VERSION $ROOT_SHA512
```

После этого можно использовать артефакты в рамках желаемого канала обновления:

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
trdl-example.sh
v0.0.1
```
</div>

<div id="windows" class="tabs__content" markdown="1">

```shell
trdl-example.ps1
v0.0.1
```
</div>
