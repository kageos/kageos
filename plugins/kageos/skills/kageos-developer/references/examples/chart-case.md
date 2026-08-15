# Chart 图表最佳实践案例

适用：用户要求基于业务表生成趋势图、分布图、排行榜、状态占比。优先使用 SDK chart 结构，不手写前端图表。

## 折线图模式

```go
type TrendReq struct {
	TimeRange string `json:"time_range" form:"time_range" widget:"name:时间范围;type:select;options:最近1天,最近7天,最近30天,自定义;render_default:最近1天"`
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Bucket    string `json:"bucket" form:"bucket" widget:"name:聚合粒度;type:select;options:自动,按分钟,按5分钟,按小时,按天,按月;render_default:自动"`
}

func TrendChart(ctx *app.Context, resp response.Response) error {
	var req TrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	q := db.Table("sample_item")
	start, end := resolveTrendWindow(req, time.Now())
	q = q.Where("created_at >= ? AND created_at <= ?", start, end)

	type stat struct {
		Day   string `json:"day"`
		Total int    `json:"total"`
	}
	var stats []stat
	decision := app.ResolveChartBucket(app.ChartBucketPolicy{
		// Requested 是前端选择的粒度；未选择或无法识别时返回 TimeBucketAuto。
		Requested: requestedTrendBucket(req.Bucket),
		// WindowStart/WindowEnd 必须和查询 Where 条件使用的时间窗口一致。
		WindowStart: start,
		WindowEnd:   end,
		// SeriesCount 是最终返回的系列数量。这里只有“新增数量”一条线，所以传 1。
		SeriesCount: 1,
		// MaxValues 可选；默认不填，表示不限制细粒度、不自动放粗。
	})
	// dateExpr 用于 Select 生成时间桶，groupExpr 用于 Group；created_at 必须是真实 datetime 字段。
	dateExpr, groupExpr := app.DateTimeBucketExpr(db, "created_at", decision.Bucket)
	if err := q.Select(dateExpr+" as day, COUNT(*) as total").Group(groupExpr).Order("day").Scan(&stats).Error; err != nil {
		return err
	}

	xAxis := make([]string, 0, len(stats))
	values := make([]interface{}, 0, len(stats))
	for _, row := range stats {
		xAxis = append(xAxis, row.Day)
		values = append(values, row.Total)
	}

	return resp.Chart(&chart.LineChart{
		Title: "每日新增趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "新增数量", Data: values},
		},
		Metadata: app.ChartBucketMetadata(decision),
	}).Build()
}

func resolveTrendWindow(req TrendReq, now time.Time) (time.Time, time.Time) {
	switch strings.TrimSpace(req.TimeRange) {
	case "最近7天":
		return now.AddDate(0, 0, -7), now
	case "最近30天":
		return now.AddDate(0, 0, -30), now
	case "自定义":
		start, _ := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local)
		end, _ := time.ParseInLocation("2006-01-02 15:04:05", req.EndTime, time.Local)
		if !start.IsZero() && !end.IsZero() && end.After(start) {
			return start, end
		}
	}
	return now.AddDate(0, 0, -1), now
}

func requestedTrendBucket(bucket string) app.TimeBucket {
	switch strings.TrimSpace(bucket) {
	case "按分钟":
		return app.TimeBucketMinute
	case "按5分钟":
		return app.TimeBucket5Minute
	case "按小时":
		return app.TimeBucketHour
	case "按天":
		return app.TimeBucketDay
	case "按月":
		return app.TimeBucketMonth
	default:
		return app.TimeBucketAuto
	}
}

var TrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "新增趋势",
		Request:  &TrendReq{},
		Response: &chart.LineChart{},
	},
	ChartType: app.ChartTypeLine,
}

func init() {
	packageContext.GET("trend.chart", TrendChart, TrendChartTemplate)
}
```

## 设计要求

- Chart Handler 是只读查询，不做写入副作用。
- `BaseConfig.Response` 填与 `resp.Chart(...)` 一致的 chart 包类型，例如 `&chart.LineChart{}`、`&chart.BarChart{}`。
- `ChartTemplate.ChartType` 使用 `app.ChartTypeLine`、`app.ChartTypeBar`、`app.ChartTypePie` 等常量。
- 时间聚合优先用 `app.ResolveChartBucket` + `app.DateTimeBucketExpr`，不要在业务代码里散落手写粒度判断。
- 波动型趋势图默认时间范围宜短，例如最近 1 天；前端可通过参数选择“自动、按分钟、按5分钟、按小时、按天、按月”。
- `ResolveChartBucket` 默认只推荐和估算，不硬拦细粒度；只有业务显式传 `MaxValues` 时才自动放粗，适合默认总览保护前端。
- `ChartBucketPolicy.Requested` 是前端请求粒度，`WindowStart/WindowEnd` 是真实查询窗口，`SeriesCount` 是返回系列数，`MaxValues` 是可选保护预算且默认不填。
- 返回图表时建议合并 `app.ChartBucketMetadata(decision)` 到 `Metadata`，便于查看实际粒度和估算点数。
- X 轴和 Series 的数据长度必须一致。
- 多系列图用多个 `chart.ChartSeries`，系列名称要能直接被业务用户理解。
- `YAxis` 是可选配置，不是所有图表都要加；只有需要覆盖默认数字展示时再声明。
- 需要声明 Y 轴数值格式时，不要把单位塞进 `Metadata` 让前端猜；使用 `YAxis: &chart.AxisConfig{ValueFormat: ...}`。
- 耗时趋势图的 `Series.Data` 保持毫秒原始值，例如 `1200` 表示 1200ms，然后按需设置 `ValueFormat: chart.ValueFormatDurationMS`，不要为了避开 K/M 缩写把后端数据改成秒。

```go
YAxis: &chart.AxisConfig{
	// ValueFormat 控制 Y 轴标签、十字准星标签和 tooltip 的数值显示。
	// 不填 YAxis 时保持默认展示，相当于 chart.ValueFormatCompact。
	// 可选值：
	// - chart.ValueFormatCompact：默认大数字缩写，如 1200 显示为 1.2K
	// - chart.ValueFormatPlain：普通数字，不做 K/M 缩写
	// - chart.ValueFormatDurationMS：数据原始单位是毫秒，前端显示为 ms/s/min
	// - chart.ValueFormatPercent：百分比数字，前端追加 %
	ValueFormat: chart.ValueFormatDurationMS,
},
```
