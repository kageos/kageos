# 默认定时函数最佳实践案例

适用：用户要求“每天生成日报”“每 5 分钟巡检”“自动提醒”“自动续期”等固定 Form 能力。kageos 的默认定时函数通过 `app.FormTemplate.Schedules` 声明；运行时状态由调度服务维护，业务代码只负责一次执行的逻辑。

## 核心模式

```go
type SweepReq struct {
	WindowMinutes int `json:"window_minutes" widget:"name:扫描窗口;type:integer;default:30"`
}

type SweepResp struct {
	Summary string `json:"summary" widget:"name:执行摘要;type:textarea"`
}

func Sweep(ctx *app.Context, resp response.Response) error {
	var req SweepReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	if req.WindowMinutes <= 0 {
		req.WindowMinutes = 30
	}

	// 在这里读取业务表、执行巡检、发送通知、写入执行标记。
	return resp.Form(&SweepResp{Summary: "本轮巡检完成"}).Build()
}

func init() {
	packageContext.POST("sweep.form", Sweep, &app.FormTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "自动巡检",
			Desc:     "按固定频率扫描业务数据并输出执行摘要。",
			Tags:     []string{"定时任务", "巡检"},
			Request:  &SweepReq{},
			Response: &SweepResp{},
		},
		Schedules: []app.FormSchedule{
			{
				Code:        "sample_sweep_every_30m",
				Title:       "每 30 分钟自动巡检",
				Description: "周期性扫描最近 30 分钟需要处理的数据。",
				Enabled:     true,
				CronExpr:    "*/30 * * * *",
				Timezone:    "Asia/Shanghai",
				Body:        SweepReq{WindowMinutes: 30},
			},
		},
	})
}
```

## 设计要求

- `Code` 必须稳定且唯一；后续改名会影响幂等识别。
- `CronExpr` 和 `EverySeconds` 只能二选一。
- `Enabled` 是 bool 值，不要用指针；默认要开启的定时函数写 `Enabled: true`，不想安装后运行才写 `false`。不要让 AI 依赖“未填写时的默认值”猜行为。
- `Timezone` 要显式设置，面向国内业务通常用 `Asia/Shanghai`，避免部署环境时区影响触发时间。
- `Body` 必须能序列化成 JSON object，优先传请求结构体。
- 定时函数要幂等：重复执行不能重复发消息、重复扣减、重复创建同一业务记录。
- 执行结果要有简洁 `Summary`，失败时返回明确错误，方便执行记录排查。
- 默认调度声明后必须执行 build/update，平台才会创建或更新任务；只改本地文件不会让工作台出现新定时任务。
- 默认调度适合“随应用发布自动出现”的固定任务；动态由用户创建的复杂计划不要硬编码进 `Schedules`。
