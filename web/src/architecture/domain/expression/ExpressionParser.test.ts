import { describe, expect, it } from 'vitest'

import { ExpressionParser } from './ExpressionParser'

const rows = [
  { price: 10, quantity: 2, amount: 0, discount_rate: 0.9, '价格': 10 },
  { price: 20, quantity: 3, amount: 50, discount_rate: 0.8, '价格': 20 },
  { price: 15, quantity: 1, amount: 0, discount_rate: 0.7, '价格': 15 },
]

function expectExpression(expression: string, expected: number) {
  expect(ExpressionParser.evaluate(expression, rows)).toBeCloseTo(expected, 2)
}

describe('ExpressionParser', () => {
  it('evaluates arithmetic expressions with english and chinese field names', () => {
    expectExpression('sum(price * quantity)', 10 * 2 + 20 * 3 + 15 * 1)
    expectExpression('sum(价格 * quantity)', 10 * 2 + 20 * 3 + 15 * 1)
    expectExpression('sum(price * quantity * 0.9)', (10 * 2 + 20 * 3 + 15 * 1) * 0.9)
  })

  it('evaluates grouped expressions and aggregate functions', () => {
    expectExpression(
      'sum(price * quantity * (1 - discount_rate))',
      10 * 2 * (1 - 0.9) + 20 * 3 * (1 - 0.8) + 15 * 1 * (1 - 0.7),
    )
    expectExpression('count(price)', 3)
    expectExpression('avg(price * quantity)', (10 * 2 + 20 * 3 + 15 * 1) / 3)
  })

  it('evaluates IF and COALESCE functions', () => {
    expectExpression('sum(COALESCE(amount, price * quantity))', 0 + 50 + 0)
    expectExpression('sum(IF(amount > 0, amount, price * quantity))', 10 * 2 + 50 + 15 * 1)
    expectExpression('sum(IF(price > 0, price * quantity, 价格 * quantity))', 10 * 2 + 20 * 3 + 15 * 1)
  })
})
