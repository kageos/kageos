# SDK 构建与启动校验参考

## 链路

```text
build_workspace
  -> app-server update app
  -> app-runtime go build
  -> 启动新版本
  -> app.Run()
  -> CompileAndValidate()
  -> startup running / startup failed
```

`build_workspace` 不写文件，只触发当前工作空间编译部署。

## CompileAndValidate 检查什么

1. App 是否初始化。
2. 路由是否为空。
3. handler 是否为空。
4. Template 是否为空。
5. 路由后缀和 Template 类型是否匹配。
6. `getApis()` 是否能解析全部 schema。
7. widget validator 是否能通过。
8. `functionschema.Validate()` 是否能通过。

多个错误会聚合返回，模型应一次性阅读完整错误并批量修复。

## widget validator 检查什么

- widget type 是否支持。
- widget tag key 是否在白名单中。
- `depend_on` 是否引用同级字段。
- 字段筛选配置是否匹配 Go 类型。
- 组件自己的 Go 类型、必填配置和范围参数是否正确。
- 每个 supported widget 都必须注册 validator。

## 修复顺序

1. Go 编译错误：先修 import、类型、变量、语法。
2. SDK schema 错误：修 route、Template、widget、validate、筛选字段、Response 类型。
3. 源码规范错误：先按安全规则修复，不要试图绕过 `write_go_file` / build 前校验。
4. Runtime 启动错误：查启动日志和 lifecycle failed message。
5. 业务执行错误：build 通过后再用 run 工具验证。

## 构建错误处理原则

不要把 build 错误当作继续猜 API 的入口。先按错误类别回到已经读取过的 SDK 文档、案例或源码确认真实写法，再一次性批量修复。

- `undefined: <sdk package>.<symbol>`：代码使用了未确认的 SDK API。读取对应知识点文档或 SDK 源码，确认真实导出符号后再改；不要按命名直觉换另一个猜测的符号。
- `<type> has no field or method <method>`：代码使用了不存在的方法。读取该类型定义和方法集后再改。
- schema validation failed：先看 router、字段名、widget、validate、筛选字段、Request/Model 是否冲突，不要只修第一条错误。

## 高频错误速查

- `X redeclared in this block`：同一个 package 多个文件重复定义了同名 Model，或 Handler 函数名和 Model 类型名冲突。保留一个 Model 定义，Handler 改名为 `XHandler` / `XSubmit`。
- `req.GetPage undefined` / `unknown field Total/DataList`：列表分页使用 `query.PageSortReq`；Handler 显式调用 `GetOrder/GetOffset/GetLimit` 查询 `rows` 和 `total`，再 `resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()`。
- `types.Time has no field or method Format` 或 `Time.Format undefined`：使用 `t.Time().Format(...)`、`t.Time().After(...)`、`t.Time().Before(...)`。
- `unsupported widget type` / `unsupported widget tag` / `invalid tag format`：widget 只能使用 SDK 主文档组件速查和运行时白名单中确认过的类型和 key。文件上传是 `type:files`，只读展示用 `hide:"create,update"` 或 `widget:"-"`，不要编造前端习惯参数。图片/视频 files 字段需要列表缩略图时，只使用已支持的 `thumbnail:true;list_preview:true`。
- `widget "select" requires options or OnSelectFuzzyMap entry`：`select/multiselect` 必须有选项来源。简单固定枚举写静态 `options`；需要从表或接口查询时，字段写 `callback:"OnSelectFuzzy"` 并在对应 Template 的 `OnSelectFuzzyMap` 注册；纯展示名称不要写成 select，改用 input 或补真实回调。
- `integer widget requires integer Go type`：`type:integer` 必须配 Go 整数类型；`float64` 的金额、均值、评分、比例用 `type:float`。`type:number` 已废弃，出现时直接改成 `integer` 或 `float`。
- `cannot use &x (value of type *int) as *int64 value ... Count`：GORM `Count` 必须传 `*int64`。写 `var total int64; db.Count(&total)`；需要传给业务函数时再 `int(total)`。
- `assignment mismatch ... DateTimeBucketExpr returns 2 values` / `Group` 参数过多：`app.DateTimeBucketExpr` 返回两个表达式。写 `dateExpr, groupExpr := app.DateTimeBucketExpr(db, "created_at", app.TimeBucketDay)`；`Select` 用 `dateExpr`，`Group` 只传 `groupExpr`。
- `源码规范校验失败` / `应用数据库对象`：`ctx.GetGormDB()` 得到的 `db` 不能传给第三方库、外部 package、全局变量、struct 字段或 return；也不能调用 `Raw`、`Exec`、`Unscoped`、`Migrator`、`DB`、`AutoMigrate`。改为在当前文件/当前目录内直接写 GORM 链式查询、插入、更新和软删除；建表/加字段交给 Template `CreateTables`。
- `req.X undefined ... Req has no field or method X`：删除 Request 字段后，Handler 里的手写 `req.X` 筛选也要同步删除；需要筛选时把字段显式放回 Request。
- `does not implement chart.Charter`：传给 `resp.Chart(...)` 的必须是 SDK chart 包里的具体图表对象，不是自定义响应结构体；附加指标放 `Metadata`，多图拆多路由。
- `resp.Charts undefined`：SDK 没有 `resp.Charts`。一个 Chart 路由只返回一张图，使用 `resp.Chart(chart).Build()`；多图拆成多个 `.chart` 路由。
- `table request field ... conflicts with table model field ...`：Request 自定义筛选字段和 Model 字段 code 重复。`gorm:"-"` 计算/列表展示字段也算 Model 字段，不能和 Request 重名；需要筛选时使用不冲突的 Request 字段，并在 Handler 中手写 Where。
- `OnSelectFuzzyMap field ... must use select or multiselect widget`：回调 key 指向的字段不是 `type:select` / `type:multiselect`，或字段已从 Request 移除。把外键字段建模成 select，或删除该回调配置。
- `options_colors length must match options length` / `contains invalid color`：颜色数量必须和选项数量一致；颜色只用 6 位十六进制 `RRGGBB`，不带 `#`，不要用语义色或 `rgb(...)`。
- `audit field "id"/"created_at"/"updated_at"/"created_by"/"updated_by"`：系统字段必须使用规定的 widget、hide 和 gorm tag；不要省略这些 tag，也不要把 `DeletedAt` 暴露到 schema。`created_by/updated_by` 必须是 `type:user`、`hide:"create,update"`，且 `gorm:"column:created_by"` / `gorm:"column:updated_by"` 与 json 名一致。
