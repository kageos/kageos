# 修改类型：新增或修改 select 选项

静态 `select` / `multiselect` 修改 `options` 时，同步检查 `options_colors` 数量。颜色只用不带 `#` 的 6 位十六进制 `RRGGBB`。同时检查 validate 的 `oneof` 是否需要更新。
