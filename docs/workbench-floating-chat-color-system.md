# 工作台浮动对话 Mock 配色记录

记录时间：2026-05-10

来源文件：`docs/workbench-floating-chat-mock.html`

这套配色后续作为主站深色工作台方向的参考。核心感觉是“深色玻璃面板 + 低饱和蓝紫光 + 少量状态色”，不要改成大面积纯蓝、纯紫或高饱和渐变。

## 核心变量

```css
:root {
  color-scheme: dark;
  --bg: #070b12;
  --panel: rgba(12, 18, 32, 0.82);
  --panel-solid: #11192d;
  --line: rgba(128, 151, 198, 0.22);
  --line-strong: rgba(104, 119, 255, 0.48);
  --text: #edf4ff;
  --muted: #8e9fbb;
  --soft: #61718c;
  --blue: #37a3ff;
  --violet: #776bff;
  --green: #2bd59f;
  --amber: #f6bd4d;
  --red: #ff6d7e;
  --shadow: 0 24px 70px rgba(0, 0, 0, 0.42);
}
```

## 使用原则

- 页面底色：`#02050b` 到 `#070c18` 的深黑蓝，不用纯黑。
- 主面板：优先用 `rgba(12, 18, 32, 0.72~0.84)` 叠加 `backdrop-filter: blur(18px~28px)`。
- 实体面板：用 `#11192d`，适合不透明侧栏、输入框内部、按钮底。
- 边框：默认 `rgba(128, 151, 198, 0.22)`；选中态和重点浮层用 `rgba(104, 119, 255, 0.48)`。
- 主文字：`#edf4ff`；次级文字用 `#8e9fbb`；弱提示用 `#61718c`。
- 主操作：蓝紫渐变 `#6d70ff -> #8b5cf6` 或 `#6877ff -> #7364ee`，面积要小，主要用于按钮和高亮。
- 状态色：执行中用 `#2bd59f`，等待交互用 `#f6bd4d`，新消息/风险提示用 `#ff6d7e`。
- 光效：只做低透明径向光和玻璃阴影，避免大块霓虹边框。

## 背景与玻璃感

```css
background:
  radial-gradient(circle at 48% 40%, rgba(52, 162, 255, 0.13), transparent 34%),
  radial-gradient(circle at 86% 11%, rgba(119, 107, 255, 0.15), transparent 28%),
  linear-gradient(180deg, #070c18 0%, #060912 100%);
```

网格线使用：

```css
linear-gradient(rgba(72, 107, 164, 0.045) 1px, transparent 1px)
```

重点是“隐约有结构”，不要让网格成为主视觉。

## 迁移备注

主站落地时建议先抽成 `theme-workbench-dark.css` 或 CSS variables，替换工作台容器、左侧目录、浮动输入框、后台会话托盘、流式输出面板的颜色。现有 Element Plus 默认色需要局部覆盖，否则会破坏这套柔和深色玻璃感。
