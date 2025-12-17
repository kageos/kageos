/**
 * ExpressionParserV2 - 新一代聚合计算表达式解析引擎
 * 
 * 支持的新语法：
 * 1. 自然数学表达式：价格 * quantity, price * quantity * 0.9
 * 2. SQL 风格 IF：IF amount > 0 THEN amount * quantity ELSE price * quantity
 * 3. COALESCE 函数：COALESCE(amount, price * quantity)
 * 4. 括号支持：(1 - 折扣率), (price + tax) * quantity
 * 5. 操作符优先级：*, / 优先于 +, -
 * 
 * 示例：
 * - sum(价格 * quantity)
 * - sum(价格 * quantity * (1 - 折扣率))
 * - sum(COALESCE(amount, price * quantity))
 * - sum(IF amount > 0 THEN amount * quantity ELSE price * quantity)
 * - sum(IF amount > 0 AND amount < 1000 THEN amount ELSE price * quantity)
 * 
 * 设计原则：
 * 1. 保持向后兼容（旧解析器继续工作）
 * 2. 新解析器独立运行，可以逐步替换
 * 3. 支持自然数学表达式，更易读易写
 * 4. 支持复杂条件判断，解决业务痛点
 */

import { Logger } from './logger'

// ================ 类型定义 ================

/**
 * Token 类型
 */
enum TokenType {
  // 字面量
  NUMBER = 'NUMBER',           // 数字：0.9, 100
  FIELD = 'FIELD',             // 字段名：price, quantity, "价格"
  STRING = 'STRING',           // 字符串字面量："价格"
  
  // 操作符
  MULTIPLY = 'MULTIPLY',       // *
  DIVIDE = 'DIVIDE',           // /
  ADD = 'ADD',                 // +
  SUBTRACT = 'SUBTRACT',       // -
  
  // 比较操作符
  GT = 'GT',                   // >
  GTE = 'GTE',                 // >=
  LT = 'LT',                   // <
  LTE = 'LTE',                 // <=
  EQ = 'EQ',                   // ==
  NE = 'NE',                   // !=
  
  // 逻辑操作符
  AND = 'AND',                 // AND
  OR = 'OR',                   // OR
  NOT = 'NOT',                 // NOT
  
  // 括号
  LPAREN = 'LPAREN',           // (
  RPAREN = 'RPAREN',           // )
  
  // 关键字
  IF = 'IF',                   // IF
  THEN = 'THEN',               // THEN
  ELSE = 'ELSE',               // ELSE
  CASE = 'CASE',               // CASE
  WHEN = 'WHEN',               // WHEN
  END = 'END',                 // END
  
  // 函数
  COALESCE = 'COALESCE',       // COALESCE
  IFNULL = 'IFNULL',           // IFNULL
  
  // 聚合函数
  SUM = 'SUM',                 // sum
  COUNT = 'COUNT',             // count
  AVG = 'AVG',                 // avg
  MIN = 'MIN',                 // min
  MAX = 'MAX',                 // max
  LIST_SUM = 'LIST_SUM',       // list_sum
  LIST_AVG = 'LIST_AVG',       // list_avg
  LIST_COUNT = 'LIST_COUNT',   // list_count
  VALUE = 'VALUE',             // value
  
  // 分隔符
  COMMA = 'COMMA',             // ,
  
  // 结束
  EOF = 'EOF',                 // 结束
}

/**
 * Token
 */
interface Token {
  type: TokenType
  value: string | number
  position: number
}

/**
 * AST 节点类型
 */
type ASTNode =
  | NumberNode
  | FieldNode
  | BinaryOpNode
  | UnaryOpNode
  | FunctionCallNode
  | IfNode
  | CaseNode

interface NumberNode {
  type: 'Number'
  value: number
}

interface FieldNode {
  type: 'Field'
  name: string
}

interface BinaryOpNode {
  type: 'BinaryOp'
  operator: '*' | '/' | '+' | '-' | '>' | '>=' | '<' | '<=' | '==' | '!=' | 'AND' | 'OR'
  left: ASTNode
  right: ASTNode
}

interface UnaryOpNode {
  type: 'UnaryOp'
  operator: 'NOT' | '-'
  operand: ASTNode
}

interface FunctionCallNode {
  type: 'FunctionCall'
  name: string
  args: ASTNode[]
}

interface IfNode {
  type: 'If'
  condition: ASTNode
  thenBranch: ASTNode
  elseBranch: ASTNode | null
}

interface CaseNode {
  type: 'Case'
  cases: Array<{ condition: ASTNode; value: ASTNode }>
  elseValue: ASTNode | null
}

// ================ 词法分析器 ================

/**
 * Lexer - 词法分析器
 * 将表达式字符串转换为 Token 序列
 */
class Lexer {
  private input: string
  private position: number = 0
  private currentChar: string | null = null

  constructor(input: string) {
    this.input = input.trim()
    this.currentChar = this.input[this.position] || null
  }

  /**
   * 读取下一个字符
   */
  private advance(): void {
    this.position++
    this.currentChar = this.position < this.input.length ? this.input[this.position] : null
  }

  /**
   * 跳过空白字符
   */
  private skipWhitespace(): void {
    while (this.currentChar && /\s/.test(this.currentChar)) {
      this.advance()
    }
  }

  /**
   * 读取数字
   */
  private readNumber(): number {
    let result = ''
    while (this.currentChar && /[\d.]/.test(this.currentChar)) {
      result += this.currentChar
      this.advance()
    }
    return parseFloat(result)
  }

  /**
   * 读取字段名或关键字
   */
  private readIdentifier(): string {
    let result = ''
    
    // 支持中文字段名（带引号）
    if (this.currentChar === '"' || this.currentChar === "'") {
      const quote = this.currentChar
      this.advance() // 跳过引号
      while (this.currentChar && this.currentChar !== quote) {
        result += this.currentChar
        this.advance()
      }
      if (this.currentChar === quote) {
        this.advance() // 跳过结束引号
      }
      return result
    }
    
    // 读取标识符（字母、数字、下划线、中文字符）
    while (this.currentChar && /[\w\u4e00-\u9fa5]/.test(this.currentChar)) {
      result += this.currentChar
      this.advance()
    }
    return result
  }

  /**
   * 读取下一个 Token
   */
  nextToken(): Token {
    while (this.currentChar) {
      // 跳过空白字符
      if (/\s/.test(this.currentChar)) {
        this.skipWhitespace()
        continue
      }

      const startPos = this.position

      // 数字
      if (/\d/.test(this.currentChar)) {
        const value = this.readNumber()
        return { type: TokenType.NUMBER, value, position: startPos }
      }

      // 字符串字面量（带引号）
      if (this.currentChar === '"' || this.currentChar === "'") {
        const value = this.readIdentifier()
        return { type: TokenType.STRING, value, position: startPos }
      }

      // 操作符和符号
      switch (this.currentChar) {
        case '*':
          this.advance()
          return { type: TokenType.MULTIPLY, value: '*', position: startPos }
        case '/':
          this.advance()
          return { type: TokenType.DIVIDE, value: '/', position: startPos }
        case '+':
          this.advance()
          return { type: TokenType.ADD, value: '+', position: startPos }
        case '-':
          this.advance()
          return { type: TokenType.SUBTRACT, value: '-', position: startPos }
        case '(':
          this.advance()
          return { type: TokenType.LPAREN, value: '(', position: startPos }
        case ')':
          this.advance()
          return { type: TokenType.RPAREN, value: ')', position: startPos }
        case ',':
          this.advance()
          return { type: TokenType.COMMA, value: ',', position: startPos }
        case '>':
          this.advance()
          if (this.currentChar === '=') {
            this.advance()
            return { type: TokenType.GTE, value: '>=', position: startPos }
          }
          return { type: TokenType.GT, value: '>', position: startPos }
        case '<':
          this.advance()
          if (this.currentChar === '=') {
            this.advance()
            return { type: TokenType.LTE, value: '<=', position: startPos }
          }
          return { type: TokenType.LT, value: '<', position: startPos }
        case '=':
          this.advance()
          if (this.currentChar === '=') {
            this.advance()
            return { type: TokenType.EQ, value: '==', position: startPos }
          }
          return { type: TokenType.EQ, value: '=', position: startPos }
        case '!':
          this.advance()
          if (this.currentChar === '=') {
            this.advance()
            return { type: TokenType.NE, value: '!=', position: startPos }
          }
          // ! 作为 NOT 操作符（如果后面是标识符）
          break
      }

      // 标识符或关键字
      if (/[a-zA-Z\u4e00-\u9fa5_]/.test(this.currentChar)) {
        const identifier = this.readIdentifier()
        const upperIdentifier = identifier.toUpperCase()
        
        // 检查是否是关键字
        const keywordMap: Record<string, TokenType> = {
          'IF': TokenType.IF,
          'THEN': TokenType.THEN,
          'ELSE': TokenType.ELSE,
          'CASE': TokenType.CASE,
          'WHEN': TokenType.WHEN,
          'END': TokenType.END,
          'AND': TokenType.AND,
          'OR': TokenType.OR,
          'NOT': TokenType.NOT,
          'COALESCE': TokenType.COALESCE,
          'IFNULL': TokenType.IFNULL,
          'SUM': TokenType.SUM,
          'COUNT': TokenType.COUNT,
          'AVG': TokenType.AVG,
          'MIN': TokenType.MIN,
          'MAX': TokenType.MAX,
          'LIST_SUM': TokenType.LIST_SUM,
          'LIST_AVG': TokenType.LIST_AVG,
          'LIST_COUNT': TokenType.LIST_COUNT,
          'VALUE': TokenType.VALUE,
        }

        if (keywordMap[upperIdentifier]) {
          return { type: keywordMap[upperIdentifier], value: identifier, position: startPos }
        }

        // 普通字段名
        return { type: TokenType.FIELD, value: identifier, position: startPos }
      }

      // 未知字符
      throw new Error(`Unexpected character: ${this.currentChar} at position ${this.position}`)
    }

    // 结束
    return { type: TokenType.EOF, value: '', position: this.position }
  }

  /**
   * 获取所有 Token
   */
  tokenize(): Token[] {
    const tokens: Token[] = []
    let token = this.nextToken()
    while (token.type !== TokenType.EOF) {
      tokens.push(token)
      token = this.nextToken()
    }
    tokens.push(token) // 添加 EOF
    return tokens
  }
}

// ================ 语法分析器 ================

/**
 * Parser - 语法分析器
 * 将 Token 序列转换为 AST
 */
class Parser {
  private tokens: Token[]
  private position: number = 0

  constructor(tokens: Token[]) {
    this.tokens = tokens
  }

  /**
   * 获取当前 Token
   */
  private currentToken(): Token {
    return this.tokens[this.position] || { type: TokenType.EOF, value: '', position: 0 }
  }

  /**
   * 移动到下一个 Token
   */
  private advance(): void {
    if (this.position < this.tokens.length - 1) {
      this.position++
    }
  }

  /**
   * 检查当前 Token 类型
   */
  private check(type: TokenType): boolean {
    return this.currentToken().type === type
  }

  /**
   * 期望特定 Token 类型
   */
  private expect(type: TokenType): Token {
    const token = this.currentToken()
    if (token.type !== type) {
      throw new Error(`Expected ${type}, got ${token.type} at position ${token.position}`)
    }
    this.advance()
    return token
  }

  /**
   * 解析表达式（入口）
   */
  parse(): ASTNode {
    const node = this.parseExpression()
    if (!this.check(TokenType.EOF)) {
      throw new Error(`Unexpected token: ${this.currentToken().type}`)
    }
    return node
  }

  /**
   * 解析表达式（支持操作符优先级）
   */
  private parseExpression(): ASTNode {
    return this.parseLogicalOr()
  }

  /**
   * 解析逻辑或（OR）
   */
  private parseLogicalOr(): ASTNode {
    let left = this.parseLogicalAnd()
    
    while (this.check(TokenType.OR)) {
      this.advance()
      const right = this.parseLogicalAnd()
      left = { type: 'BinaryOp', operator: 'OR', left, right } as BinaryOpNode
    }
    
    return left
  }

  /**
   * 解析逻辑与（AND）
   */
  private parseLogicalAnd(): ASTNode {
    let left = this.parseComparison()
    
    while (this.check(TokenType.AND)) {
      this.advance()
      const right = this.parseComparison()
      left = { type: 'BinaryOp', operator: 'AND', left, right } as BinaryOpNode
    }
    
    return left
  }

  /**
   * 解析比较表达式（>, >=, <, <=, ==, !=）
   */
  private parseComparison(): ASTNode {
    let left = this.parseAdditive()
    
    while ([TokenType.GT, TokenType.GTE, TokenType.LT, TokenType.LTE, TokenType.EQ, TokenType.NE].includes(this.currentToken().type)) {
      const token = this.currentToken()
      this.advance()
      const right = this.parseAdditive()
      
      const operatorMap: Record<TokenType, string> = {
        [TokenType.GT]: '>',
        [TokenType.GTE]: '>=',
        [TokenType.LT]: '<',
        [TokenType.LTE]: '<=',
        [TokenType.EQ]: '==',
        [TokenType.NE]: '!=',
      }
      
      left = { type: 'BinaryOp', operator: operatorMap[token.type] as any, left, right } as BinaryOpNode
    }
    
    return left
  }

  /**
   * 解析加减表达式（+, -）
   */
  private parseAdditive(): ASTNode {
    let left = this.parseMultiplicative()
    
    while (this.check(TokenType.ADD) || this.check(TokenType.SUBTRACT)) {
      const token = this.currentToken()
      this.advance()
      const right = this.parseMultiplicative()
      left = { type: 'BinaryOp', operator: token.type === TokenType.ADD ? '+' : '-', left, right } as BinaryOpNode
    }
    
    return left
  }

  /**
   * 解析乘除表达式（*, /）
   */
  private parseMultiplicative(): ASTNode {
    let left = this.parseUnary()
    
    while (this.check(TokenType.MULTIPLY) || this.check(TokenType.DIVIDE)) {
      const token = this.currentToken()
      this.advance()
      const right = this.parseUnary()
      left = { type: 'BinaryOp', operator: token.type === TokenType.MULTIPLY ? '*' : '/', left, right } as BinaryOpNode
    }
    
    return left
  }

  /**
   * 解析一元表达式（NOT, -）
   */
  private parseUnary(): ASTNode {
    if (this.check(TokenType.NOT)) {
      this.advance()
      const operand = this.parseUnary()
      return { type: 'UnaryOp', operator: 'NOT', operand } as UnaryOpNode
    }
    
    if (this.check(TokenType.SUBTRACT)) {
      this.advance()
      const operand = this.parseUnary()
      return { type: 'UnaryOp', operator: '-', operand } as UnaryOpNode
    }
    
    return this.parsePrimary()
  }

  /**
   * 解析基础表达式（字面量、字段、函数调用、括号、IF、CASE）
   */
  private parsePrimary(): ASTNode {
    const token = this.currentToken()
    
    // 数字
    if (this.check(TokenType.NUMBER)) {
      this.advance()
      return { type: 'Number', value: token.value as number } as NumberNode
    }
    
    // 字段名
    if (this.check(TokenType.FIELD) || this.check(TokenType.STRING)) {
      this.advance()
      return { type: 'Field', name: token.value as string } as FieldNode
    }
    
    // IF 表达式
    if (this.check(TokenType.IF)) {
      return this.parseIf()
    }
    
    // CASE 表达式
    if (this.check(TokenType.CASE)) {
      return this.parseCase()
    }
    
    // 函数调用（注意：SUM、COUNT 等聚合函数在外层调用，这里只解析参数中的函数）
    if ([TokenType.COALESCE, TokenType.IFNULL].includes(token.type)) {
      return this.parseFunctionCall()
    }
    
    // 括号
    if (this.check(TokenType.LPAREN)) {
      this.advance()
      const expr = this.parseExpression()
      this.expect(TokenType.RPAREN)
      return expr
    }
    
    throw new Error(`Unexpected token: ${token.type} at position ${token.position}`)
  }

  /**
   * 解析 IF 表达式
   * IF condition THEN value1 ELSE value2
   */
  private parseIf(): ASTNode {
    this.expect(TokenType.IF)
    const condition = this.parseExpression()
    this.expect(TokenType.THEN)
    const thenBranch = this.parseExpression()
    
    let elseBranch: ASTNode | null = null
    if (this.check(TokenType.ELSE)) {
      this.advance()
      // 支持 ELSE IF
      if (this.check(TokenType.IF)) {
        elseBranch = this.parseIf()
      } else {
        elseBranch = this.parseExpression()
      }
    }
    
    return { type: 'If', condition, thenBranch, elseBranch } as IfNode
  }

  /**
   * 解析 CASE 表达式
   * CASE WHEN condition1 THEN value1 WHEN condition2 THEN value2 ELSE value3 END
   */
  private parseCase(): ASTNode {
    this.expect(TokenType.CASE)
    const cases: Array<{ condition: ASTNode; value: ASTNode }> = []
    
    while (this.check(TokenType.WHEN)) {
      this.advance()
      const condition = this.parseExpression()
      this.expect(TokenType.THEN)
      const value = this.parseExpression()
      cases.push({ condition, value })
    }
    
    let elseValue: ASTNode | null = null
    if (this.check(TokenType.ELSE)) {
      this.advance()
      elseValue = this.parseExpression()
    }
    
    this.expect(TokenType.END)
    return { type: 'Case', cases, elseValue } as CaseNode
  }

  /**
   * 解析函数调用
   * FUNCTION_NAME(arg1, arg2, ...)
   */
  private parseFunctionCall(): ASTNode {
    const token = this.currentToken()
    const funcName = token.value as string
    this.advance()
    
    this.expect(TokenType.LPAREN)
    const args: ASTNode[] = []
    
    if (!this.check(TokenType.RPAREN)) {
      args.push(this.parseExpression())
      while (this.check(TokenType.COMMA)) {
        this.advance()
        args.push(this.parseExpression())
      }
    }
    
    this.expect(TokenType.RPAREN)
    return { type: 'FunctionCall', name: funcName, args } as FunctionCallNode
  }
}

// ================ 表达式计算器 ================

/**
 * Evaluator - 表达式计算器
 * 执行 AST，计算表达式结果
 */
class Evaluator {
  /**
   * 计算表达式（单行数据）
   */
  static evaluate(node: ASTNode, row: any): any {
    switch (node.type) {
      case 'Number':
        return node.value
      
      case 'Field':
        return this.getFieldValue(row, node.name)
      
      case 'BinaryOp':
        return this.evaluateBinaryOp(node, row)
      
      case 'UnaryOp':
        return this.evaluateUnaryOp(node, row)
      
      case 'FunctionCall':
        return this.evaluateFunctionCall(node, row)
      
      case 'If':
        return this.evaluateIf(node, row)
      
      case 'Case':
        return this.evaluateCase(node, row)
      
      default:
        throw new Error(`Unknown node type: ${(node as any).type}`)
    }
  }

  /**
   * 计算二元操作
   */
  private static evaluateBinaryOp(node: BinaryOpNode, row: any): any {
    const left = this.evaluate(node.left, row)
    const right = this.evaluate(node.right, row)
    
    switch (node.operator) {
      case '*':
        return Number(left || 0) * Number(right || 0)
      case '/':
        const rightNum = Number(right || 0)
        if (rightNum === 0) return 0
        return Number(left || 0) / rightNum
      case '+':
        return Number(left || 0) + Number(right || 0)
      case '-':
        return Number(left || 0) - Number(right || 0)
      case '>':
        return Number(left || 0) > Number(right || 0)
      case '>=':
        return Number(left || 0) >= Number(right || 0)
      case '<':
        return Number(left || 0) < Number(right || 0)
      case '<=':
        return Number(left || 0) <= Number(right || 0)
      case '==':
        return left == right
      case '!=':
        return left != right
      case 'AND':
        return this.evaluate(node.left, row) && this.evaluate(node.right, row)
      case 'OR':
        return this.evaluate(node.left, row) || this.evaluate(node.right, row)
      default:
        throw new Error(`Unknown operator: ${node.operator}`)
    }
  }

  /**
   * 计算一元操作
   */
  private static evaluateUnaryOp(node: UnaryOpNode, row: any): any {
    const operand = this.evaluate(node.operand, row)
    
    switch (node.operator) {
      case 'NOT':
        return !operand
      case '-':
        return -Number(operand || 0)
      default:
        throw new Error(`Unknown unary operator: ${node.operator}`)
    }
  }

  /**
   * 计算函数调用
   */
  private static evaluateFunctionCall(node: FunctionCallNode, row: any): any {
    const args = node.args.map(arg => this.evaluate(arg, row))
    
    switch (node.name.toUpperCase()) {
      case 'COALESCE':
        // 返回第一个非空值
        for (const arg of args) {
          if (arg !== null && arg !== undefined && arg !== '') {
            return arg
          }
        }
        return null
      
      case 'IFNULL':
        // 如果第一个参数为空，返回第二个参数
        if (args.length < 2) throw new Error('IFNULL requires 2 arguments')
        return (args[0] !== null && args[0] !== undefined && args[0] !== '') ? args[0] : args[1]
      
      default:
        throw new Error(`Unknown function: ${node.name}`)
    }
  }

  /**
   * 计算 IF 表达式
   */
  private static evaluateIf(node: IfNode, row: any): any {
    const condition = this.evaluate(node.condition, row)
    if (condition) {
      return this.evaluate(node.thenBranch, row)
    } else if (node.elseBranch) {
      return this.evaluate(node.elseBranch, row)
    }
    return null
  }

  /**
   * 计算 CASE 表达式
   */
  private static evaluateCase(node: CaseNode, row: any): any {
    for (const caseItem of node.cases) {
      const condition = this.evaluate(caseItem.condition, row)
      if (condition) {
        return this.evaluate(caseItem.value, row)
      }
    }
    
    if (node.elseValue) {
      return this.evaluate(node.elseValue, row)
    }
    
    return null
  }

  /**
   * 获取字段值（支持中文字段名、英文字段名）
   */
  private static getFieldValue(row: any, fieldName: string): any {
    if (!row || !fieldName) {
      return null
    }
    
    // 直接访问字段
    if (row.hasOwnProperty(fieldName)) {
      return row[fieldName]
    }
    
    // 尝试处理嵌套字段
    if (fieldName.includes('.')) {
      const parts = fieldName.split('.')
      let value = row
      for (const part of parts) {
        if (value && value.hasOwnProperty(part)) {
          value = value[part]
        } else {
          return null
        }
      }
      return value
    }
    
    return null
  }
}

// ================ 聚合函数 ================

/**
 * ExpressionParserV2 - 新一代表达式解析器
 */
export class ExpressionParserV2 {
  /**
   * 计算表达式
   * @param expression 表达式字符串
   * @param data 数据数组
   * @param selectedItem 当前选中项（用于 value() 函数），可选
   * @returns 计算结果
   */
  static evaluate(expression: string, data: any[], selectedItem?: any): any {
    if (!expression) {
      return ''
    }

    try {
      // 解析表达式：函数名(参数)
      const match = expression.match(/^(\w+)\((.*)\)$/)
      if (!match) {
        // 不是函数调用，可能是纯文本，直接返回
        return expression
      }

      const [, funcName, argsStr] = match

      // 特殊处理：value() 函数
      if (funcName === 'value') {
        return this.evaluateValue(argsStr.trim(), selectedItem)
      }

      // 对于聚合函数，需要数据数组
      if (!data || data.length === 0) {
        return 0
      }

      // 判断是 List 层聚合还是行内聚合
      if (funcName.startsWith('list_')) {
        return this.evaluateListAggregation(funcName.slice(5), argsStr, data)
      } else {
        return this.evaluateRowAggregation(funcName, argsStr, data)
      }
    } catch (error) {
      Logger.error('ExpressionParserV2', `计算表达式失败: ${expression}`, error)
      return 0
    }
  }

  /**
   * 计算行内聚合
   */
  private static evaluateRowAggregation(funcName: string, argsStr: string, data: any[]): number {
    // 解析表达式参数（可能是复杂表达式）
    const ast = this.parseExpression(argsStr)
    
    // 根据函数名计算
    switch (funcName.toLowerCase()) {
      case 'sum':
        return this.calculateSum(ast, data)
      
      case 'count':
        return this.calculateCount(ast, data)
      
      case 'avg':
        return this.calculateAvg(ast, data)
      
      case 'min':
        return this.calculateMin(ast, data)
      
      case 'max':
        return this.calculateMax(ast, data)
      
      default:
        Logger.warn('ExpressionParserV2', `未知函数: ${funcName}`)
        return 0
    }
  }

  /**
   * 计算 List 层聚合
   */
  private static evaluateListAggregation(funcName: string, argsStr: string, data: any[]): number {
    const field = argsStr.trim()
    
    switch (funcName.toLowerCase()) {
      case 'sum':
        return this.calculateListSum(field, data)
      
      case 'avg':
        return this.calculateListAvg(field, data)
      
      case 'count':
        return data.length
      
      case 'min':
        return this.calculateListMin(field, data)
      
      case 'max':
        return this.calculateListMax(field, data)
      
      default:
        Logger.warn('ExpressionParserV2', `未知 List 函数: list_${funcName}`)
        return 0
    }
  }

  /**
   * 解析表达式（词法分析 + 语法分析）
   */
  private static parseExpression(expression: string): ASTNode {
    const lexer = new Lexer(expression)
    const tokens = lexer.tokenize()
    const parser = new Parser(tokens)
    return parser.parse()
  }

  /**
   * 计算求和
   */
  private static calculateSum(ast: ASTNode, data: any[]): number {
    return data.reduce((sum, row) => {
      try {
        const value = Evaluator.evaluate(ast, row)
        return sum + Number(value || 0)
      } catch (error) {
        Logger.warn('ExpressionParserV2', `计算行数据失败`, error)
        return sum
      }
    }, 0)
  }

  /**
   * 计算计数
   */
  private static calculateCount(ast: ASTNode, data: any[]): number {
    // Count 需要特殊处理：统计有多少个不同的值
    const values = new Set<string>()
    
    data.forEach(row => {
      try {
        const value = Evaluator.evaluate(ast, row)
        if (value !== null && value !== undefined && value !== '') {
          values.add(String(value))
        }
      } catch (error) {
        Logger.warn('ExpressionParserV2', `计算行数据失败`, error)
      }
    })
    
    return values.size
  }

  /**
   * 计算平均值
   */
  private static calculateAvg(ast: ASTNode, data: any[]): number {
    const sum = this.calculateSum(ast, data)
    const count = this.calculateCount(ast, data)
    return count > 0 ? sum / count : 0
  }

  /**
   * 计算最小值
   */
  private static calculateMin(ast: ASTNode, data: any[]): number {
    const values = data
      .map(row => {
        try {
          return Evaluator.evaluate(ast, row)
        } catch (error) {
          return null
        }
      })
      .filter(v => v !== null && v !== undefined && v !== '')
      .map(v => Number(v))
    
    return values.length > 0 ? Math.min(...values) : 0
  }

  /**
   * 计算最大值
   */
  private static calculateMax(ast: ASTNode, data: any[]): number {
    const values = data
      .map(row => {
        try {
          return Evaluator.evaluate(ast, row)
        } catch (error) {
          return null
        }
      })
      .filter(v => v !== null && v !== undefined && v !== '')
      .map(v => Number(v))
    
    return values.length > 0 ? Math.max(...values) : 0
  }

  /**
   * List 层求和
   */
  private static calculateListSum(field: string, data: any[]): number {
    return data.reduce((sum, row) => {
      const value = this.getFieldValue(row, field)
      return sum + Number(value || 0)
    }, 0)
  }

  /**
   * List 层平均值
   */
  private static calculateListAvg(field: string, data: any[]): number {
    const sum = this.calculateListSum(field, data)
    return data.length > 0 ? sum / data.length : 0
  }

  /**
   * List 层最小值
   */
  private static calculateListMin(field: string, data: any[]): number {
    const values = data
      .map(row => this.getFieldValue(row, field))
      .filter(v => v !== null && v !== undefined && v !== '')
      .map(v => Number(v))
    
    return values.length > 0 ? Math.min(...values) : 0
  }

  /**
   * List 层最大值
   */
  private static calculateListMax(field: string, data: any[]): number {
    const values = data
      .map(row => this.getFieldValue(row, field))
      .filter(v => v !== null && v !== undefined && v !== '')
      .map(v => Number(v))
    
    return values.length > 0 ? Math.max(...values) : 0
  }

  /**
   * 获取字段值
   */
  private static getFieldValue(row: any, fieldName: string): any {
    if (!row || !fieldName) {
      return null
    }
    
    if (row.hasOwnProperty(fieldName)) {
      return row[fieldName]
    }
    
    if (fieldName.includes('.')) {
      const parts = fieldName.split('.')
      let value = row
      for (const part of parts) {
        if (value && value.hasOwnProperty(part)) {
          value = value[part]
        } else {
          return null
        }
      }
      return value
    }
    
    return null
  }

  /**
   * 计算选中项的字段值（value() 函数）
   */
  private static evaluateValue(fieldName: string, selectedItem?: any): any {
    if (!selectedItem || !fieldName) {
      return ''
    }
    
    let displayInfo: any = null
    
    if (selectedItem.displayInfo) {
      displayInfo = selectedItem.displayInfo
    } else if (selectedItem.display_info) {
      displayInfo = selectedItem.display_info
    } else if (typeof selectedItem === 'object' && !Array.isArray(selectedItem)) {
      displayInfo = selectedItem
    }
    
    if (!displayInfo || typeof displayInfo !== 'object') {
      return ''
    }
    
    const value = displayInfo[fieldName]
    
    if (value === null || value === undefined || value === '') {
      return ''
    }
    
    return value
  }
}

