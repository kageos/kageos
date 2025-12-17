/**
 * ExpressionParserV2 测试文件
 * 用于验证新解析引擎的功能
 */

import { ExpressionParserV2 } from './ExpressionParserV2'

// 测试数据
const testData = [
  { price: 10, quantity: 2, amount: 0, discount_rate: 0.9, '价格': 10 },
  { price: 20, quantity: 3, amount: 50, discount_rate: 0.8, '价格': 20 },
  { price: 15, quantity: 1, amount: 0, discount_rate: 0.7, '价格': 15 },
]

/**
 * 测试函数
 */
function test(name: string, expression: string, expected: number, data: any[] = testData) {
  try {
    const result = ExpressionParserV2.evaluate(expression, data)
    const passed = Math.abs(result - expected) < 0.01
    console.log(`${passed ? '✅' : '❌'} ${name}`)
    console.log(`   表达式: ${expression}`)
    console.log(`   期望: ${expected}, 实际: ${result}`)
    if (!passed) {
      console.error(`   ❌ 测试失败！`)
    }
    return passed
  } catch (error) {
    console.error(`❌ ${name} - 错误:`, error)
    return false
  }
}

/**
 * 运行所有测试
 */
export function runTests() {
  console.log('=== ExpressionParserV2 测试 ===\n')

  let passed = 0
  let total = 0

  // 测试1：基础数学表达式
  total++
  if (test('基础乘法', 'sum(price * quantity)', 10*2 + 20*3 + 15*1)) passed++

  total++
  if (test('中文字段名', 'sum(价格 * quantity)', 10*2 + 20*3 + 15*1)) passed++

  total++
  if (test('多因子乘法', 'sum(price * quantity * 0.9)', (10*2 + 20*3 + 15*1) * 0.9)) passed++

  total++
  if (test('括号表达式', 'sum(price * quantity * (1 - discount_rate))', 
    10*2*(1-0.9) + 20*3*(1-0.8) + 15*1*(1-0.7))) passed++

  // 测试2：COALESCE 函数
  total++
  if (test('COALESCE - 使用 amount', 'sum(COALESCE(amount, price * quantity))', 
    0 + 50 + 0)) passed++

  total++
  if (test('COALESCE - 回退到 price*quantity', 'sum(COALESCE(amount, price * quantity))', 
    10*2 + 50 + 15*1)) passed++

  // 测试3：IF 表达式
  total++
  if (test('IF - 简单条件', 'sum(IF amount > 0 THEN amount ELSE price * quantity)', 
    10*2 + 50 + 15*1)) passed++

  total++
  if (test('IF - 复杂条件', 'sum(IF amount > 0 AND amount < 100 THEN amount ELSE price * quantity)', 
    10*2 + 50 + 15*1)) passed++

  // 测试4：Count
  total++
  if (test('Count - 统计种类数', 'count(price)', 3)) passed++

  // 测试5：Avg
  total++
  if (test('Avg - 平均值', 'avg(price * quantity)', (10*2 + 20*3 + 15*1) / 3)) passed++

  console.log(`\n=== 测试结果: ${passed}/${total} 通过 ===`)
  
  return { passed, total }
}

// 如果直接运行此文件，执行测试
if (require.main === module) {
  runTests()
}

