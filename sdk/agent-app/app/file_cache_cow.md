# 文件缓存写时复制（Copy-on-Write）机制

## 📚 问题场景

当多个请求需要修改同一个文件时，如果使用硬链接，修改一个文件会影响所有硬链接：

```
缓存文件：/app/workplace/file-cache/abc123...def456  (100MB)
请求1：   /app/workplace/uploads/trace1/file.pdf  → 硬链接
请求2：   /app/workplace/uploads/trace2/file.pdf  → 硬链接
请求3：   /app/workplace/uploads/trace3/file.pdf  → 硬链接

❌ 问题：如果请求1修改了文件，请求2和请求3的文件也会被修改！
```

## ✅ 解决方案：内核级写时复制（Copy-on-Write）

我们提供了两种方法：

### 1. `GetOrDownload` - 只读场景（使用硬链接）

**适用场景**：只需要读取文件，不需要修改

**优点**：
- 极快的创建速度（微秒级）
- 不占用额外磁盘空间
- 多个请求共享同一份数据

**缺点**：
- 修改会影响所有硬链接

**使用示例**：
```go
// 只读场景：使用硬链接
localPath, fromCache, err := fileCache.GetOrDownload(ctx, hash, downloadURL, targetPath)
if err != nil {
    return err
}

// 读取文件
data, err := os.ReadFile(localPath)
// 注意：不要修改这个文件，否则会影响其他硬链接
```

### 2. `GetOrDownloadForWrite` - 修改场景（使用内核级 reflink）

**适用场景**：需要修改文件

**优点**：
- **内核级 COW**：使用 Linux reflink（FICLONE ioctl），性能接近硬链接（微秒级）
- 每个请求获得独立副本，可以安全修改
- 修改不会影响其他请求
- **写时复制**：只有在实际写入时才复制数据块，节省磁盘空间
- 自动回退：如果文件系统不支持 reflink，自动回退到普通复制

**缺点**：
- 需要文件系统支持 reflink（Btrfs、XFS、OCFS2）
- 如果不支持，会回退到普通复制（时间取决于文件大小）

**使用示例**：
```go
// 修改场景：使用写时复制
localPath, fromCache, err := fileCache.GetOrDownloadForWrite(ctx, hash, downloadURL, targetPath)
if err != nil {
    return err
}

// 可以安全修改文件
err = os.WriteFile(localPath, newData, 0644)
// 或者使用其他方式修改文件
// 修改不会影响其他请求的文件
```

## 🔄 工作流程

### 只读场景（硬链接）

```
1. 检查缓存是否存在
   ├─ 存在 → 创建硬链接（微秒级）
   └─ 不存在 → 下载到缓存 → 创建硬链接

2. 多个请求共享同一份数据
   ├─ 请求1：硬链接 → 缓存文件
   ├─ 请求2：硬链接 → 缓存文件
   └─ 请求3：硬链接 → 缓存文件

3. 磁盘空间：只有1份文件（100MB）
```

### 修改场景（内核级 reflink）

```
1. 检查缓存是否存在
   ├─ 存在 → 尝试 reflink（微秒级，内核级 COW）
   │   ├─ 成功 → 创建 reflink（写时复制，共享数据块）
   │   └─ 失败 → 回退到普通复制（时间取决于文件大小）
   └─ 不存在 → 下载到缓存 → 尝试 reflink

2. 每个请求获得独立副本（reflink）
   ├─ 请求1：reflink（可修改，写时复制）
   ├─ 请求2：reflink（可修改，写时复制）
   └─ 请求3：reflink（可修改，写时复制）

3. 磁盘空间：
   - reflink 成功：1份缓存 + 共享数据块（接近 100MB，只有修改的部分才复制）
   - reflink 失败：1份缓存 + N份副本（100MB + N×100MB）
```

## 📊 性能对比

| 场景 | 方法 | 创建时间 | 磁盘空间（3个请求） | 修改影响 |
|------|------|----------|-------------------|----------|
| 只读 | `GetOrDownload` | < 1ms | 100MB | ❌ 会影响所有硬链接 |
| 修改（reflink） | `GetOrDownloadForWrite` | < 1ms（内核级） | ~100MB（共享数据块，只有修改部分才复制） | ✅ 独立修改 |
| 修改（回退） | `GetOrDownloadForWrite` | 100-500ms（文件系统不支持 reflink） | 400MB（100MB缓存 + 3×100MB副本） | ✅ 独立修改 |

## 🎯 使用建议

1. **默认使用 `GetOrDownload`**（只读场景）
   - 如果只是读取文件，使用硬链接可以节省空间和时间

2. **需要修改时使用 `GetOrDownloadForWrite`**（修改场景）
   - 如果需要对文件进行任何修改操作，使用写时复制

3. **混合场景**
   - 可以先使用 `GetOrDownload` 读取文件
   - 如果需要修改，再调用 `GetOrDownloadForWrite` 获取独立副本

## 🔧 文件系统支持

### reflink 支持的文件系统

- ✅ **Btrfs**：完全支持 reflink（推荐）
- ✅ **XFS**：完全支持 reflink（推荐）
- ✅ **OCFS2**：支持 reflink
- ❌ **ext4**：不支持 reflink（会回退到普通复制）
- ❌ **其他文件系统**：不支持 reflink（会回退到普通复制）

### 自动回退机制

如果文件系统不支持 reflink，`GetOrDownloadForWrite` 会自动回退到普通文件复制（`io.Copy`），确保功能正常工作。

### 检查文件系统类型

```bash
# 在容器中检查文件系统类型
df -T /app/workplace/file-cache
```

如果显示 `btrfs` 或 `xfs`，则支持 reflink，性能最佳。

## 💡 实现细节

### 内核级 reflink（FICLONE）

使用 Linux `ioctl` 系统调用 `FICLONE` 创建 reflink：

```go
// FICLONE = 0x40049409 (Linux ioctl number)
const FICLONE = 0x40049409

// 调用 ioctl: ioctl(dst_fd, FICLONE, src_fd)
syscall.Syscall(syscall.SYS_IOCTL, uintptr(dstFd), uintptr(FICLONE), uintptr(srcFd))
```

**工作原理**：
1. 创建时：reflink 和源文件共享相同的数据块（类似硬链接）
2. 写入时：内核自动复制被修改的数据块（写时复制）
3. 结果：只有实际修改的部分占用额外空间

### 引用计数

- **硬链接**：参与引用计数，删除硬链接时减少引用计数
- **reflink/独立副本**：不参与引用计数，直接删除即可

### 清理机制

- **硬链接**：当所有硬链接都被删除且引用计数为0时，标记为延迟删除
- **reflink/独立副本**：直接删除，不影响缓存文件

### 内存映射

- `pathIsCopy`：记录每个文件路径是否为独立副本（reflink 或普通复制）
- `pathToHash`：记录每个文件路径对应的hash
- `refCount`：记录缓存文件的硬链接数量

