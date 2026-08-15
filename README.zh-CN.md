<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/hero-dark.svg">
  <img alt="devtree — 住在仓库里的树形开发计划" src="docs/hero.svg" width="800">
</picture>

[English](README.md) · [Русский](README.ru.md) · **中文** · [Deutsch](README.de.md) · [Français](README.fr.md)

[![CI](https://github.com/SergeyLubivui-dev/devtree/actions/workflows/ci.yml/badge.svg)](https://github.com/SergeyLubivui-dev/devtree/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/SergeyLubivui-dev/devtree?color=1a7f37)](https://github.com/SergeyLubivui-dev/devtree/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

你的开发计划就是一个文件：`.devtree/tree.yaml`，它和代码放在一起，跟着代码一起进版本库，在拉取请求里
接受评审，按行合并。devtree 从它生成 [Mermaid](https://mermaid.js.org/) 图，写进 `TREE.md` 或者直接
写进你的 `README.md`，也可以生成自己绘制的 `.svg`。GitHub 和 GitLab 都原生渲染这两种输出：不需要浏览器
扩展，不需要手工重新导出图片，也没有任何东西要托管。

---

## 为什么把计划放进仓库

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/why-dark.svg">
  <img alt="三个理由：会被评审、能被合并、保持诚实" src="docs/why.svg" width="800">
</picture>

- **它会被评审。** 对计划的修改出现在 diff 里，就在对代码的修改旁边。
- **它能被合并。** 节点列表是扁平的，两个分支各自加了一个任务也能干净地合并。
- **它保持诚实。** 提交前钩子和 CI 检查不允许图和计划走样。

放在工单系统里的路线图会和它描述的分支渐行渐远，放在 wiki 里的路线图只会被读一次。放在仓库里的路线图，
是一个大家早就有使用习惯的文件。

---

## 工作方式

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/pipeline-dark.svg">
  <img alt="循环：编辑 tree.yaml，运行 devtree render，文件被重写，GitHub 渲染" src="docs/pipeline.svg" width="800">
</picture>

---

## 安装

```bash
go install github.com/SergeyLubivui-dev/devtree@latest
```

每个[发行版](https://github.com/SergeyLubivui-dev/devtree/releases/latest)都附带 Linux、macOS 和
Windows 的预编译二进制文件以及校验和。也可以从源码构建，只需要 Go 1.22 或更高版本：

```bash
git clone https://github.com/SergeyLubivui-dev/devtree.git
cd devtree
go test ./...
go build -o devtree .
```

---

## 快速上手

```bash
cd 你的项目

devtree init --project "支付网关" --repo https://github.com/acme/pay --hook --action

devtree add "Authentication"  -p mvp -b feat/auth -i 12 -o ann -s wip
devtree add "OAuth providers" -p authentication -b feat/oauth
devtree add "Password reset"  -p authentication -s blocked -n "等待 SMTP"

devtree ls                                   # 在终端里看这棵树
devtree done oauth-providers
git add . && git commit -m "feat: oauth"     # 钩子会自动刷新图
```

标识符由标题推导而来："OAuth providers" 变成 `oauth-providers`；非拉丁字母的标题会被转写，保证
标识符始终打得出来。想自己指定就用 `--id`。

---

## 日常用法

都是真实会做的小事，复制一段改掉文字即可。

**开一个功能，做完一个功能。** 任务和分支一起命名，看图的人就知道代码在哪里：

```bash
devtree add "Search filters" -p mvp -b feat/search -i 214 -o you -s wip
git switch -c feat/search
# ...写代码...
devtree done search-filters
git commit -am "feat: search filters"        # 钩子会刷新图
```

**把大事拆成能做的事。** 父节点会自动统计子节点，里程碑的进度永远不需要手工更新：

```bash
devtree add "Billing" -s wip
devtree add "Invoices" -p billing
devtree add "Refunds"  -p billing
devtree add "Dunning"  -p billing -s blocked -n "依赖支付 API"
devtree ls
```

**接一个工单。** 填上 issue 号，它就会变成图下方表格里的链接：

```bash
devtree add "Fix timezone drift" -i 512 -o ann -s wip --tags bug
```

**把做不完的事先挂起。** 被阻塞却没写原因的任务是 `check` 唯一会抱怨的东西——没有原因的"阻塞"没人接得住：

```bash
devtree set password-reset -s blocked -n "等待 SMTP 合同"
devtree board -s blocked
```

**周一早上的三条命令：**

```bash
devtree board          # 什么在进行、什么卡住了、什么在等待
devtree ls -s blocked  # 只看需要拍板的分支
devtree check          # 有没有标成完成、下面却还开着的工作
```

**两个人、两个分支、一份计划。** 双方各自加任务、各自提交，合并过程平淡无奇：节点列表是扁平的，
`.gitattributes` 里写着 `merge=union`。万一两个分支挑中了同一个标识符，`devtree check` 会当场失败并
指出重复的那个。

---

## 看板

树回答的是"这些工作怎么组织"，看板回答的是"今天早上它们处于什么状态"。同一个文件，两个问题：

```bash
devtree board
```

```text
支付网关

☐ not started · 1
  Apple Pay        Payments  #51

◐ in progress · 1
  OAuth providers  Authentication  !31

⛔ blocked · 1
  Password reset   Authentication  — 等待 SMTP

✔ done · 1
  Stripe           Payments  !44
```

只有叶子节点会出现在看板上。里程碑是容器而不是任务，把容器和它内部的工作并排列出来的看板就不再是看板
了——所以每张卡片改为把里程碑作为面包屑带在身上。空的列干脆不画。

同一个看板也能渲染成 SVG，只要把输出文件命名为 `board.svg`：

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/board-dark.svg">
  <img alt="devtree 自己的看板：按状态分列的工作" src="docs/board.svg">
</picture>

---

## 已完结的工作

一份保留了历史上每一个已完成任务的计划就不再是计划，而是日志：看板被"已完成"的列塞满，图上长出一条
没人会读的尾巴。归档保留记录，但不保留噪声。

```bash
devtree archive          # 哪些已经完结、可以移走
devtree archive --all    # 移入 .devtree/archive.yaml
devtree archive v1       # 或者只移这一支
devtree archive --list   # 已经移走的有哪些
devtree restore v1       # 连同它下面的一切一起取回
```

在你开口之前什么都不会动：单独的 `devtree archive` 只做汇报。只有当一个节点的整个子树都是 `done`
或 `dropped` 时它才够格，所以活着的工作绝不会随着上面的里程碑一起消失。归档的任务保留原来的父节点，
这是它记住自己来处的方式；如果取回时父节点已经不在了，任务会作为根节点回来，并且明确告诉你。

归档使用和计划完全相同的格式：没有第二套解析器要学，diff 里也不会出现新东西。

**关掉 git 已经知道的事情：**

```bash
devtree sync           # 列出分支已被合并的任务
devtree sync --apply   # 把它们标记为完成
```

它只提议，不动手。git 知道哪些分支被合并了，却不知道其中哪些是*做完了*才合并的——在功能开关后面合并
的分支并不等于任务已完成。这是唯一会调用外部程序的命令，其余一切只和文件打交道。

---

## 命令

| 命令 | 作用 |
|---|---|
| `init [--project N] [--repo URL] [--outputs F] [--hook] [--action] [--empty]` | 创建 `.devtree/tree.yaml`、`.gitattributes` 和第一张图 |
| `add "标题" [-p ID] [-s 状态] [-b 分支] [-i N] [--pr N] [-o 负责人] [--tags a,b] [-n 备注] [--id ID]` | 添加任务 |
| `set ID [--title T] [-s ...] [-p ...] [...]` | 修改任务字段；只改你传入的那些 |
| `done ID [ID...]` | 把任务标记为完成 |
| `mv ID 父节点\|root` | 把任务挂到另一个父节点下 |
| `rm ID [--cascade]` | 删除任务；不加 `--cascade` 时子节点上移到它的父节点 |
| `ls [-s 状态]` | 在终端打印这棵树 |
| `board [-s 状态]` | 按状态分组打印工作 |
| `archive [ID...] [--all] [--list]` | 把已完结的计划分支移入归档 |
| `restore ID [ID...]` | 把归档里的工作取回来 |
| `sync [--apply]` | 关闭那些分支已被 git 合并的任务 |
| `render [--file F] [--quiet]` | 重新生成所有输出文件 |
| `check [--strict]` | 校验计划——供 CI 和钩子使用 |
| `install hook\|action\|all` | 安装提交前钩子和 GitHub Action |
| `outputs` | 打印输出文件列表 |

标志位放在标题前后都可以，两种写法完全等价：

```bash
devtree add "Authentication" -p mvp -s wip
devtree add -p mvp -s wip "Authentication"
```

### 状态

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/statuses-dark.svg">
  <img alt="todo、in_progress、blocked、done、dropped 以及各自接受的简写" src="docs/statuses.svg" width="800">
</picture>

无论你输入哪种简写，落到文件里的始终是标准写法。

---

## 文件格式

```yaml
version: 1
project: "支付网关"
repo: "https://github.com/acme/pay"
outputs: "TREE.md"
nodes:
  - id: "mvp"
    title: "MVP"
    status: "in_progress"
  - id: "auth"
    title: "Authentication"
    status: "in_progress"
    parent: "mvp"
    branch: "feat/auth"
    issue: "12"
    owner: "ann"
```

这个列表是**扁平的**，层级关系由 `parent` 字段决定。这是一次有意的取舍：嵌套意味着添加一个任务就要
重写一整块已有内容——正是这种形状让两个功能分支互相冲突。扁平列表把"添加任务"变成"追加几行"，并行分支
因此能干净合并，`init` 还会在 `.gitattributes` 里写上 `merge=union`。最坏的情况不过是标识符重复，而
`devtree check` 会立刻抓到它。

文件随时可以手工编辑，解析器很严格，并且会告诉你出错的行号：

```text
devtree: .devtree/tree.yaml: line 14: unknown node field "assignee"
```

---

## 自动化

**提交前钩子** —— `devtree install hook`。它校验计划、重新生成所有输出，并把结果一起加入本次提交。
已有的钩子会被保留为 `pre-commit.devtree-backup`。没装 devtree 的同事不会被挡住：钩子发现找不到二进制
文件就会自动让路。

**GitHub Action** —— `devtree install action`。在 `pull_request` 上，如果图过期就让流水线失败，让作者
在自己的 diff 里修好；在推送到主分支时，自动刷新并提交。

---

## 绘制成 SVG

Mermaid 有天花板：GitHub 用严格的净化器和页面级 CSP 渲染它，所以图中的节点既不能带图标也不能带链接。
devtree 自己绘制的文件没有这个限制。文件名决定一切：

| 文件名 | 画出什么 |
|---|---|
| `TREE.md`、`README.md` | 标记之间的 Mermaid 块 |
| `docs/tree.svg` | 树，浅色配色 |
| `docs/tree-dark.svg` | 树，深色配色 |
| `docs/board.svg` | 看板 |

图里有三样东西在动，每一样都携带信息：一条虚线沿着通往"进行中"任务的连线流动，正在推进的任务上的图标
缓慢旋转，进度条在加载时增长一次。其余一切保持静止。如果读者的系统要求减少动效，得到的就是一张静止的图。

图标来自 [Reicon](https://reicon.dev)（MIT 许可），以路径数据形式内置在 `internal/icons` 中，见
[NOTICE](NOTICE)。绘制时不会下载任何东西，二进制文件依然没有依赖。

---

## 局限

- 存储格式是 YAML 的一个严格子集：由标量字段组成的扁平列表。锚点、多行块和嵌套结构都不支持，遇到时会
  带着行号被拒绝。
- README 里渲染出来的东西都不可点击：GitHub 的 Mermaid 会忽略 `click` 指令，而作为图片提供的 SVG 处于
  沙箱中。链接放在图下方那张折叠表格里。
- SVG 中的文字宽度是估算的，而不是取自字体度量——否则就得随程序分发一份字体。卡片因此会宽出几个像素。
- 非常宽的树（几百个节点）在浏览器里渲染很慢，用 `--outputs` 把它们拆到多个输出文件里。

---

## 许可

[MIT](LICENSE) © SergeyLubivui-dev

`internal/icons` 中的矢量图标来自 [Reicon](https://reicon.dev)，同为 MIT 许可，见 [NOTICE](NOTICE)。

> 完整文档（包括项目架构一节）见[英文版](README.md)。
