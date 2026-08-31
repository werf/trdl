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

### Сборочный бекенд

Во время релиза плагин собирает релизные артефакты с помощью `docker buildx` во временном сборщике, создаваемом на каждую сборку. По умолчанию используется драйвер `docker-container`, которому нужен доступ к Docker-демону (см. выше).

В Kubernetes-инсталляциях сборку можно делегировать BuildKit-подам внутри кластера, используя buildx-драйвер `kubernetes`. Драйвер настраивается переменными окружения процесса Vault:

* `TRDL_BUILDX_DRIVER` — используемый buildx-драйвер: `docker-container` (по умолчанию) или `kubernetes`;
* `TRDL_BUILDX_DRIVER_OPTS_<ЛЮБОЙ_СУФФИКС>` — дополнительные значения `--driver-opt` для `docker buildx create`. Таких переменных может быть любое количество, каждая содержит одну опцию; они применяются в лексикографическом порядке имён переменных. Значение передаётся без изменений, поэтому опции с запятыми, например `nodeselector=disktype=ssd,zone=a`, не требуют экранирования;
* `TRDL_BUILDX_DRIVER_OPTS_SEPARATOR` — если задан, значение каждой переменной разделяется на несколько опций по этому разделителю. По умолчанию не задан, то есть одна переменная содержит ровно одну опцию.

Пример конфигурации для драйвера `kubernetes`:

```shell
TRDL_BUILDX_DRIVER=kubernetes
TRDL_BUILDX_DRIVER_OPTS_NAMESPACE=namespace=trdl-build
TRDL_BUILDX_DRIVER_OPTS_ROOTLESS=rootless=true
```

или та же конфигурация в одной переменной с явным разделителем:

```shell
TRDL_BUILDX_DRIVER=kubernetes
TRDL_BUILDX_DRIVER_OPTS_SEPARATOR=;
TRDL_BUILDX_DRIVER_OPTS_KUBE='namespace=trdl-build;rootless=true'
```

Те же две настройки доступны и в конфигурации плагина, отдельно для каждого проекта. Это единственный способ задать их, когда плагин вкомпилен в хост-процесс, окружением которого администратор не управляет:

* `buildx_driver` — те же значения, что и у `TRDL_BUILDX_DRIVER`;
* `buildx_driver_opts` — список значений `--driver-opt`, по одному в элементе, каждое передаётся без изменений.

```json
{
  "buildx_driver": "kubernetes",
  "buildx_driver_opts": ["namespace=trdl-build", "rootless=true"]
}
```

Каждая из двух настроек разрешается независимо: конфигурация плагина имеет приоритет над переменными окружения, а переменные окружения — над умолчанием, то есть драйвером `docker-container` без опций. Поле, не переданное в `configure` или переданное пустым, означает «не задано» и отдаёт решение переменным окружения, а не перекрывает их пустым значением. Чтобы собирать вообще без опций драйвера, когда в окружении они заданы, эти переменные нужно снять.

Ни одно из этих полей нельзя задать вместе с `buildkitd_address` или `buildkitd_driver` (см. ниже): эти настройки заменяют весь buildx-путь, сборщик через CLI `docker` не создаётся, и настройки драйвера ни на что не влияли бы — поэтому `configure` отвергает такую комбинацию, а не игнорирует их. Проверка касается только одного вызова `configure`. Переменная `TRDL_BUILDKITD_ADDRESS`, заданная процессу Vault, тоже имеет приоритет над этими настройками, но она процессная и может измениться уже после настройки проекта, поэтому отвергнуть её на этом этапе нельзя: вместо этого сборка сообщает в лог плагина, что настройки не используются.

Особенности драйвера `kubernetes`:

* целевой namespace должен существовать, а процессу Vault нужны права на управление Deployment и Pod в нём: сборщик работает как BuildKit Deployment и удаляется после сборки;
* кластер определяется стандартным способом — через kubeconfig или in-cluster ServiceAccount;
* rootless BuildKit (`rootless=true`) не укладывается и в уровень PodSecurity `baseline`: buildx задаёт поду сборщика `seccompProfile: Unconfined` и AppArmor-аннотацию `unconfined`, а оба этих значения запрещены уже на `baseline`. Namespace сборщика должен быть помечен как `privileged` (либо исключён из PodSecurity-admission);
* доступные опции драйвера — в [документации buildx kubernetes driver](https://docs.docker.com/build/builders/drivers/kubernetes/).

#### Сборка без buildx-драйверов

Оба buildx-драйвера выше запускают CLI `docker`, поэтому требуют наличия бинарника рядом с плагином. Этот путь заменяют две настройки, и обе обращаются к BuildKit через Go-клиент:

* `buildkitd_driver` — плагин сам поднимает временный `buildkitd`, по одному на сборку, и удаляет его после неё. Внешних бинарников он не запускает вовсе, поэтому годится там, где `docker` отсутствует, — например когда плагин встроен в другой процесс, поставляемый в distroless-образе;
* `buildkitd_address` — плагин подключается к `buildkitd`, который запускает кто-то другой, и ничего не создаёт. Нужен ли бинарник, зависит от схемы: `unix://` и `tcp://` обходятся без него, а `docker-container://` и `kube-pod://` по-прежнему запускают `docker` и `kubectl` соответственно (см. ниже).

Задать можно только один из них, и `configure` отвергает любой из них рядом с *полями* buildx. С *переменными окружения* buildx иначе: `TRDL_BUILDX_DRIVER` и `TRDL_BUILDX_DRIVER_OPTS_*` — это запасной вариант для полей, и заданный `buildkitd_driver` или `buildkitd_address` просто побеждает их, потому что процессную переменную нельзя отвергнуть в момент настройки проекта.

##### Временный buildkitd, поднимаемый плагином

`buildkitd_driver=kubernetes` запускает сборщик подом на время одной сборки:

```shell
vault write trdl-test-project/configure ... \
  buildkitd_driver=kubernetes \
  buildkitd_driver_opts=namespace=trdl-build \
  buildkitd_driver_opts=serviceaccount=trdl-buildkit
```

или в виде JSON:

```json
{
  "buildkitd_driver": "kubernetes",
  "buildkitd_driver_opts": ["namespace=trdl-build", "serviceaccount=trdl-buildkit"]
}
```

Опции — пары `name=value`, по одной на элемент списка, передаются как есть, поэтому значение с запятыми вроде `nodeselector=disktype=ssd,zone=a` не нужно экранировать. Они проверяются в момент записи конфигурации: опцию, которую драйвер не умеет применять, отвергает `configure`, а не упавший позже релиз. Драйвер `kubernetes` принимает:

| Опция | Значение |
|---|---|
| `namespace` | namespace для сборщика; по умолчанию — namespace собственного ServiceAccount плагина либо текущего контекста kubeconfig, а если не задан ни там ни там — `default` |
| `image` | образ buildkitd; по умолчанию `moby/buildkit:buildx-stable-1`, а при `rootless=true` — его вариант `-rootless` |
| `rootless` | запускать rootless BuildKit |
| `serviceaccount` | ServiceAccount для пода сборщика |
| `nodeselector`, `labels`, `annotations` | пары `name=value` через запятую |
| `requests.cpu`, `requests.memory`, `requests.ephemeral-storage` | requests пода |
| `limits.cpu`, `limits.memory`, `limits.ephemeral-storage` | limits пода |
| `timeout` | сколько ждать готовности сборщика, например `5m`; по умолчанию `2m` |
| `deadline` | жёсткий предел времени жизни пода сборщика (`activeDeadlineSeconds`), например `2h`; по умолчанию не задан, задаётся целым числом секунд не меньше `1s`. Это не льготный период: сборка, которая к этому моменту ещё идёт, тоже будет убита, поэтому значение берут заведомо больше самого долгого релиза проекта |

Имена опций совпадают с опциями buildx-драйвера `kubernetes` там, где опции пересекаются, но набор здесь свой, а не buildx: опции, которые принимает buildx и не принимает этот драйвер — `replicas`, `loadbalance`, `tolerations`, `schedulername`, `qemu.*`, опции постоянного тома, — отвергаются, а у `deadline` соответствия в buildx нет вовсе.

В целевом namespace плагину нужны `create`, `get` и `delete` на `pods` плюс `create` на `pods/exec`; namespaced-роли достаточно. Группа API `apps` не используется. Поток сборки идёт через exec-канал API-сервера, поэтому сетевой доступ от плагина к поду сборщика не нужен. Кластер определяется стандартным способом — через kubeconfig или in-cluster ServiceAccount.

Особенности пода:

* namespace должен существовать и должен пропускать под сборщика. По умолчанию контейнер запускается `privileged`; при `rootless=true` он непривилегированный, но требует seccomp `Unconfined` и AppArmor-аннотации `unconfined`. И то и другое запрещено на уровне PodSecurity `baseline`, поэтому namespace должен быть помечен как `privileged` либо исключён из PodSecurity-admission — ровно то же требование, что и у buildx-драйвера `kubernetes`. В отличие от buildx-пути, отказ приходит прямо из вызова `create`, а не в виде таймаута готовности;
* под удаляется по окончании сборки, в том числе при её падении и отмене, а также если сборщик так и не стал готов. Он не удаляется, если сам процесс плагина умер целиком, — на этот случай есть `deadline`, ограничивающий время жизни брошенного сборщика ценой такого же ограничения для нормальной сборки;
* сборщик — именно отдельный Pod с `restartPolicy: Never`, и это осознанно: его нельзя подменять посреди сборки, потому что замена окажется сборщиком, с которым релиз не связан.

Из того, что под создаёт сам плагин своими кредами, следуют две вещи, и обе — предмет решения администратора:

* **Право записи в `configure` становится правом создавать поды.** Кто может писать конфигурацию проекта, тот выбирает `namespace`, в котором поднимется сборщик, `serviceaccount`, под которым он побежит, и `image`, который в нём запустится, — везде, куда достаёт Role самого плагина. Ограничьте доступ на запись в `<project>/configure` тем же кругом, которому доверены релизные ключи, а Role плагина — namespace'ами, в которых больше нечего брать. Сборщику давайте ServiceAccount, права которого проверены целиком: создание workload'а позволяет запуститься от имени любого ServiceAccount этого namespace, а ServiceAccount несёт всё, что привязано к нему напрямую, через namespaced RoleBinding, через ClusterRoleBinding и через групповые привязки вроде `system:serviceaccounts`. Отсутствие RoleBinding в namespace само по себе гарантией изоляции не является, и токен под сборщика монтируется.
* **Контейнер сборщика по умолчанию привилегированный.** Этого требует BuildKit; `rootless=true` меняет привилегию на seccomp `Unconfined` и AppArmor-аннотацию `unconfined`. В любом случае сборка исполняет инструкции проекта в контейнере, который уровень PodSecurity `baseline` не пропустил бы, — значит и namespace, и узлы под неё надо давать те, которые не жалко, а не те, где лежат ключи подписи.

##### Подключение к существующему buildkitd

Сборку можно направить на уже запущенный `buildkitd`: плагин обращается к нему напрямую через клиент BuildKit, и сборщик не создаётся и не удаляется на каждую сборку.

Адрес buildkitd задаётся для каждого проекта в конфигурации плагина:

```shell
vault write trdl-test-project/configure ... buildkitd_address=tcp://buildkitd.trdl-build.svc:1234
```

или, как запасной вариант для всех проектов, переменной окружения `TRDL_BUILDKITD_ADDRESS` процесса Vault. Значение из конфигурации проекта имеет приоритет. Поддерживаемые схемы адреса:

* `unix://` и `tcp://` — прямое gRPC-соединение с buildkitd, внешние бинарники не нужны;
* `docker-container://` и `kube-pod://` — соединение через `docker exec`/`kubectl exec`, требуется соответствующий CLI.

Если не заданы ни `buildkitd_address`, ни `buildkitd_driver`, сборка идёт через `docker buildx`, как описано выше. Развёртывание самого `buildkitd` находится вне зоны ответственности trdl: в Kubernetes это обычно Deployment или StatefulSet в отдельном namespace, метки PodSecurity которого задаёт владелец кластера, поскольку BuildKit требует ослабленный профиль seccomp/AppArmor даже в rootless-режиме.

Защита канала и изоляция демона — ответственность администратора. Через это соединение плагин передаёт весь контекст сборки и все секреты сборки: секреты проекта, а если настроена подпись для macOS — сертификат подписи, его пароль и notary-ключ. Что из этого следует:

* **`tcp://` — соединение без шифрования и без аутентификации.** Плагин не шифрует трафик и не проверяет идентичность демона, к которому подключается, поэтому любой, кто способен перехватить соединение или занять адрес endpoint'а, получает эти секреты. Используйте `tcp://` только поверх канала, конфиденциальность и аутентичность которого обеспечены иными средствами: сетевой сегмент, недоступный другим нагрузкам, service mesh с mTLS или равноценный туннель. Если это не гарантировано, запускайте buildkitd рядом с плагином и используйте `unix://` с общим сокетом.
* **Адрес — граница доверия.** Тот, кто может записать конфигурацию проекта или задать `TRDL_BUILDKITD_ADDRESS` для процесса Vault, выбирает, какой демон получит секреты релиза. Право записи в `<проект>/configure` должно быть только у тех, кому доверены ключи релиза.
* **Демон общий и ничем не ограничен.** buildkitd выполняет инструкции сборки проекта, при этом сборщик не создаётся и не удаляется на каждую сборку — параллельные релизы и все проекты, направленные на один адрес, используют один экземпляр, его кэш и его привилегии. Выделяйте отдельный экземпляр на каждый домен доверия и считайте доступ к нему доступом к выпускаемым артефактам.

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

**Добавление ключа**

```shell
vault write trdl-test-project/configure/trusted_pgp_public_key name=developer public_key=@developer.pgp
```

где, `developer.pgp` — файл с публичным PGP-ключом, полученный в результате вызова команды `gpg --armor --output developer.pgp --export developer@trdl.dev`.

{% offtopic title="Контент файла developer.pgp" %}
```shell
-----BEGIN PGP PUBLIC KEY BLOCK-----

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
XaOlJrPDM5E9zw==
=bIYD
-----END PGP PUBLIC KEY BLOCK-----
```
{% endofftopic %}

Подробнее про экспорт с gpg можно почитать [в документации утилиты](https://www.gnupg.org/gph/en/manual/x56.html#AEN64).

**Листинг ключей**

```shell
vault read trdl-test-project/configure/trusted_pgp_public_key
````

```shell
Key     Value
---     -----
keys    [developer]
```

**Просмотр ключа**

```shell
vault read trdl-test-project/configure/trusted_pgp_public_key/developer
```

{% offtopic title="Вывод команды" %}
```shell
Key           Value
---           -----
name          developer

public_key    -----BEGIN PGP PUBLIC KEY BLOCK-----

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
TJ9EZRY=
=vkcq
-----END PGP PUBLIC KEY BLOCK-----
```
{% endofftopic %}

**Удаление ключа**

```shell
vault delete trdl-test-project/configure/trusted_pgp_public_key/developer 
````

```
Success! Data deleted (if it existed) at: trdl-test-project/configure/trusted_pgp_public_key/developer
```

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

Вся конфигурация сборки — окружение и сборочные инструкции — описывается в файле [trdl.yaml](/reference/trdl_yaml.html).

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

При добавлении репозитория пользователю потребуются данные, которые вендор должен предоставить пользователю для верификации TUF-репозитория при первичном обращении: адрес TUF-репозитория (`URL`), номер доверенной версии (`ROOT_VERSION`) и хеш-сумма соответствующего файла `<VERSION>.root.json` (`ROOT_SHA512`).

В нашем случае пользователь получает следующие данные от вендора:

```shell
URL=https://storage.googleapis.com/trdl-test-project-tuf
ROOT_VERSION=1
ROOT_SHA512=$(curl -Ls ${URL}/${ROOT_VERSION}.root.json | sha512sum | cut -c 1-128)
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
