# Hub 目录包元数据契约

KageOS 导出的 `capability.bundle.v1` 首先是可安装目录包，其次才是 Hub 投稿输入。
安装内容和运营数据必须分离：KageOS 只导出能够从目录本身确定的事实，价格、图片、
营销文案和审核状态由 Hub 管理。

## 顶层结构

目录包继续使用 `capability.bundle.v1`，新增可选的 `metadata.directory`：

```json
{
  "schema_version": "capability.bundle.v1",
  "name": "黄金盯盘助手",
  "metadata": {
    "directory": {
      "code": "gold_watch",
      "name": "黄金盯盘助手",
      "description": "监控黄金价格并生成提醒。",
      "tags": ["黄金价格", "行情监控", "价格提醒"],
      "source_revision": "v12",
      "release_version": "0.1.0"
    }
  },
  "packages": [],
  "tree_nodes": [],
  "docs": [],
  "files": []
}
```

字段规则：

| 字段 | 来源 | 规则 |
|---|---|---|
| `code` | 导出根 package 的 `ServiceTree.Code` | 保留原始 Go package 标识，例如 `gold_watch`，不得转成 `gold-watch` |
| `name` | 导出根 package 的名称 | 目录本身的名称，不负责生成中英文翻译 |
| `description` | 导出根 package 的描述 | 目录事实描述，Hub 可以基于它补充投稿摘要 |
| `tags` | 导出根 package 的标签 | 规范化为去重字符串数组 |
| `source_revision` | KageOS 目录内部版本 | 用于追溯，不等同于 Hub 商品版本 |
| `release_version` | 可选语义版本 | 只有目录本身已有合法 SemVer 时才导出 |

## 身份规则

Hub 来源身份由下面两个值共同确定：

```text
投稿者账号 ID + metadata.directory.code
```

投稿者账号不写入目录包，避免把工作空间用户、应用路径或租户信息泄露到可分发 JSON。
不同投稿者可以提交相同的 `code`；同一投稿者再次提交相同 `code` 时进入同一目录的
新版本或新修订。

## Hub 解析优先级

Hub 按以下顺序补全投稿表单：

1. `metadata.directory` 中的确定性字段；
2. `tree_nodes`、`packages`、`docs` 中已有的节点元数据；
3. AI 只补充缺失的翻译、分类和面向读者的摘要；
4. 投稿者人工修改建议稿；
5. 官方审核后生成正式发布快照。

AI 不得生成或改写目录 `code`，也不得把下划线转换成中划线。

## 不进入目录包的字段

以下字段属于 Hub 运营数据，不写入 KageOS 导出 JSON：

- 投稿者账号和署名；
- 建议价格、官方售价和支付产品；
- 封面、截图、视频和详情页排版；
- 投稿授权、审核意见和状态；
- 上架、下架、订单、退款和售后数据。

这样同一个目录包可以在不依赖 Hub 的情况下继续安装、导入和迁移。
