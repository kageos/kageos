# Files 组件架构设计

## 一、核心问题分析

### 当前痛点

1. **旧版本与七牛云强耦合** ❌
   - 无法支持私有化部署
   - 无法切换存储服务商
   - 数据必须出域

2. **私有化部署需求** ✅
   - 企业客户数据不能出域
   - 需要支持自建存储
   - 需要支持多种存储后端

3. **两种使用场景混杂** ⚠️
   - 场景1：后端需要处理文件（视频转换、OCR 等）
   - 场景2：后端只存储元数据（工单附件）

---

## 二、架构设计（核心方案）

### 🎯 设计原则

1. **存储抽象** - 支持多种存储后端（MinIO、七牛云、阿里云 OSS、AWS S3）
2. **职责分离** - 前端上传、后端处理、存储服务独立
3. **安全第一** - 签名上传、权限控制、病毒扫描
4. **灵活配置** - 支持云端和私有化部署

---

### 🏗️ 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Files 组件架构                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    前端 (Vue)                             │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │   │
│  │  │ FileUpload │  │ FileList   │  │ FilePreview│         │   │
│  │  │  Widget    │  │  Widget    │  │   Widget   │         │   │
│  │  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘         │   │
│  │        │                │                │                 │   │
│  │        └────────────────┴────────────────┘                 │   │
│  │                         │                                   │   │
│  └─────────────────────────┼───────────────────────────────┘   │
│                            │                                     │
│                            │ HTTP API                            │
│                            │                                     │
│  ┌─────────────────────────▼───────────────────────────────┐   │
│  │                  App Server (Go)                         │   │
│  │  ┌────────────────────────────────────────────────────┐ │   │
│  │  │           Storage Manager (抽象层)                  │ │   │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐         │ │   │
│  │  │  │  MinIO   │  │  Qiniu   │  │  Local   │  ...    │ │   │
│  │  │  │ Provider │  │ Provider │  │ Provider │         │ │   │
│  │  │  └──────────┘  └──────────┘  └──────────┘         │ │   │
│  │  └────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────┬───────────────────────────────┘   │
│                            │                                     │
│  ┌─────────────────────────▼───────────────────────────────┐   │
│  │              实际存储 (可插拔)                           │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐              │   │
│  │  │  MinIO   │  │ 七牛云   │  │ 本地磁盘 │  ...         │   │
│  │  │  集群    │  │   OSS    │  │  /data   │              │   │
│  │  └──────────┘  └──────────┘  └──────────┘              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 三、后端存储抽象层设计

### 📦 Storage Provider 接口

```go
// pkg/storage/provider.go

package storage

import (
    "context"
    "io"
    "time"
)

// Provider 存储提供者接口（核心抽象）
type Provider interface {
    // GetUploadToken 获取上传凭证（前端直传需要）
    GetUploadToken(ctx context.Context, opts *UploadOptions) (*UploadToken, error)
    
    // Upload 上传文件（后端处理文件时需要）
    Upload(ctx context.Context, reader io.Reader, opts *UploadOptions) (*FileInfo, error)
    
    // Download 下载文件
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    
    // GetURL 获取文件访问 URL（支持临时签名）
    GetURL(ctx context.Context, key string, expire time.Duration) (string, error)
    
    // Delete 删除文件
    Delete(ctx context.Context, key string) error
    
    // Exists 检查文件是否存在
    Exists(ctx context.Context, key string) (bool, error)
    
    // GetMetadata 获取文件元数据
    GetMetadata(ctx context.Context, key string) (*FileMetadata, error)
}

// UploadOptions 上传选项
type UploadOptions struct {
    Key         string            // 文件存储路径（如 "users/123/avatar.jpg"）
    Bucket      string            // 存储桶名称
    ContentType string            // 文件类型
    MaxSize     int64             // 最大文件大小（字节）
    Expire      time.Duration     // 上传凭证过期时间
    Metadata    map[string]string // 自定义元数据
    Public      bool              // 是否公开访问
}

// UploadToken 上传凭证
type UploadToken struct {
    Token      string            // 上传凭证
    URL        string            // 上传地址
    Key        string            // 文件存储路径
    Expire     time.Time         // 过期时间
    Method     string            // 上传方式（PUT/POST/FORM）
    Headers    map[string]string // 需要的 HTTP 头
    FormFields map[string]string // 表单字段（POST 上传时需要）
}

// FileInfo 文件信息
type FileInfo struct {
    Key         string            // 文件存储路径
    URL         string            // 访问 URL
    Size        int64             // 文件大小
    ContentType string            // 文件类型
    Hash        string            // 文件哈希（MD5/SHA256）
    UploadedAt  time.Time         // 上传时间
    Metadata    map[string]string // 元数据
}

// FileMetadata 文件元数据
type FileMetadata struct {
    Key         string
    Size        int64
    ContentType string
    Hash        string
    CreatedAt   time.Time
    ModifiedAt  time.Time
}
```

---

### 🔌 具体实现（MinIO Provider）

```go
// pkg/storage/minio/provider.go

package minio

import (
    "context"
    "fmt"
    "io"
    "time"
    
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
    "github.com/ai-agent-os/ai-agent-os/pkg/storage"
)

type MinIOProvider struct {
    client *minio.Client
    bucket string
    region string
}

func NewMinIOProvider(endpoint, accessKey, secretKey, bucket, region string, useSSL bool) (*MinIOProvider, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
        Region: region,
    })
    if err != nil {
        return nil, err
    }
    
    return &MinIOProvider{
        client: client,
        bucket: bucket,
        region: region,
    }, nil
}

// GetUploadToken 生成预签名上传 URL
func (p *MinIOProvider) GetUploadToken(ctx context.Context, opts *storage.UploadOptions) (*storage.UploadToken, error) {
    // MinIO 使用预签名 URL 方式上传
    presignedURL, err := p.client.PresignedPutObject(ctx, 
        opts.Bucket, 
        opts.Key, 
        opts.Expire,
    )
    if err != nil {
        return nil, err
    }
    
    return &storage.UploadToken{
        URL:    presignedURL.String(),
        Key:    opts.Key,
        Method: "PUT",
        Expire: time.Now().Add(opts.Expire),
        Headers: map[string]string{
            "Content-Type": opts.ContentType,
        },
    }, nil
}

// Upload 直接上传文件（后端使用）
func (p *MinIOProvider) Upload(ctx context.Context, reader io.Reader, opts *storage.UploadOptions) (*storage.FileInfo, error) {
    info, err := p.client.PutObject(ctx,
        opts.Bucket,
        opts.Key,
        reader,
        opts.MaxSize,
        minio.PutObjectOptions{
            ContentType: opts.ContentType,
            UserMetadata: opts.Metadata,
        },
    )
    if err != nil {
        return nil, err
    }
    
    return &storage.FileInfo{
        Key:         opts.Key,
        URL:         p.getPublicURL(opts.Bucket, opts.Key),
        Size:        info.Size,
        ContentType: opts.ContentType,
        Hash:        info.ETag,
        UploadedAt:  time.Now(),
        Metadata:    opts.Metadata,
    }, nil
}

// Download 下载文件
func (p *MinIOProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
    object, err := p.client.GetObject(ctx, p.bucket, key, minio.GetObjectOptions{})
    if err != nil {
        return nil, err
    }
    return object, nil
}

// GetURL 获取访问 URL（支持临时签名）
func (p *MinIOProvider) GetURL(ctx context.Context, key string, expire time.Duration) (string, error) {
    if expire > 0 {
        // 返回临时签名 URL
        presignedURL, err := p.client.PresignedGetObject(ctx, p.bucket, key, expire, nil)
        if err != nil {
            return "", err
        }
        return presignedURL.String(), nil
    }
    
    // 返回公开 URL
    return p.getPublicURL(p.bucket, key), nil
}

// 其他方法实现...
```

---

### 🔌 七牛云 Provider

```go
// pkg/storage/qiniu/provider.go

package qiniu

import (
    "context"
    "io"
    "time"
    
    "github.com/qiniu/go-sdk/v7/auth"
    "github.com/qiniu/go-sdk/v7/storage"
    "github.com/ai-agent-os/ai-agent-os/pkg/storage"
)

type QiniuProvider struct {
    mac    *auth.Credentials
    bucket string
    domain string // CDN 域名
}

func NewQiniuProvider(accessKey, secretKey, bucket, domain string) *QiniuProvider {
    return &QiniuProvider{
        mac:    auth.New(accessKey, secretKey),
        bucket: bucket,
        domain: domain,
    }
}

// GetUploadToken 生成上传凭证
func (p *QiniuProvider) GetUploadToken(ctx context.Context, opts *storage.UploadOptions) (*storage.UploadToken, error) {
    putPolicy := storage.PutPolicy{
        Scope:      fmt.Sprintf("%s:%s", p.bucket, opts.Key),
        Expires:    uint64(opts.Expire.Seconds()),
        ReturnBody: `{"key":"$(key)","hash":"$(etag)","size":$(fsize)}`,
    }
    
    token := putPolicy.UploadToken(p.mac)
    
    return &storage.UploadToken{
        Token:  token,
        URL:    "https://upload.qiniup.com", // 七牛云上传地址
        Key:    opts.Key,
        Method: "POST",
        Expire: time.Now().Add(opts.Expire),
        FormFields: map[string]string{
            "token": token,
            "key":   opts.Key,
        },
    }, nil
}

// 其他方法实现...
```

---

### 🔌 本地文件系统 Provider（私有化部署）

```go
// pkg/storage/local/provider.go

package local

import (
    "context"
    "crypto/md5"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    
    "github.com/ai-agent-os/ai-agent-os/pkg/storage"
)

type LocalProvider struct {
    basePath  string // 基础路径（如 "/data/uploads"）
    baseURL   string // 访问基础 URL（如 "http://localhost:8080/files"）
}

func NewLocalProvider(basePath, baseURL string) (*LocalProvider, error) {
    // 确保目录存在
    if err := os.MkdirAll(basePath, 0755); err != nil {
        return nil, err
    }
    
    return &LocalProvider{
        basePath: basePath,
        baseURL:  baseURL,
    }, nil
}

// GetUploadToken 本地存储不需要 token，返回上传 API 地址
func (p *LocalProvider) GetUploadToken(ctx context.Context, opts *storage.UploadOptions) (*storage.UploadToken, error) {
    return &storage.UploadToken{
        URL:    fmt.Sprintf("%s/upload", p.baseURL),
        Key:    opts.Key,
        Method: "POST",
        Expire: time.Now().Add(opts.Expire),
    }, nil
}

// Upload 上传文件到本地磁盘
func (p *LocalProvider) Upload(ctx context.Context, reader io.Reader, opts *storage.UploadOptions) (*storage.FileInfo, error) {
    filePath := filepath.Join(p.basePath, opts.Key)
    
    // 确保目录存在
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    
    // 创建文件
    file, err := os.Create(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    // 复制数据并计算哈希
    hash := md5.New()
    written, err := io.Copy(io.MultiWriter(file, hash), reader)
    if err != nil {
        os.Remove(filePath) // 清理失败的文件
        return nil, err
    }
    
    return &storage.FileInfo{
        Key:         opts.Key,
        URL:         fmt.Sprintf("%s/%s", p.baseURL, opts.Key),
        Size:        written,
        ContentType: opts.ContentType,
        Hash:        fmt.Sprintf("%x", hash.Sum(nil)),
        UploadedAt:  time.Now(),
        Metadata:    opts.Metadata,
    }, nil
}

// Download 从本地磁盘下载文件
func (p *LocalProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
    filePath := filepath.Join(p.basePath, key)
    return os.Open(filePath)
}

// 其他方法实现...
```

---

### 🔧 Storage Manager（统一管理）

```go
// pkg/storage/manager.go

package storage

import (
    "context"
    "fmt"
    "sync"
)

// Manager 存储管理器（支持多个存储后端）
type Manager struct {
    providers map[string]Provider
    default   string
    mu        sync.RWMutex
}

func NewManager() *Manager {
    return &Manager{
        providers: make(map[string]Provider),
    }
}

// Register 注册存储提供者
func (m *Manager) Register(name string, provider Provider, isDefault bool) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.providers[name] = provider
    if isDefault {
        m.default = name
    }
}

// GetProvider 获取存储提供者
func (m *Manager) GetProvider(name string) (Provider, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if name == "" {
        name = m.default
    }
    
    provider, ok := m.providers[name]
    if !ok {
        return nil, fmt.Errorf("storage provider %s not found", name)
    }
    
    return provider, nil
}

// GetUploadToken 获取上传凭证
func (m *Manager) GetUploadToken(ctx context.Context, providerName string, opts *UploadOptions) (*UploadToken, error) {
    provider, err := m.GetProvider(providerName)
    if err != nil {
        return nil, err
    }
    
    return provider.GetUploadToken(ctx, opts)
}

// Upload 上传文件
func (m *Manager) Upload(ctx context.Context, providerName string, reader io.Reader, opts *UploadOptions) (*FileInfo, error) {
    provider, err := m.GetProvider(providerName)
    if err != nil {
        return nil, err
    }
    
    return provider.Upload(ctx, reader, opts)
}

// 其他方法代理...
```

---

## 四、前端组件设计

### 📦 FileUpload Widget

```typescript
// web/src/core/widgets/FileUploadWidget.ts

import { h } from 'vue'
import { ElUpload, ElButton, ElIcon, ElMessage } from 'element-plus'
import { Upload, Delete, Download } from '@element-plus/icons-vue'
import type { UploadFile, UploadRawFile } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig } from '../types/field'
import { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import { api } from '@/api'

export interface FileItem {
  uid: string
  name: string
  size: number
  url: string
  type: string
  hash?: string
  uploadedAt?: string
}

export interface FileUploadConfig {
  accept?: string           // 接受的文件类型（如 "image/*", ".pdf"）
  multiple?: boolean        // 是否支持多文件上传
  max_count?: number        // 最大文件数量
  max_size?: number         // 最大文件大小（MB）
  storage_provider?: string // 存储提供者（minio/qiniu/local）
  list_type?: 'text' | 'picture' | 'picture-card'
  drag?: boolean            // 是否支持拖拽上传
}

export class FileUploadWidget extends BaseWidget {
  private config: FileUploadConfig
  private uploadToken: any = null
  private tokenExpire: Date | null = null

  constructor(
    field: FieldConfig,
    formManager: ReactiveFormDataManager | null,
    fieldPath: string
  ) {
    super(field, formManager, fieldPath)
    this.config = (field.widget?.config || {}) as FileUploadConfig
  }

  render() {
    const currentValue = this.getCurrentValue()
    const fileList = Array.isArray(currentValue?.raw) ? currentValue.raw : []

    return h('div', { class: 'file-upload-widget' }, [
      h(
        ElUpload,
        {
          action: '#', // 不使用默认上传
          fileList: this.convertToElFileList(fileList),
          accept: this.config.accept,
          multiple: this.config.multiple,
          limit: this.config.max_count,
          listType: this.config.list_type || 'text',
          drag: this.config.drag,
          httpRequest: this.handleUpload.bind(this),
          beforeUpload: this.beforeUpload.bind(this),
          onRemove: this.handleRemove.bind(this),
          onPreview: this.handlePreview.bind(this),
          onExceed: () => {
            ElMessage.warning(`最多只能上传 ${this.config.max_count} 个文件`)
          }
        },
        {
          default: () => this.renderUploadButton(),
          tip: () => this.renderTip()
        }
      )
    ])
  }

  private renderUploadButton() {
    if (this.config.list_type === 'picture-card') {
      return h(ElIcon, { class: 'el-icon--upload' }, () => h(Upload))
    }
    
    if (this.config.drag) {
      return h('div', { class: 'el-upload__text' }, [
        h(ElIcon, { class: 'el-icon--upload' }, () => h(Upload)),
        h('div', '将文件拖到此处，或'),
        h('em', '点击上传')
      ])
    }
    
    return h(
      ElButton,
      { type: 'primary', icon: Upload },
      () => '选择文件'
    )
  }

  private renderTip() {
    const tips: string[] = []
    
    if (this.config.accept) {
      tips.push(`支持格式：${this.config.accept}`)
    }
    if (this.config.max_size) {
      tips.push(`单个文件不超过 ${this.config.max_size}MB`)
    }
    if (this.config.max_count) {
      tips.push(`最多上传 ${this.config.max_count} 个文件`)
    }
    
    if (tips.length === 0) return null
    
    return h('div', { class: 'el-upload__tip' }, tips.join('，'))
  }

  // 上传前校验
  private async beforeUpload(file: UploadRawFile): Promise<boolean> {
    // 校验文件大小
    if (this.config.max_size) {
      const maxSizeBytes = this.config.max_size * 1024 * 1024
      if (file.size > maxSizeBytes) {
        ElMessage.error(`文件大小不能超过 ${this.config.max_size}MB`)
        return false
      }
    }
    
    // 校验文件类型
    if (this.config.accept) {
      const accept = this.config.accept.split(',').map(s => s.trim())
      const fileType = file.type
      const fileName = file.name
      
      const isAccepted = accept.some(type => {
        if (type.startsWith('.')) {
          // 扩展名匹配（如 ".pdf"）
          return fileName.toLowerCase().endsWith(type.toLowerCase())
        } else if (type.endsWith('/*')) {
          // MIME 类型通配符（如 "image/*"）
          const prefix = type.slice(0, -2)
          return fileType.startsWith(prefix)
        } else {
          // 精确 MIME 类型匹配
          return fileType === type
        }
      })
      
      if (!isAccepted) {
        ElMessage.error(`不支持的文件类型：${fileName}`)
        return false
      }
    }
    
    return true
  }

  // 自定义上传
  private async handleUpload(options: any) {
    const { file, onProgress, onSuccess, onError } = options
    
    try {
      // 1. 获取上传凭证
      const token = await this.getUploadToken(file)
      
      // 2. 上传文件
      const fileInfo = await this.uploadFile(file, token, onProgress)
      
      // 3. 更新表单数据
      const currentFiles = this.getCurrentFileList()
      currentFiles.push({
        uid: file.uid,
        name: file.name,
        size: file.size,
        url: fileInfo.url,
        type: file.type,
        hash: fileInfo.hash,
        uploadedAt: new Date().toISOString()
      })
      
      this.updateValue(currentFiles)
      onSuccess(fileInfo)
      
      ElMessage.success('上传成功')
    } catch (error: any) {
      onError(error)
      ElMessage.error(`上传失败：${error.message}`)
    }
  }

  // 获取上传凭证
  private async getUploadToken(file: File) {
    // 检查 token 是否过期
    if (this.uploadToken && this.tokenExpire && new Date() < this.tokenExpire) {
      return this.uploadToken
    }
    
    // 请求新的 token
    const response = await api.post('/storage/upload-token', {
      provider: this.config.storage_provider || 'minio',
      key: this.generateFileKey(file),
      content_type: file.type,
      max_size: file.size,
      expire: 3600 // 1 小时
    })
    
    this.uploadToken = response.data
    this.tokenExpire = new Date(response.data.expire)
    
    return this.uploadToken
  }

  // 上传文件（根据不同的存储提供者使用不同的上传方式）
  private async uploadFile(file: File, token: any, onProgress: (percent: number) => void) {
    if (token.method === 'PUT') {
      // MinIO 预签名 URL 上传
      return this.uploadViaPut(file, token, onProgress)
    } else if (token.method === 'POST') {
      // 七牛云表单上传
      return this.uploadViaPost(file, token, onProgress)
    } else {
      throw new Error(`Unsupported upload method: ${token.method}`)
    }
  }

  // PUT 方式上传（MinIO）
  private async uploadViaPut(file: File, token: any, onProgress: (percent: number) => void) {
    return new Promise<any>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          const percent = Math.round((e.loaded / e.total) * 100)
          onProgress(percent)
        }
      })
      
      xhr.addEventListener('load', () => {
        if (xhr.status === 200) {
          resolve({
            url: token.url.split('?')[0], // 去掉签名参数
            hash: xhr.getResponseHeader('ETag') || '',
            key: token.key
          })
        } else {
          reject(new Error(`Upload failed: ${xhr.statusText}`))
        }
      })
      
      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'))
      })
      
      xhr.open('PUT', token.url)
      
      // 设置自定义头
      if (token.headers) {
        Object.entries(token.headers).forEach(([key, value]) => {
          xhr.setRequestHeader(key, value as string)
        })
      }
      
      xhr.send(file)
    })
  }

  // POST 方式上传（七牛云）
  private async uploadViaPost(file: File, token: any, onProgress: (percent: number) => void) {
    const formData = new FormData()
    
    // 添加表单字段
    if (token.form_fields) {
      Object.entries(token.form_fields).forEach(([key, value]) => {
        formData.append(key, value as string)
      })
    }
    
    formData.append('file', file)
    
    return new Promise<any>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          const percent = Math.round((e.loaded / e.total) * 100)
          onProgress(percent)
        }
      })
      
      xhr.addEventListener('load', () => {
        if (xhr.status === 200) {
          const response = JSON.parse(xhr.responseText)
          resolve({
            url: `${token.cdn_url}/${response.key}`,
            hash: response.hash,
            key: response.key
          })
        } else {
          reject(new Error(`Upload failed: ${xhr.statusText}`))
        }
      })
      
      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'))
      })
      
      xhr.open('POST', token.url)
      xhr.send(formData)
    })
  }

  // 移除文件
  private handleRemove(file: UploadFile) {
    const currentFiles = this.getCurrentFileList()
    const index = currentFiles.findIndex(f => f.uid === file.uid)
    if (index > -1) {
      currentFiles.splice(index, 1)
      this.updateValue(currentFiles)
    }
  }

  // 预览文件
  private handlePreview(file: UploadFile) {
    const fileItem = this.getCurrentFileList().find(f => f.uid === file.uid)
    if (fileItem?.url) {
      window.open(fileItem.url, '_blank')
    }
  }

  // 生成文件存储路径
  private generateFileKey(file: File): string {
    const timestamp = Date.now()
    const random = Math.random().toString(36).substring(2, 8)
    const ext = file.name.split('.').pop()
    return `uploads/${this.fieldPath}/${timestamp}_${random}.${ext}`
  }

  // 获取当前文件列表
  private getCurrentFileList(): FileItem[] {
    const value = this.getCurrentValue()
    return Array.isArray(value?.raw) ? value.raw : []
  }

  // 更新值
  private updateValue(files: FileItem[]) {
    this.setValue({
      raw: files,
      display: files.map(f => f.name).join(', '),
      meta: {}
    })
  }

  // 转换为 Element Plus 的文件列表格式
  private convertToElFileList(files: FileItem[]): UploadFile[] {
    return files.map(file => ({
      uid: file.uid,
      name: file.name,
      url: file.url,
      status: 'success'
    }))
  }
}
```

---

## 五、后端 SDK 设计

### 📦 Files 类型定义

```go
// sdk/agent-app/model/files.go

package model

import (
    "context"
    "database/sql/driver"
    "encoding/json"
    "fmt"
    "io"
    "time"
)

// Files 文件集合类型
type Files struct {
    items []FileItem
    storage StorageClient // 存储客户端（用于自动上传）
}

// FileItem 单个文件信息
type FileItem struct {
    UID        string            `json:"uid"`
    Name       string            `json:"name"`
    Size       int64             `json:"size"`
    URL        string            `json:"url"`
    Type       string            `json:"type"`
    Hash       string            `json:"hash,omitempty"`
    UploadedAt string            `json:"uploaded_at,omitempty"`
    Metadata   map[string]string `json:"metadata,omitempty"`
}

// NewFiles 创建 Files 实例
func NewFiles(storage StorageClient) *Files {
    return &Files{
        items:   make([]FileItem, 0),
        storage: storage,
    }
}

// Add 添加文件
func (f *Files) Add(item FileItem) {
    f.items = append(f.items, item)
}

// Get 获取所有文件
func (f *Files) Get() []FileItem {
    return f.items
}

// First 获取第一个文件
func (f *Files) First() *FileItem {
    if len(f.items) == 0 {
        return nil
    }
    return &f.items[0]
}

// Count 文件数量
func (f *Files) Count() int {
    return len(f.items)
}

// IsEmpty 是否为空
func (f *Files) IsEmpty() bool {
    return len(f.items) == 0
}

// DownloadFirst 下载第一个文件到本地（用于后端处理）
func (f *Files) DownloadFirst(ctx context.Context, localPath string) error {
    if f.IsEmpty() {
        return fmt.Errorf("no files to download")
    }
    
    first := f.items[0]
    reader, err := f.storage.Download(ctx, first.URL)
    if err != nil {
        return err
    }
    defer reader.Close()
    
    // 写入本地文件
    file, err := os.Create(localPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = io.Copy(file, reader)
    return err
}

// UploadFile 上传文件（用于后端生成文件后上传）
func (f *Files) UploadFile(ctx context.Context, localPath string, filename string) error {
    file, err := os.Open(localPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    stat, err := file.Stat()
    if err != nil {
        return err
    }
    
    // 上传文件
    fileInfo, err := f.storage.Upload(ctx, file, &storage.UploadOptions{
        Key:         generateFileKey(filename),
        ContentType: getContentType(filename),
        MaxSize:     stat.Size(),
    })
    if err != nil {
        return err
    }
    
    // 添加到列表
    f.Add(FileItem{
        UID:        generateUID(),
        Name:       filename,
        Size:       fileInfo.Size,
        URL:        fileInfo.URL,
        Type:       fileInfo.ContentType,
        Hash:       fileInfo.Hash,
        UploadedAt: fileInfo.UploadedAt.Format(time.RFC3339),
    })
    
    return nil
}

// Scan 实现 sql.Scanner 接口（从数据库读取）
func (f *Files) Scan(value interface{}) error {
    if value == nil {
        f.items = make([]FileItem, 0)
        return nil
    }
    
    bytes, ok := value.([]byte)
    if !ok {
        return fmt.Errorf("failed to unmarshal Files value: %v", value)
    }
    
    return json.Unmarshal(bytes, &f.items)
}

// Value 实现 driver.Valuer 接口（写入数据库）
func (f Files) Value() (driver.Value, error) {
    if len(f.items) == 0 {
        return "[]", nil
    }
    
    bytes, err := json.Marshal(f.items)
    if err != nil {
        return nil, err
    }
    
    return string(bytes), nil
}

// MarshalJSON 实现 JSON 序列化
func (f Files) MarshalJSON() ([]byte, error) {
    return json.Marshal(f.items)
}

// UnmarshalJSON 实现 JSON 反序列化
func (f *Files) UnmarshalJSON(data []byte) error {
    return json.Unmarshal(data, &f.items)
}
```

---

### 使用示例

```go
// 场景1：后端处理文件（如视频转换）

package tools

import (
    "context"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/model"
)

type VideoTranscode struct {
    InputVideo  *model.Files `json:"input_video" runner:"name:输入视频" widget:"type:file_upload;accept:video/*"`
    Format      string       `json:"format" runner:"name:输出格式" widget:"type:select;options:mp4,avi,mkv,mov"`
    OutputVideo *model.Files `json:"output_video" runner:"name:输出视频" permission:"read"`
}

func (VideoTranscode) TableName() string { return "video_transcode" }

func init() {
    app.Register(&app.Function{
        Code: "video_transcode",
        Name: "视频转码",
        OnSubmit: func(ctx context.Context, req *VideoTranscode) (*VideoTranscode, error) {
            // 1. 下载输入视频
            inputPath := "/tmp/input.mp4"
            if err := req.InputVideo.DownloadFirst(ctx, inputPath); err != nil {
                return nil, err
            }
            
            // 2. 调用 FFmpeg 转码
            outputPath := "/tmp/output." + req.Format
            if err := transcode(inputPath, outputPath, req.Format); err != nil {
                return nil, err
            }
            
            // 3. 上传输出视频
            req.OutputVideo = model.NewFiles(app.GetStorage())
            if err := req.OutputVideo.UploadFile(ctx, outputPath, "output."+req.Format); err != nil {
                return nil, err
            }
            
            return req, nil
        },
    })
}
```

```go
// 场景2：后端只存储元数据（如工单附件）

type Ticket struct {
    ID          int          `json:"id" gorm:"primaryKey"`
    Title       string       `json:"title" runner:"name:标题" widget:"type:input"`
    Description string       `json:"description" runner:"name:描述" widget:"type:textarea"`
    Attachments *model.Files `json:"attachments" runner:"name:附件" widget:"type:file_upload;multiple:true;max_count:5"`
}

func (Ticket) TableName() string { return "tickets" }

// 后端无需处理文件，Files 会自动序列化/反序列化
```

---

## 六、部署配置

### 🔧 配置文件示例

```yaml
# configs/storage.yaml

storage:
  # 默认存储提供者
  default: minio
  
  # MinIO 配置（推荐用于私有化部署）
  minio:
    endpoint: localhost:9000
    access_key: minioadmin
    secret_key: minioadmin
    bucket: ai-agent-os
    region: us-east-1
    use_ssl: false
    
  # 七牛云配置（推荐用于云端部署）
  qiniu:
    access_key: your_access_key
    secret_key: your_secret_key
    bucket: ai-agent-os
    domain: https://cdn.yourdomain.com
    
  # 本地文件系统（开发环境）
  local:
    base_path: /data/uploads
    base_url: http://localhost:8080/files
```

---

## 七、安全性设计

### 🛡️ 核心安全措施

#### 1. 上传凭证签名
```go
// 防止恶意上传，所有上传必须先获取签名凭证
token, err := storage.GetUploadToken(ctx, &storage.UploadOptions{
    Key:     generateSecureKey(), // 服务端生成 key
    Expire:  time.Hour,            // 限制有效期
    MaxSize: 100 * 1024 * 1024,   // 限制文件大小
})
```

#### 2. 文件类型校验
```go
// 前端校验（用户体验）
if !isAcceptedType(file.type, acceptTypes) {
    return error("不支持的文件类型")
}

// 后端校验（安全）
if !isValidFileType(uploadedFile) {
    return error("文件类型校验失败")
}
```

#### 3. 病毒扫描（可选，企业版）
```go
// 上传后自动扫描
if config.EnableVirusScan {
    if infected, err := virusScanner.Scan(filePath); infected {
        storage.Delete(ctx, fileKey)
        return error("检测到恶意文件")
    }
}
```

#### 4. 访问权限控制
```go
// 临时签名 URL（私有文件）
url, err := storage.GetURL(ctx, fileKey, time.Hour)

// 公开 URL（公共文件）
url, err := storage.GetURL(ctx, fileKey, 0)
```

---

## 八、推荐方案总结

### ✅ 最佳实践

| 场景 | 推荐方案 | 原因 |
|------|----------|------|
| **私有化部署** | MinIO | 开源、兼容 S3、易部署 |
| **云端部署** | 七牛云/阿里云 OSS | CDN 加速、稳定可靠 |
| **开发环境** | 本地文件系统 | 简单快速 |
| **企业版** | 支持多种存储 | 灵活配置 |

---

### 📋 实施优先级

1. **Phase 1**（核心，2-3 周）
   - ✅ 实现存储抽象层（Provider 接口）
   - ✅ 实现 MinIO Provider（私有化部署）
   - ✅ 实现本地 Provider（开发环境）
   - ✅ 实现 FileUpload Widget（前端）
   - ✅ 实现 Files 类型（后端 SDK）

2. **Phase 2**（扩展，1-2 周）
   - ✅ 实现七牛云 Provider
   - ✅ 实现阿里云 OSS Provider
   - ✅ 添加文件预览功能
   - ✅ 添加图片裁剪功能

3. **Phase 3**（优化，按需）
   - ✅ 病毒扫描
   - ✅ 图片压缩
   - ✅ 水印添加
   - ✅ 文件转换服务

---

## 九、总结

### 核心设计思想

1. **抽象优先** - Provider 接口抽象存储细节
2. **职责分离** - 前端上传、后端处理、存储服务独立
3. **灵活配置** - 支持多种存储后端，一键切换
4. **安全第一** - 签名上传、类型校验、权限控制

### 关键优势

- ✅ 支持私有化部署（MinIO）
- ✅ 支持云端部署（七牛云、阿里云）
- ✅ 支持多种存储后端，可扩展
- ✅ 前后端解耦，易维护
- ✅ 安全可靠，符合企业标准

这个设计方案可以满足你项目的所有需求，既支持云端部署，也支持私有化部署，还为未来的扩展留足了空间！🚀

