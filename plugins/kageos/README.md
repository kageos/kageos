# kageos 套件

`kageos` 是面向 Codex 和 Claude Code 的中文开发套件：既帮助用户在本地电脑拉取、启动、排障和贡献 kageos 平台源码，也能把一个 kageos 工作空间目录从设计、开发、构建、真实验收一直推进到 Hub 发布和状态确认。

套件沿用 kageos 官网品牌标识，插件卡片和 Codex 输入区图标位于 `assets/`。

套件包含五个 Skill：

- `kageos`：默认统一入口，负责完整闭环和失败续跑。
- `kageos-contributor`：检查环境、拉取和启动平台源码、排障、源码导览和贡献准备。
- `kageos-developer`：目录设计、开发、本地检查和平台 build/update。
- `kageos-operator`：真实业务操作、验收、测试数据清理和证据报告。
- `kageos-hub-publisher`：目录包、截图、元数据、确认、投稿和状态查询。

## Codex 使用

安装后新建一个任务，输入：

```text
$kageos 检查我的电脑环境，拉取并启动 kageos，成功后打开本地页面。
```

目录完整交付使用：

```text
$kageos 把 /user/app/package 从设计、开发、构建、真实验收一直做到 Hub 投稿。
```

若通过仓库开发 marketplace 测试：

```bash
codex plugin marketplace add /absolute/path/to/kageos
codex plugin add kageos@kageos-dev
```

个人 marketplace 使用 `~/.agents/plugins/marketplace.json`，不需要额外注册 marketplace。宿主不支持插件命令时，可使用官网安装脚本把套件放入个人插件目录，然后在 Codex 的 Plugins 页面启用。

## Claude Code 使用

开发时直接加载：

```bash
claude --plugin-dir /absolute/path/to/kageos/plugins/kageos
```

完整流程使用：

```text
/kageos:kageos 帮我把 kageos 拉到本地并启动。
```

目录完整交付使用：

```text
/kageos:kageos 把这个目录从设计做到发布。
```

## 更新和打包

仓库 `skills/` 是五个 Skill 的唯一编辑源。修改后运行：

```bash
python3 ../../scripts/package-kageos-plugin.py
python3 ../../scripts/release-kageos-plugin.py \
  --website-root ../../../kageos-website
```

不要直接编辑本目录下 `skills/` 中的生成副本。版本号以 `VERSION`、两个插件 manifest 和 `compatibility.json` 一致为准。

完整更新记录见 `CHANGELOG.zh-CN.md`。
