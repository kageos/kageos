# 修改类型：修改搜索条件

Table 筛选字段写在 Request 中，并嵌入 `query.PageSortReq`。范围筛选使用明确字段名，例如 `created_start` / `created_end`，并在 Handler 中手写 `>=` / `<=` 条件。排序由 `PageSortReq` 和 `Table` 自动承接，业务 Handler 不手写排序透传。
