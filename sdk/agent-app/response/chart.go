package response

type Chart interface {
	Builder
}

// Chart 方法返回 Chart 接口，允许链式调用
// 例如：resp.Chart(chart).Build()

