# 修改类型：修改 Chart 指标

## 硬规则

1. 一个 Chart 路由只返回一张图。
2. 不要写 `resp.Charts(...)`，SDK 没有这个 API。
3. 不要写 `resp.Chart(chart1, chart2)`，`resp.Chart(chart).Build()` 只接收一个 SDK chart 对象。
4. 多张图拆成多个 `.chart` 路由、多个 Handler、多个 ChartTemplate。
5. 总数、占比、NPS 值、平均值等汇总指标放到 `Metadata`，不要为了返回多个指标编造多图响应。

## 图表对象

只使用 SDK chart 包里的具体类型：`chart.LineChart`、`chart.BarChart`、`chart.PieChart`、`chart.GaugeChart`。

ChartTemplate 必须和返回图表一致：

- 折线图：`Response: &chart.LineChart{}` + `ChartType: app.ChartTypeLine`
- 柱状图：`Response: &chart.BarChart{}` + `ChartType: app.ChartTypeBar`
- 饼图：`Response: &chart.PieChart{}` + `ChartType: app.ChartTypePie`
- 仪表盘：`Response: &chart.GaugeChart{}` + `ChartType: app.ChartTypeGauge`

## 时间分组

时间分组优先使用 SDK helper，不要自己写死 MySQL/SQLite 语法。

```go
dateExpr, groupExpr := app.DateTimeBucketExpr(db, "created_at", app.TimeBucketDay)
err := queryDB.
    Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateExpr)).
    Group(groupExpr).
    Scan(&stats).Error
```

`app.DateTimeBucketExpr` 返回两个值：`dateExpr` 用于 `Select`，`groupExpr` 用于 `Group`。`Group` 只传一个字符串参数。

## GORM 统计

GORM `Count` 必须传 `*int64`：

```go
var total int64
if err := queryDB.Count(&total).Error; err != nil {
    return err
}
```

需要传给只接收 `int` 的业务函数时，再显式转换：`int(total)`。

`types.Time` 格式化或比较要先取内部时间：`t.Time().Format(...)`、`t.Time().After(...)`。
