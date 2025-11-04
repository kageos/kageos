# API 使用示例

本文档演示如何使用 app-storage 的各个 API。

## 前置条件

1. MinIO 已启动（`http://localhost:9000`）
2. app-storage 已启动（`http://localhost:8083`）
3. 已获取 JWT Token（需要先登录 app-server）

```bash
# 设置环境变量
export JWT_TOKEN="your-jwt-token-here"
export BASE_URL="http://localhost:8083/api/v1/storage"
```

## 完整上传流程

### 步骤 1：获取上传凭证

```bash
curl -X POST "$BASE_URL/upload_token" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN" \
  -d '{
    "router": "luobei/test88888/tools/cashier_desk",
    "file_name": "invoice.pdf",
    "content_type": "application/pdf",
    "file_size": 102400
  }'
```

响应：

```json
{
  "code": 0,
  "data": {
    "url": "http://localhost:9000/ai-agent-os/luobei/test88888/tools/cashier_desk/2025/11/03/550e8400-e29b-41d4-a716-446655440000.pdf?X-Amz-Algorithm=AWS4-HMAC-SHA256&...",
    "key": "luobei/test88888/tools/cashier_desk/2025/11/03/550e8400-e29b-41d4-a716-446655440000.pdf",
    "method": "PUT",
    "expire": "2025-11-03 22:48:00",
    "headers": {
      "Content-Type": "application/pdf"
    },
    "bucket": "ai-agent-os"
  }
}
```

### 步骤 2：上传文件（前端直接调用）

```bash
# 提取 URL
UPLOAD_URL=$(curl -s -X POST "$BASE_URL/upload_token" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN" \
  -d '{
    "router": "luobei/test88888/tools/cashier_desk",
    "file_name": "test.txt",
    "content_type": "text/plain",
    "file_size": 100
  }' | jq -r '.data.url')

# 上传文件
echo "Hello, MinIO!" > /tmp/test.txt
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: text/plain" \
  --data-binary "@/tmp/test.txt"
```

### 步骤 3：获取文件信息

```bash
FILE_KEY="luobei/test88888/tools/cashier_desk/2025/11/03/550e8400-e29b-41d4-a716-446655440000.txt"

curl -X GET "$BASE_URL/files/$FILE_KEY/info" \
  -H "X-Token: $JWT_TOKEN"
```

### 步骤 4：获取下载链接

```bash
curl -X GET "$BASE_URL/download/$FILE_KEY" \
  -H "X-Token: $JWT_TOKEN"
```

### 步骤 5：下载文件

```bash
DOWNLOAD_URL=$(curl -s -X GET "$BASE_URL/download/$FILE_KEY" \
  -H "X-Token: $JWT_TOKEN" | jq -r '.data.url')

curl -s "$DOWNLOAD_URL"
```

## 批量操作示例

### 1. 列举某个函数的所有文件

```bash
curl -X GET "$BASE_URL/files?router=luobei/test88888/tools/cashier_desk" \
  -H "X-Token: $JWT_TOKEN"
```

响应：

```json
{
  "code": 0,
  "data": {
    "router": "luobei/test88888/tools/cashier_desk",
    "files": [
      "luobei/test88888/tools/cashier_desk/2025/11/03/xxx.pdf",
      "luobei/test88888/tools/cashier_desk/2025/11/03/yyy.jpg",
      "luobei/test88888/tools/cashier_desk/2025/11/04/zzz.png"
    ],
    "count": 3
  }
}
```

### 2. 获取存储统计

```bash
curl -X GET "$BASE_URL/stats?router=luobei/test88888/tools/cashier_desk" \
  -H "X-Token: $JWT_TOKEN"
```

响应：

```json
{
  "code": 0,
  "data": {
    "router": "luobei/test88888/tools/cashier_desk",
    "file_count": 15,
    "total_size": 2048576,
    "size_human": "2.0 MB"
  }
}
```

### 3. 按应用统计

```bash
# 统计整个应用的存储占用
curl -X GET "$BASE_URL/stats?router=luobei/test88888" \
  -H "X-Token: $JWT_TOKEN"
```

### 4. 按租户统计

```bash
# 统计整个租户的存储占用
curl -X GET "$BASE_URL/stats?router=luobei" \
  -H "X-Token: $JWT_TOKEN"
```

### 5. 批量删除（危险操作）

```bash
# 删除某个函数的所有文件
curl -X POST "$BASE_URL/batch_delete" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN" \
  -d '{
    "router": "luobei/test88888/tools/cashier_desk"
  }'
```

响应：

```json
{
  "code": 0,
  "data": {
    "router": "luobei/test88888/tools/cashier_desk",
    "deleted_count": 15
  }
}
```

## 多租户场景示例

### 场景 1：不同租户上传文件

```bash
# 租户 A 上传文件
curl -X POST "$BASE_URL/upload_token" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN_A" \
  -d '{
    "router": "tenant_a/app1/function1",
    "file_name": "file_a.jpg",
    "content_type": "image/jpeg",
    "file_size": 102400
  }'

# 租户 B 上传文件
curl -X POST "$BASE_URL/upload_token" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN_B" \
  -d '{
    "router": "tenant_b/app1/function1",
    "file_name": "file_b.jpg",
    "content_type": "image/jpeg",
    "file_size": 102400
  }'
```

结果：两个文件存储在不同的路径下，完全隔离。

### 场景 2：统计每个租户的存储占用

```bash
# 统计租户 A 的总存储
curl -X GET "$BASE_URL/stats?router=tenant_a" \
  -H "X-Token: $JWT_TOKEN_ADMIN"

# 统计租户 B 的总存储
curl -X GET "$BASE_URL/stats?router=tenant_b" \
  -H "X-Token: $JWT_TOKEN_ADMIN"
```

### 场景 3：清理某个应用的所有文件

```bash
# 删除应用 app1 下的所有文件（所有函数）
curl -X POST "$BASE_URL/batch_delete" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN" \
  -d '{
    "router": "tenant_a/app1"
  }'
```

## 前端集成示例

### Vue.js 示例

```typescript
// 上传文件到 MinIO
async function uploadFile(router: string, file: File) {
  // 1. 获取上传凭证
  const tokenRes = await fetch('/api/v1/storage/upload_token', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Token': getJwtToken(),
    },
    body: JSON.stringify({
      router: router,
      file_name: file.name,
      content_type: file.type,
      file_size: file.size,
    }),
  });
  
  const tokenData = await tokenRes.json();
  const { url, key } = tokenData.data;
  
  // 2. 直接上传到 MinIO
  await fetch(url, {
    method: 'PUT',
    headers: {
      'Content-Type': file.type,
    },
    body: file,
  });
  
  // 3. 返回文件 Key
  return key;
}

// 使用示例
const file = event.target.files[0];
const router = 'luobei/test88888/tools/cashier_desk';
const fileKey = await uploadFile(router, file);
console.log('File uploaded:', fileKey);
```

### React 示例

```typescript
import { useState } from 'react';

function FileUploader({ router }: { router: string }) {
  const [uploading, setUploading] = useState(false);
  
  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    
    setUploading(true);
    
    try {
      // 获取上传凭证
      const tokenRes = await fetch('/api/v1/storage/upload_token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Token': localStorage.getItem('jwt_token'),
        },
        body: JSON.stringify({
          router,
          file_name: file.name,
          content_type: file.type,
          file_size: file.size,
        }),
      });
      
      const { data } = await tokenRes.json();
      
      // 上传到 MinIO
      await fetch(data.url, {
        method: 'PUT',
        headers: { 'Content-Type': file.type },
        body: file,
      });
      
      alert('Upload success! Key: ' + data.key);
    } catch (err) {
      alert('Upload failed: ' + err.message);
    } finally {
      setUploading(false);
    }
  };
  
  return (
    <div>
      <input type="file" onChange={handleUpload} disabled={uploading} />
      {uploading && <span>Uploading...</span>}
    </div>
  );
}
```

## 常见问题

### Q1: 如何处理大文件上传？

A: MinIO 支持分片上传，但需要客户端实现。建议使用 MinIO 官方的 JavaScript SDK：

```bash
npm install minio
```

### Q2: 上传凭证过期了怎么办？

A: 默认凭证有效期为 1 小时。如果上传时间超过 1 小时，需要重新获取凭证。

### Q3: 如何实现断点续传？

A: 使用 MinIO 的 Multipart Upload API，客户端需要记录已上传的分片，失败后可以从断点继续。

### Q4: 如何限制文件类型？

A: 在后端配置 `allowed_types`，或在前端验证 `file.type`。

### Q5: 如何生成缩略图？

A: 可以在上传成功后，后端异步生成缩略图并保存到 MinIO，缩略图的 Key 可以是 `{original_key}_thumb.jpg`。

