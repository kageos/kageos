package statistics

import (
	"fmt"
	"strings"
)

// ================ 基础聚合函数 ================

// Sum 求和
//
// 支持的表达式（与 MySQL/SQL 一致，前端 ExpressionParserV2 解析）：
//
//   - 基础运算：
//     Sum("价格")                    → "sum(价格)"
//     Sum("价格 * quantity")         → "sum(价格 * quantity)"
//     Sum("单价 * 数量 * (1 - 折扣率)") → "sum(单价 * 数量 * (1 - 折扣率))"
//
//   - 条件表达式 IF(cond, thenExpr, elseExpr)：
//     Sum("IF(price > 0, price * quantity, 销售价 * quantity)")
//     Sum("IF(amount > 0, amount, 价格 * quantity)")
//
//   - 空值处理 COALESCE/IFNULL：
//     Sum("COALESCE(amount, 0) * quantity")
//     Sum("IFNULL(price, 默认价) * quantity")
//
//   - CASE WHEN 多分支：
//     Sum("CASE WHEN type=1 THEN amount ELSE 0 END")
//     Sum("CASE WHEN status='已支付' THEN 金额 ELSE 0 END")
//
// 参数说明：
//   - expression: 完整表达式，空格+星号表示乘，支持 IF、COALESCE、CASE WHEN 等
//
// 注意：使用空格格式，不要用逗号（如 "价格,*quantity" 已废弃）
func Sum(expression string) string {
	return fmt.Sprintf("sum(%s)", expression)
}

// Count 计数
//
// 支持的表达式（自动去重，等价于 COUNT(DISTINCT)）：
//   - Count("id")           → "count(id)"            // 统计唯一 id 数量
//   - Count("价格")         → "count(价格)"          // 统计选中商品种类数
//   - Count("product_id")   → "count(product_id)"    // 统计不同商品数
//
// 参数说明：
//   - field: 字段名或表达式（来自 DisplayInfo 的 key）
//
// 使用场景：
//   - 计算选中商品种类数：Count("价格")
//   - 统计不同供应商数：Count("supplier_id")
func Count(field string) string {
	return fmt.Sprintf("count(%s)", field)
}

// Avg 平均值
//
// 支持的表达式：
//   - Avg("价格")                  → "avg(价格)"
//   - Avg("价格 * quantity")        → "avg(价格 * quantity)"
//   - Avg("COALESCE(amount, 0)")   → "avg(COALESCE(amount, 0))"
//
// 参数说明：
//   - field: 主字段名或表达式（来自 DisplayInfo 的 key）
//   - multipliers: 可选的乘数字段或系数（自动添加 * 前缀）
//
// 使用场景：
//   - 计算平均价格：Avg("价格")
//   - 计算平均订单金额：Avg("金额")
func Avg(field string, multipliers ...string) string {
	if len(multipliers) == 0 {
		return fmt.Sprintf("avg(%s)", field)
	}

	args := []string{field}
	for _, m := range multipliers {
		args = append(args, "*"+m)
	}
	return fmt.Sprintf("avg(%s)", strings.Join(args, ","))
}

// Min 最小值
//
// 支持的表达式：
//   - Min("价格")       → "min(价格)"       // 最低价格
//   - Min("created_at") → "min(created_at)" // 最早创建时间
//   - Min("金额")       → "min(金额)"       // 最小订单金额
//
// 参数说明：
//   - field: 字段名（来自 DisplayInfo 的 key）
//
// 使用场景：
//   - 计算最低价格：Min("价格")
//   - 最早入库日期：Min("入库日期")
func Min(field string) string {
	return fmt.Sprintf("min(%s)", field)
}

// Max 最大值
//
// 支持的表达式：
//   - Max("价格")       → "max(价格)"       // 最高价格
//   - Max("amount")     → "max(amount)"     // 最大金额
//   - Max("created_at") → "max(created_at)" // 最晚创建时间
//
// 参数说明：
//   - field: 字段名（来自 DisplayInfo 的 key）
//
// 使用场景：
//   - 计算最高价格：Max("价格")
//   - 最大库存数量：Max("库存")
func Max(field string) string {
	return fmt.Sprintf("max(%s)", field)
}

// ================ List 层聚合函数 ================

// ListSum List 层求和（用于 MultiSelect 二层聚合）
//
// 支持的表达式：
//   - ListSum("用户总价") → "list_sum(用户总价)"
//
// 参数说明：
//   - field: 字段名（来自行的计算结果，如行内聚合的结果）
//
// 使用场景：
//   - 在 List 内 MultiSelect 场景中，对所有行的行内统计结果求和
//   - 例如：每行计算了"用户总价"，ListSum("用户总价") 对所有行的"用户总价"求和
func ListSum(field string) string {
	return fmt.Sprintf("list_sum(%s)", field)
}

// ListAvg List 层平均值
//
// 支持的表达式：
//   - ListAvg("用户总价") → "list_avg(用户总价)"
//
// 参数说明：
//   - field: 字段名（来自行的计算结果）
//
// 使用场景：
//   - 计算所有行的平均值
func ListAvg(field string) string {
	return fmt.Sprintf("list_avg(%s)", field)
}

// ListCount List 层计数
//
// 支持的表达式：
//   - ListCount() → "list_count()"
//
// 使用场景：
//   - 计算 List 中有多少行
func ListCount() string {
	return "list_count()"
}

// ================ 选中项字段值函数 ================

// Value 显示选中项的字段值（动态值）
//
// 支持的表达式：
//   - Value("余额") → "value(余额)"
//   - Value("折扣") → "value(折扣)"
//
// 参数说明：
//   - field: 字段名（来自 DisplayInfo 的 key）
//
// 使用场景：
//   - 单选场景：显示当前选中项的某个字段值
//   - 例如：选中会员卡后，显示该会员卡的"余额"、"折扣"等字段值
//   - 前端会从选中项的 DisplayInfo 中获取对应的值并显示
//
// 注意：
//   - 仅适用于单选场景（SelectWidget）
//   - 多选场景（MultiSelectWidget）会显示第一个选中项的值
//   - 返回值可以是任何类型（文本、数字、日期等）
func Value(field string) string {
	return fmt.Sprintf("value(%s)", field)
}
