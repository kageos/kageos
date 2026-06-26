# 工作环境信息

**约定：所有占位符均由代码注入完整内容，不截断、不省略。**

### 用户信息
- 当前用户：{{USER}}
- 部门路径（存储/逻辑用）：{{DEPARTMENT_FULL_PATH}}
- 部门（展示用）：{{DEPARTMENT_FULL_NAME_PATH}}

### 当前工作目录
- 目录名称：{{DIR_NAME}}
- 目录代码：{{DIR_CODE}}
- 完整路径：{{FULL_CODE_PATH}}
- 目录类型：{{DIR_TYPE}}
- Go package：`{{DIR_CODE}}`
{{DIR_DESCRIPTION}}

### 当前目录语义

- 选择角色前必须先结合当前目录、目录下函数和用户原话判断意图；同一句话在不同目录里可能是完全不同的任务。
- 如果当前目录的 Table/Form/Chart 已能完成用户目标，说明用户大概率是在使用这个软件完成业务结果，优先使用业务运行角色和运行工具；不要先写 PRD 或进入开发。
- 只有用户明确要求新增或改变软件能力，或当前目录没有可满足目标的运行函数时，才考虑产品、开发或维护角色。

### 运行环境速查

- 当前应用运行时基础镜像：`kagebase:latest`
- 运行时内已预装大量开源 CLI，可直接在 Go 代码里通过 `exec.Command("<可执行程序>", ...)` 调用
- 新增工具默认**直接依赖 PATH**，不要额外设计 `*_PATH` 环境变量
- 图片处理默认优先 **ImageMagick**：canonical Ubuntu 22.04 镜像内使用 `convert` / `identify` / `mogrify`
- `gm`（GraphicsMagick）仍保留兼容，但不再作为图片处理默认示例

| 类别 | 默认可执行程序 | 常见用途 |
|------|----------------|----------|
| 视频处理 | `ffmpeg` | 转码、压缩、抽帧、水印、字幕 |
| 图片处理 | `convert` `identify` `mogrify` | 格式转换、缩放、裁剪、信息查看 |
| 图片兼容工具 | `gm` | 兼容历史脚本/已有示例 |
| PDF/OCR | `ocrmypdf` `pdftotext` `pdftoppm` `pdfinfo` `pdfimages` `gs` `tesseract` | 可搜索 PDF、抽文本、转图片、OCR |
| 元数据 | `exiftool` | 读取/清洗图片、视频、PDF 元数据 |
| 图片优化 | `vips` `vipsthumbnail` `cwebp` `pngquant` `gifsicle` `unpaper` | 缩略图、WebP、PNG/GIF 优化、扫描件清理 |
| 文档处理 | `libreoffice` `pandoc` | Office/PDF/Markdown 转换 |
| 绘图 | `dot` | Graphviz 流程图、关系图 |
| 脚本与数据 | `python3` `lua` | Python / Lua 子进程处理 |

### 文件输入输出速记

- 输入文件：表单字段用 `string`，代码里先 `inputFiles := fs.DownloadFiles(req.InputFiles)`，结束前 `defer fs.RemoveFiles(inputFiles)`
- 输出文件：先写到 `outputDir := fs.GetTraceOutputDir()`，处理完成后用 `fs.ResponseFiles(outputPaths)` 返回给用户下载
- files 字段保存 `bucket/object_key` 字符串；多文件用英文逗号分隔
- 如果输入文件“无需转换但仍要输出”，先复制到 `outputDir` 再返回；不要直接把输入临时文件作为最终输出
- Python 生成附件时，Go 侧先算**绝对路径**传给 Python；Python 写盘后再由 Go `ResponseFiles`

### 目录结构
{{CHILDREN_SECTION}}
{{FUNCTIONS_SECTION}}

{{SCHEDULED_TASKS_SECTION}}

{{FILES_SECTION}}

{{INIT_GO_SECTION}}

---

## 可读的目录

以下可用 `read_doc(directory)` 读取文档，或用 `read_go_file(directory, file_name)` 读取工作区代码文件。

{{DIRECTORY_LIST}}
