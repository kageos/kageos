# 使用GORM Tabler接口获取表名

## 🎯 改进说明

我们改进了`CreateTables`字段的提取方式，从使用反射改为使用GORM的`schema.Tabler`接口，这样更加优雅和类型安全。

## 📋 改进前后对比

### 改进前（使用反射）
```go
// 复杂的反射调用
t := reflect.TypeOf(createTable)
if t.Kind() == reflect.Ptr {
    t = t.Elem()
}
if t.Kind() == reflect.Struct {
    if method, ok := t.MethodByName("TableName"); ok {
        if method.Type.NumIn() == 0 && method.Type.NumOut() == 1 {
            results := method.Func.Call([]reflect.Value{reflect.ValueOf(createTable)})
            if len(results) > 0 {
                if tableName, ok := results[0].Interface().(string); ok {
                    api.CreateTables = append(api.CreateTables, tableName)
                }
            }
        }
    }
}
```

### 改进后（使用Tabler接口）
```go
// 简洁的类型断言
if tabler, ok := createTable.(interface{ TableName() string }); ok {
    api.CreateTables = append(api.CreateTables, tabler.TableName())
}
```

## 🔧 使用方式

### 1. 在你的模型结构体中实现Tabler接口

```go
package crm

import "gorm.io/gorm/schema"

type CrmTicket struct {
    ID       int    `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
    Title    string `json:"title" gorm:"column:title"`
    Priority string `json:"priority" gorm:"column:priority"`
}

// 实现Tabler接口
func (CrmTicket) TableName() string {
    return "crm_ticket"
}

// 或者使用GORM的内置Tabler
// func (CrmTicket) TableName() string {
//     return schema.NamingStrategy{}.TableName("CrmTicket")
// }
```

### 2. 在模板中声明创建表

```go
var CrmTicketTemplate = &app.TableTemplate{
    BaseConfig: app.BaseConfig{
        Name: "工单管理",
        CreateTables: []interface{}{
            &CrmTicket{}, // 只需要声明即可
        },
        Request:  &CrmTicket{},
        Response: []*CrmTicket{},
    },
    AutoCrudTable: &CrmTicket{},
    // ... 其他配置
}
```

### 3. 系统自动提取表名

```go
// SDK内部自动处理
for _, createTable := range base.CreateTables {
    if createTable != nil {
        createTables = append(createTables, createTable)

        // 自动调用TableName()方法获取表名
        if tabler, ok := createTable.(interface{ TableName() string }); ok {
            api.CreateTables = append(api.CreateTables, tabler.TableName())
        }
    }
}
```

## ✅ 优势

1. **类型安全**: 编译时检查，避免运行时错误
2. **代码简洁**: 一行代码搞定，无需复杂的反射操作
3. **性能更好**: 避免反射的性能开销
4. **符合Go习惯**: 使用接口断言而非反射
5. **易于维护**: 代码更清晰，更容易理解

## 🔗 GORM Tabler接口说明

GORM提供了`schema.Tabler`接口，让你的模型可以自定义表名：

```go
// GORM的Tabler接口定义
type Tabler interface {
    TableName() string
}
```

### 常见的表名命名策略

```go
// 1. 直接指定表名
func (User) TableName() string {
    return "app_users"
}

// 2. 使用GORM的命名策略
func (User) TableName() string {
    return schema.NamingStrategy{}.TableName("User")
}

// 3. 带前缀的表名
func (User) TableName() string {
    return "crm_" + schema.NamingStrategy{}.TableName("User")
}

// 4. 动态表名（不推荐，但支持）
func (User) TableName() string {
    return "users_" + time.Now().Format("200601")
}
```

## 🎯 在你的SDK中的应用

这个改进让你的API diff功能能够：

1. **准确识别表变更**: 当模型结构体发生变化时，自动检测到表结构变更
2. **支持复杂表名**: 支持任何自定义的表名格式
3. **自动创建表**: 在API变更时自动创建新的数据表
4. **版本化管理**: 将表名变更记录在API版本历史中

## 📊 实际应用示例

当用户说："我需要给工单系统增加一个附件字段"，LLM生成的代码：

```go
type CrmTicket struct {
    ID          int       `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
    Title       string    `json:"title" gorm:"column:title"`
    Priority    string    `json:"priority" gorm:"column:priority"`
    Attachments *Files    `json:"attachments" gorm:"type:text;column:attachments"` // 新增字段
}

func (CrmTicket) TableName() string {
    return "crm_ticket"
}
```

系统会自动：
1. 识别到新增了`Attachments`字段
2. 检测到表结构变更
3. 在diff结果中标记为`update`
4. 前端自动添加文件上传组件

这就是"所描即所得"的技术实现！🚀
