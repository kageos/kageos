# 权限常量统一管理规范

## 📋 概述

为了避免硬编码权限字符串导致的混乱和难以维护，我们统一管理权限常量。

### 🎯 设计原则

1. **统一定义**：权限常量在前后端各有一个统一的定义文件
2. **避免硬编码**：禁止在代码中直接写 `'directory:read'` 等字符串
3. **命名规范**：使用小驼峰命名（如 `DirectoryPermission`），避免纯大写
4. **前后端同步**：前后端常量定义保持一致

---

## 📁 文件位置

### 前端
```
web/src/constants/permissions.ts
```

### 后端
```
core/app-server/constants/permission.go
```

---

## 🎨 权限体系

### 资源类型（Resource Type）

| 资源类型 | 说明 | 示例 |
|---------|------|------|
| `directory` | 目录（包括根目录/工作空间） | `/user/app/package1` |
| `function` | 函数 | `/user/app/package1/func1` |
| `table` | 表格 | Table 类型的函数 |
| `form` | 表单 | Form 类型的函数 |
| `chart` | 图表 | Chart 类型的函数 |
| `docs` | 文档 | Docs 类型的节点 |

### 操作类型（Action Type）

| 操作类型 | 说明 |
|---------|------|
| `read` | 查看权限 |
| `write` | 写入/创建权限 |
| `update` | 更新权限 |
| `delete` | 删除权限 |
| `admin` | 管理员权限（包含所有权限） |
| `query` | 查询权限（仅用于 chart） |

### 权限点格式

```
资源类型:操作类型
```

示例：
- `directory:read` - 查看目录
- `function:write` - 创建/写入函数
- `table:update` - 更新表格数据
- `docs:admin` - 文档管理员

---

## 💻 前端使用示例

### 1. 导入权限常量

```typescript
// ✅ 推荐：从统一常量文件导入
import { 
  DirectoryPermission, 
  FunctionPermission,
  TablePermission,
  DocsPermission,
} from '@/constants/permissions'

// ✅ 也可以从 utils/permission 导入（会自动转发）
import { DirectoryPermission } from '@/utils/permission'
```

### 2. 使用权限常量

```typescript
// ✅ 正确：使用常量
if (hasPermission(node, DirectoryPermission.read)) {
  // 查看目录
}

if (hasPermission(node, DirectoryPermission.write)) {
  // 创建子目录
}

if (hasPermission(node, DocsPermission.admin)) {
  // 管理文档
}

// ❌ 错误：硬编码字符串
if (hasPermission(node, 'directory:read')) {  // 禁止！
  // ...
}
```

### 3. 构建权限点

```typescript
import { buildPermission, ResourceType, ActionType } from '@/constants/permissions'

// 动态构建权限点
const permission = buildPermission('directory', 'read')  // 'directory:read'

// 使用枚举
const permission2 = buildPermission(ResourceType.directory, ActionType.read)
```

### 4. 解析权限点

```typescript
import { parsePermission } from '@/constants/permissions'

const result = parsePermission('directory:read')
// { resourceType: 'directory', actionType: 'read' }
```

### 5. 根据资源类型获取权限对象

```typescript
import { getPermissionsByResourceType, ResourceType } from '@/constants/permissions'

const perms = getPermissionsByResourceType(ResourceType.directory)
// DirectoryPermission 对象
```

---

## 🔧 后端使用示例

### 1. 导入权限常量

```go
import "github.com/ai-agent-os/ai-agent-os/core/app-server/constants"
```

### 2. 使用权限常量

```go
// ✅ 正确：使用常量
if hasPermission(user, resourcePath, constants.PermissionDirectoryRead) {
    // 查看目录
}

if hasPermission(user, resourcePath, constants.PermissionDirectoryWrite) {
    // 创建子目录
}

if hasPermission(user, resourcePath, constants.PermissionDocsAdmin) {
    // 管理文档
}

// ❌ 错误：硬编码字符串
if hasPermission(user, resourcePath, "directory:read") {  // 禁止！
    // ...
}
```

### 3. 构建权限点

```go
// 动态构建权限点
permission := constants.BuildPermission(constants.ResourceTypeDirectory, constants.ActionTypeRead)
// "directory:read"
```

### 4. 解析权限点

```go
resourceType, actionType, ok := constants.ParsePermission("directory:read")
// resourceType: "directory"
// actionType: "read"
// ok: true
```

### 5. 获取资源类型的所有权限

```go
permissions := constants.GetPermissionsByResourceType(constants.ResourceTypeDirectory)
// []string{"directory:read", "directory:write", "directory:update", "directory:delete", "directory:admin"}
```

---

## 📋 权限常量清单

### 目录权限（Directory）

```typescript
// 前端
DirectoryPermission.read    // 'directory:read'
DirectoryPermission.write   // 'directory:write'
DirectoryPermission.update  // 'directory:update'
DirectoryPermission.delete  // 'directory:delete'
DirectoryPermission.admin   // 'directory:admin'
```

```go
// 后端
constants.PermissionDirectoryRead
constants.PermissionDirectoryWrite
constants.PermissionDirectoryUpdate
constants.PermissionDirectoryDelete
constants.PermissionDirectoryAdmin
```

### 函数权限（Function）

```typescript
// 前端
FunctionPermission.read
FunctionPermission.write
FunctionPermission.update
FunctionPermission.delete
FunctionPermission.admin
```

```go
// 后端
constants.PermissionFunctionRead
constants.PermissionFunctionWrite
constants.PermissionFunctionUpdate
constants.PermissionFunctionDelete
constants.PermissionFunctionAdmin
```

### 表格权限（Table）

```typescript
// 前端
TablePermission.read
TablePermission.write
TablePermission.update
TablePermission.delete
TablePermission.admin
```

```go
// 后端
constants.PermissionTableRead
constants.PermissionTableWrite
constants.PermissionTableUpdate
constants.PermissionTableDelete
constants.PermissionTableAdmin
```

### 表单权限（Form）

```typescript
// 前端
FormPermission.read
FormPermission.write
FormPermission.update
FormPermission.delete
FormPermission.admin
```

```go
// 后端
constants.PermissionFormRead
constants.PermissionFormWrite
constants.PermissionFormUpdate
constants.PermissionFormDelete
constants.PermissionFormAdmin
```

### 图表权限（Chart）

```typescript
// 前端
ChartPermission.read
ChartPermission.query   // 特有：查询数据
ChartPermission.update
ChartPermission.delete
ChartPermission.admin
```

```go
// 后端
constants.PermissionChartRead
constants.PermissionChartQuery
constants.PermissionChartUpdate
constants.PermissionChartDelete
constants.PermissionChartAdmin
```

### 文档权限（Docs）

```typescript
// 前端
DocsPermission.read
DocsPermission.write
DocsPermission.update
DocsPermission.delete
DocsPermission.admin
```

```go
// 后端
constants.PermissionDocsRead
constants.PermissionDocsWrite
constants.PermissionDocsUpdate
constants.PermissionDocsDelete
constants.PermissionDocsAdmin
```

---

## ⚠️ 迁移指南

### 旧代码（硬编码）

```typescript
// ❌ 旧代码
if (hasPermission(node, 'directory:read')) { }
if (hasPermission(node, 'directory:write')) { }
if (hasPermission(node, 'table:update')) { }
```

### 新代码（使用常量）

```typescript
// ✅ 新代码
import { DirectoryPermission, TablePermission } from '@/constants/permissions'

if (hasPermission(node, DirectoryPermission.read)) { }
if (hasPermission(node, DirectoryPermission.write)) { }
if (hasPermission(node, TablePermission.update)) { }
```

### 批量替换（VS Code）

1. 打开全局搜索（Cmd+Shift+F）
2. 搜索：`'directory:read'`
3. 替换为：`DirectoryPermission.read`
4. 添加导入：`import { DirectoryPermission } from '@/constants/permissions'`

---

## 🔍 代码审查检查点

在代码审查时，请检查：

- [ ] ✅ 是否使用了权限常量
- [ ] ❌ 是否有硬编码的权限字符串
- [ ] ✅ 是否有正确的导入语句
- [ ] ✅ 新增的权限点是否已添加到常量文件

---

## 📝 注意事项

1. **禁止硬编码**：任何权限判断都必须使用常量，禁止直接写字符串
2. **前后端同步**：修改权限体系时，前后端常量文件需要同步更新
3. **命名规范**：使用小驼峰，如 `DirectoryPermission`，不要用 `DIRECTORY_PERMISSION`
4. **向后兼容**：旧代码中的 `DirectoryPermissions` 仍然可用，但建议迁移到新的 `DirectoryPermission`

---

## 🎯 最佳实践

### 1. 权限检查

```typescript
// ✅ Good
import { DirectoryPermission } from '@/constants/permissions'

if (hasPermission(node, DirectoryPermission.admin)) {
  // 显示管理按钮
}

// ❌ Bad
if (hasPermission(node, 'directory:admin')) {
  // 硬编码字符串
}
```

### 2. 权限申请

```typescript
// ✅ Good
import { DirectoryPermission } from '@/constants/permissions'

const defaultAction = DirectoryPermission.read
router.push(`/permissions/apply?action=${defaultAction}`)

// ❌ Bad
router.push('/permissions/apply?action=directory:read')
```

### 3. 条件渲染

```typescript
// ✅ Good
import { DirectoryPermission, TablePermission } from '@/constants/permissions'

<el-button v-if="hasPermission(node, DirectoryPermission.write)">
  创建子目录
</el-button>

// ❌ Bad
<el-button v-if="hasPermission(node, 'directory:write')">
  创建子目录
</el-button>
```

---

## 🚀 后续优化

1. 添加 ESLint 规则，禁止硬编码权限字符串
2. 自动化工具：检测代码中的硬编码权限字符串
3. 前后端常量文件自动生成工具
4. 权限点文档自动生成

---

## 📚 相关文档

- [权限系统架构](./permission-system.md)
- [角色管理](./role-management.md)
- [权限申请流程](./permission-apply.md)
