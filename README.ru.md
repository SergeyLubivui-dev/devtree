<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/hero-dark.svg">
  <img alt="devtree — древовидное планирование разработки внутри репозитория" src="docs/hero.svg" width="800">
</picture>

[English](README.md) · **Русский** · [中文](README.zh-CN.md) · [Deutsch](README.de.md) · [Français](README.fr.md)

[![CI](https://github.com/SergeyLubivui-dev/devtree/actions/workflows/ci.yml/badge.svg)](https://github.com/SergeyLubivui-dev/devtree/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/SergeyLubivui-dev/devtree?color=1a7f37)](https://github.com/SergeyLubivui-dev/devtree/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

План проекта — это файл `.devtree/tree.yaml`, который лежит рядом с кодом. Он версионируется вместе
с кодом, ревьюится в пул-реквесте и мержится построчно. Из него devtree генерирует диаграмму
[Mermaid](https://mermaid.js.org/) в `TREE.md` или прямо в ваш `README.md`, а также собственную
отрисовку в `.svg`. GitHub и GitLab рендерят и то, и другое нативно: расширений для браузера не
нужно, картинки руками пересобирать не нужно, хостить нечего.

---

## Зачем держать план в репозитории

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/why-dark.svg">
  <img alt="Три причины: попадает в ревью, нормально мержится, остаётся честным" src="docs/why.svg" width="800">
</picture>

- **Попадает в ревью.** Правка плана видна в диффе, рядом с правкой кода.
- **Нормально мержится.** Список узлов плоский, поэтому две ветки, каждая из которых добавила
  задачу, сливаются без конфликта.
- **Остаётся честным.** Pre-commit хук и проверка в CI не дают диаграмме разойтись с планом.

Дорожная карта в трекере отрывается от ветки, которую описывает. Дорожная карта в вики читается
один раз и больше никогда. Дорожная карта в репозитории — это файл, с которым у людей уже есть
привычки.

---

## Как это работает

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/pipeline-dark.svg">
  <img alt="Цикл: правим tree.yaml, запускаем devtree render, файлы перезаписываются, GitHub их рисует" src="docs/pipeline.svg" width="800">
</picture>

---

## Установка

```bash
go install github.com/SergeyLubivui-dev/devtree@latest
```

Готовые бинарники для Linux, macOS и Windows приложены к каждому
[релизу](https://github.com/SergeyLubivui-dev/devtree/releases/latest), вместе с контрольными
суммами. Или собрать из исходников — нужен только Go 1.22 или новее:

```bash
git clone https://github.com/SergeyLubivui-dev/devtree.git
cd devtree
go test ./...
go build -o devtree .
```

---

## Быстрый старт

```bash
cd ваш-проект

devtree init --project "Платёжный шлюз" --repo https://github.com/acme/pay --hook --action

devtree add "Аутентификация"  -p mvp -b feat/auth -i 12 -o ann -s wip
devtree add "OAuth провайдеры" -p autentifikaciya -b feat/oauth
devtree add "Сброс пароля"     -p autentifikaciya -s blocked -n "ждём SMTP"

devtree ls                                   # дерево в терминале
devtree done oauth-provaydery
git add . && git commit -m "feat: oauth"     # хук сам обновит диаграмму
```

Идентификаторы выводятся из заголовков с транслитерацией — «Аутентификация» превращается в
`autentifikaciya`, чтобы id оставался набираемым на любой раскладке. Свой можно задать через `--id`.

---

## Рецепты на каждый день

Короткие реальные вещи, которые делают с планом. Возьмите любую и поменяйте слова.

**Начать фичу и закончить фичу.** Задача и ветка называются вместе, поэтому по диаграмме видно, где
лежит код:

```bash
devtree add "Фильтры поиска" -p mvp -b feat/search -i 214 -o you -s wip
git switch -c feat/search
# ...пишем код...
devtree done filtry-poiska
git commit -am "feat: фильтры поиска"        # хук обновит диаграмму
```

**Разбить большое на выполнимое.** Родители сами считают детей, поэтому прогресс вехи никогда не
приходится обновлять руками:

```bash
devtree add "Биллинг" -s wip
devtree add "Счета"   -p billing
devtree add "Возвраты" -p billing
devtree add "Напоминания" -p billing -s blocked -n "нужен платёжный API"
devtree ls
```

**Взять тикет.** Номер issue превращается в ссылку в таблице под диаграммой:

```bash
devtree add "Починить сдвиг таймзоны" -i 512 -o ann -s wip --tags bug
```

**Отложить то, что не доделать.** Заблокированная задача без записки — единственное, на что ругается
`check`: «заблокировано» без причины никто не подхватит.

```bash
devtree set sbros-parolya -s blocked -n "ждём договор по SMTP"
devtree board -s blocked
```

**Утро понедельника в трёх командах:**

```bash
devtree board          # что в работе, что застряло, что ждёт
devtree ls -s blocked  # только ветки, где нужно решение
devtree check          # нет ли «готового», под которым осталась открытая работа
```

**Двое, две ветки, один план.** Оба добавляют задачи, оба коммитят, и мерж получается скучным:
список узлов плоский, а `.gitattributes` говорит `merge=union`. Если две ветки случайно выбрали
один id, `devtree check` тут же падает и называет дубликат.

---

## Доска

Дерево говорит, как работа устроена. Доска — в каком она состоянии сегодня утром. Один и тот же
файл, разные вопросы:

```bash
devtree board
```

```text
Платёжный шлюз

☐ not started · 1
  Apple Pay        Приём платежей  #51

◐ in progress · 1
  OAuth провайдеры  Аутентификация  !31

⛔ blocked · 1
  Сброс пароля      Аутентификация  — ждём SMTP

✔ done · 1
  Stripe            Приём платежей  !44
```

На доску попадают только листья. Веха — это контейнер, а не задача, и доска, где контейнеры лежат
вперемешку с работой внутри них, перестаёт быть доской: вместо этого каждая карточка несёт свою веху
хлебной крошкой. Пустые колонки не рисуются вовсе.

Та же доска отрисовывается в SVG — достаточно назвать выходной файл `board.svg`:

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/board-dark.svg">
  <img alt="Доска devtree: колонки работы по статусам" src="docs/board.svg">
</picture>

---

## Команды

| Команда | Что делает |
|---|---|
| `init [--project N] [--repo URL] [--outputs F] [--hook] [--action] [--empty]` | Создаёт `.devtree/tree.yaml`, `.gitattributes` и первую диаграмму |
| `add "Заголовок" [-p ID] [-s СТАТУС] [-b ВЕТКА] [-i N] [--pr N] [-o КТО] [--tags a,b] [-n ЗАМЕТКА] [--id ID]` | Добавляет задачу |
| `set ID [--title T] [-s ...] [-p ...] [...]` | Меняет поля задачи; трогаются только переданные флаги |
| `done ID [ID...]` | Отмечает задачи выполненными |
| `mv ID РОДИТЕЛЬ\|root` | Перевешивает задачу под другого родителя |
| `rm ID [--cascade]` | Удаляет задачу; без `--cascade` дети переходят к её родителю |
| `ls [-s СТАТУС]` | Печатает дерево в терминале |
| `board [-s СТАТУС]` | Печатает работу, сгруппированную по статусам |
| `render [--file F] [--quiet]` | Перегенерирует все выходные файлы |
| `check [--strict]` | Валидирует план — для CI и хуков |
| `install hook\|action\|all` | Ставит pre-commit хук и GitHub Action |
| `outputs` | Печатает список выходных файлов |

Флаги можно ставить и до, и после заголовка — обе записи делают одно и то же:

```bash
devtree add "Аутентификация" -p mvp -s wip
devtree add -p mvp -s wip "Аутентификация"
```

### Статусы

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/statuses-dark.svg">
  <img alt="todo, in_progress, blocked, done, dropped — и сокращения, которые принимает каждый" src="docs/statuses.svg" width="800">
</picture>

В файл всегда попадает каноническое написание, каким бы сокращением вы ни воспользовались.

---

## Формат файла

```yaml
version: 1
project: "Платёжный шлюз"
repo: "https://github.com/acme/pay"
outputs: "TREE.md"
nodes:
  - id: "mvp"
    title: "MVP"
    status: "in_progress"
  - id: "auth"
    title: "Аутентификация"
    status: "in_progress"
    parent: "mvp"
    branch: "feat/auth"
    issue: "12"
    owner: "ann"
```

Список **плоский**, иерархию задаёт поле `parent`. Это осознанный размен. Вложенность означала бы,
что добавление задачи переписывает существующий блок — ровно та форма, из-за которой конфликтуют две
ветки. Плоский список превращает «добавить задачу» в «дописать строки», поэтому параллельные ветки
сливаются чисто, а `init` дополнительно прописывает правило `merge=union` в `.gitattributes`. Худшее,
что может случиться, — дублирующийся id, и его сразу ловит `devtree check`.

Файл можно править руками. Парсер строгий и называет номер строки:

```text
devtree: .devtree/tree.yaml: line 14: unknown node field "assignee"
```

---

## Автоматизация

**Pre-commit хук** — `devtree install hook`. Проверяет план, пересобирает все выходные файлы и
докладывает результат в текущий коммит. Существующий хук сохраняется как `pre-commit.devtree-backup`.
Коллеги без установленного devtree не блокируются: хук замечает отсутствие бинарника и отходит в
сторону.

**GitHub Action** — `devtree install action`. На `pull_request` падает, если диаграмма устарела,
чтобы автор поправил её в своём же диффе; на `push` в основную ветку сам обновляет и коммитит.

---

## Отрисовка в SVG

У Mermaid есть потолок: GitHub рендерит его со строгим санитайзером и CSP уровня страницы, поэтому
узел диаграммы не может нести ни иконку, ни ссылку. У файла, который devtree рисует сам, потолка
нет. Имя файла решает всё:

| Имя файла | Что рисуется |
|---|---|
| `TREE.md`, `README.md` | блок Mermaid между маркерами |
| `docs/tree.svg` | дерево, светлая палитра |
| `docs/tree-dark.svg` | дерево, тёмная палитра |
| `docs/board.svg` | доска |

Три вещи в отрисовке двигаются, и каждая несёт смысл: штрих бежит по ребру, ведущему в задачу «в
работе», глиф вращается на задаче, которая прямо сейчас в движении, и полоса прогресса один раз
вырастает при загрузке. Всё остальное стоит на месте. Читателям, чья система просит меньше
движения, достаётся статичная картинка.

Глифы взяты из [Reicon](https://reicon.dev) (MIT) как данные путей в `internal/icons` — см.
[NOTICE](NOTICE). Во время отрисовки ничего не скачивается, зависимостей у бинарника по-прежнему нет.

---

## Ограничения

- Формат хранения — строгое подмножество YAML: плоский список скалярных полей. Якоря, многострочные
  блоки и вложенные структуры не поддерживаются и отвергаются с номером строки.
- Ничто, отрисованное в README, не кликабельно: Mermaid на GitHub игнорирует директивы `click`, а
  SVG, отданный как картинка, работает в песочнице. Ссылки живут в свёрнутой таблице под диаграммой.
- Текст в SVG измеряется оценкой, а не метриками шрифта — иначе пришлось бы поставлять шрифт.
  Карточки на пару пикселей просторнее, чем нужно, чтобы это компенсировать.
- Очень широкие деревья (сотни узлов) браузер рисует медленно; разносите их по нескольким выходным
  файлам через `--outputs`.

---

## Лицензия

[MIT](LICENSE) © SergeyLubivui-dev

Векторные глифы в `internal/icons` взяты из [Reicon](https://reicon.dev), тоже MIT — см.
[NOTICE](NOTICE).

> Полная документация, включая раздел об устройстве проекта, — в [английской версии](README.md).
