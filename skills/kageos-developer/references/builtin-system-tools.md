# 安装自带系统工具

`/system/tools` 是 kageos 安装自带的固定系统工具目录，不是普通用户临时创建的业务目录。做方案或开发前，先判断需求是否能复用这里的通用能力。

## 什么时候优先复用

优先复用 `/system/tools`：

- 一次性文件处理、格式转换、压缩包查看和解压。
- PDF 信息读取、拆分、合并、压缩、OCR、页面渲染、文本或图片提取。
- 图片裁剪、缩放、水印、二维码、差异对比、缩略图。
- 音频/视频信息读取、转码、裁剪、提取音频、抽帧、波形或缩略图总览。
- CSV/Excel/JSON/SQLite 的轻量检查、转换、导出和只读查询。
- 文本处理、关键词、词云、OCR、HTML/图表/看板生成。
- 临时 Python/Lua 脚本执行和中间数据处理；Python 固定工具入口是 `/system/tools/runtime/python.form`。
- 站内消息或通知类基础动作。

## 什么时候新建业务 app

在目标工作台目录下新建或修改业务 app：

- 需要长期保存业务数据，例如预约、工单、域名、证书资产、销售流水。
- 需要业务规则、状态机、权限边界、负责人、通知人、幂等标记。
- 需要定时巡检、日报分析、Chart 趋势、多人协作入口。
- 需要把多个系统工具串成稳定业务闭环，并沉淀可复用目录。

## 设计原则

- `/system/tools` 是基础能力层，业务目录是场景解决方案层。
- 不要把 PDF、图片、音视频、CSV 转换这类通用能力复制到每个业务 app。
- 如果业务 app 需要调用通用处理能力，优先通过工作台/智能体编排复用系统工具；只有需要稳定 SDK 级集成时再写业务函数。
- 临时脚本优先用 `/system/tools/runtime/python.form`；需要固定字段、命名规则、响应结构、权限和业务语义时，再在业务 app 里用 Go 调 Python runtime。
- `/system/tools/runtime/python.form` 支持文件输入和输出：顶层 `input_files` 会下载为本地路径并注入 `args["input_files"]`，Python 返回 `output_files` 后平台负责上传并展示文件组件。
- 文档和案例里可以把 `/system/tools` 当作固定系统目录提及，但不要写成本机磁盘路径。

## 常见组合

- 合同归档：业务 app 保存合同台账；需要 OCR、拆页、压缩时复用 `/system/tools/pdf`。
- 会议内容处理：会议 app 管理预约和纪要；音频转码、提取音轨、波形图复用 `/system/tools/audio` 和 `/system/tools/video`。
- 数据日报：业务 app 保存经营数据；临时 CSV 体检、SQLite 查询、图表图片生成复用 `/system/tools/table`、`/system/tools/database`、`/system/tools/chart`。
- 内容情报：定时会话负责采集和总结；下载文件、HTML 预览、文本关键词分析可复用 `/system/tools/file`、`/system/tools/html`、`/system/tools/text`。
