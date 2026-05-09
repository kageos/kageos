# 发布和复用

## 使用条件

用户要求搜索、复制、发布或推送更新。

## 流程

1. 用 `search_tools` 搜索对应 `/system/openapi` 函数，并按函数 schema 搜索或确认目标目录。
3. 说明版本、目标远程地址和影响范围。
4. 通过平台 OpenAPI 函数执行复制、发布或推送。
5. 必要时验证复制后的关键函数。

## 按需参考

- 需要在 SDK 代码里调用平台 Web API：`read_doc("/system/prompt/sdk/reference/platform-api")`

## 注意

复用优先于重复创建。发布和推送属于副作用，必须确认目标和影响范围。
