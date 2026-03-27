<template>
  <div class="board-view" v-loading="loading">
    <!-- ⭐ 权限不足时显示申请权限（与表格/表单等一致） -->
    <PermissionDeniedView v-if="permissionError" />

    <div v-show="!permissionError" class="board-main">
    <div class="board-header">
      <h1 class="board-title">{{ node?.name || '讨论区' }}</h1>
      <el-button type="primary" :icon="Plus" @click="showCreateForm = true">发帖</el-button>
    </div>

    <!-- 搜索（仅列表页显示） -->
    <div v-if="!selectedPostId" class="board-search">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索帖子标题…"
        clearable
        class="board-search-input"
        :prefix-icon="Search"
      />
    </div>

    <!-- 帖子列表：标题为主、用户信息弱化、摘要与标题分区、封面缩略 -->
    <div class="post-list board-feed" v-if="!selectedPostId">
      <div
        v-for="row in filteredPosts"
        :key="row.id"
        class="board-card"
        @click="openPost(row.id)"
      >
        <!-- 用户信息：单行小字，不抢眼 -->
        <div class="board-card-meta">
          <el-avatar :size="24" :src="authorAvatar(row.author)" class="board-card-avatar">
            {{ authorInitial(row.author) }}
          </el-avatar>
          <span class="board-card-meta-text">{{ authorDisplayName(row.author) }} · {{ formatTime(row.created_at) }}</span>
        </div>
        <!-- 标题：突出显示 -->
        <h2 class="board-card-title">{{ row.title }}</h2>
        <!-- 摘要：与标题留白区分 -->
        <p v-if="row.summary" class="board-card-summary">{{ row.summary }}</p>
        <!-- 封面：缩略尺寸 -->
        <div
          v-if="row.cover && (row.cover as string[]).length"
          :class="['board-card-covers', 'board-card-covers-' + Math.min((row.cover as string[]).length, 9)]"
        >
          <img
            v-for="(url, i) in (row.cover as string[]).slice(0, 9)"
            :key="i"
            :src="url"
            alt=""
            class="board-card-cover-img"
          />
        </div>
        <div class="board-card-actions" @click.stop>
          <el-button link type="primary" size="small" @click="openPost(row.id)">查看</el-button>
          <el-button link type="danger" size="small" @click="handleDeletePost(row)">删除</el-button>
        </div>
      </div>
      <el-empty
        v-if="filteredPosts.length === 0 && !loading"
        :description="searchKeyword.trim() ? '未找到匹配的帖子' : '暂无帖子'"
        :image-size="80"
        class="post-list-empty"
      />
      <el-pagination
        v-if="total > pageSize"
        class="post-pagination"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="onPageChange"
      />
    </div>

    <!-- 帖子详情（居中阅读区 + 作者/标题/封面/正文，支持 Markdown） -->
    <div v-else class="post-detail">
      <div class="post-detail-header">
        <el-button link :icon="ArrowLeft" @click="selectedPostId = null">返回列表</el-button>
        <el-button v-if="postDetail" link type="primary" @click="openEditForm">编辑</el-button>
      </div>
      <div v-if="postDetail" class="post-detail-body post-detail-body-centered">
        <!-- 作者区 -->
        <div class="post-detail-author">
          <el-avatar :size="44" :src="authorAvatar(postDetail.author)" class="detail-avatar">{{ authorInitial(postDetail.author) }}</el-avatar>
          <div class="post-detail-user-block">
            <div class="post-detail-name-row">
              <span class="post-detail-name">{{ authorDisplayName(postDetail.author) }}</span>
              <span class="post-detail-time">
                创建于 {{ formatTime(postDetail.created_at) }}
                <template v-if="postDetail.updated_at && postDetail.updated_at !== postDetail.created_at"> · 更新于 {{ formatTime(postDetail.updated_at) }}</template>
              </span>
            </div>
            <span v-if="getAuthorInfo(postDetail.author)?.nickname" class="post-detail-username">@{{ postDetail.author }}</span>
            <span v-if="getAuthorInfo(postDetail.author)?.signature" class="post-detail-signature">{{ getAuthorInfo(postDetail.author)!.signature }}</span>
          </div>
        </div>
        <!-- 标题 -->
        <h1 class="post-detail-title">{{ postDetail.title }}</h1>
        <!-- 摘要（若有） -->
        <p v-if="postDetail.summary" class="post-detail-summary">{{ postDetail.summary }}</p>
        <!-- 封面九宫格 -->
        <div
          v-if="postDetail.cover && postDetail.cover.length"
          :class="['post-detail-covers', 'moments-photos-' + Math.min(postDetail.cover.length, 9)]"
        >
          <img
            v-for="(url, i) in postDetail.cover.slice(0, 9)"
            :key="i"
            :src="url"
            alt=""
            class="post-detail-cover-img"
          />
        </div>
        <!-- 正文（居中、限制宽度、Markdown 样式完善） -->
        <div class="post-detail-content markdown-content" v-html="renderedContent" />
      </div>
    </div>

    <!-- 发帖对话框（与 docs 一致的弹性布局，内容区自适应不溢出） -->
    <el-dialog v-model="showCreateForm" title="发帖" width="720" @close="resetCreateForm" class="board-create-dialog board-dialog-with-editor">
      <el-form :model="createForm" label-width="80px" class="board-form-in-dialog">
        <el-form-item label="封面（可多图）" class="board-form-item-static">
          <el-upload
            class="cover-uploader"
            :file-list="coverFileList"
            :http-request="handleCoverUpload"
            :show-file-list="true"
            :limit="9"
            accept="image/*"
            list-type="picture-card"
            multiple
            :on-remove="handleCoverRemove"
          >
            <el-icon class="cover-uploader-icon"><Plus /></el-icon>
          </el-upload>
          <div v-if="coverUploading" class="cover-uploading-tip">封面上传中…</div>
        </el-form-item>
        <el-form-item label="标题" required class="board-form-item-static">
          <el-input v-model="createForm.title" placeholder="请输入标题" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="摘要" class="board-form-item-static">
          <el-input v-model="createForm.summary" type="textarea" :rows="2" placeholder="选填，列表展示；不填则从正文自动截取" maxlength="500" show-word-limit />
        </el-form-item>
        <el-form-item label="内容" class="board-form-item-editor">
          <div class="board-dialog-editor-wrap">
            <VditorEditor
              v-model="createForm.content"
              height="100%"
              placeholder="请输入正文（支持 Markdown），支持拖拽/粘贴上传"
              class="board-vditor-editor"
            />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateForm = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmitPost">发布</el-button>
      </template>
    </el-dialog>

    <!-- 编辑帖子对话框（与 docs 一致的弹性布局） -->
    <el-dialog v-model="showEditForm" title="编辑帖子" width="720" @close="resetEditForm" class="board-edit-dialog board-dialog-with-editor">
      <el-form v-if="postDetail" :model="editForm" label-width="80px" class="board-form-in-dialog">
        <el-form-item label="封面（可多图）" class="board-form-item-static">
          <el-upload
            class="cover-uploader"
            :file-list="editCoverFileList"
            :http-request="handleEditCoverUpload"
            :show-file-list="true"
            :limit="9"
            accept="image/*"
            list-type="picture-card"
            multiple
            :on-remove="handleEditCoverRemove"
          >
            <el-icon class="cover-uploader-icon"><Plus /></el-icon>
          </el-upload>
          <div v-if="editCoverUploading" class="cover-uploading-tip">封面上传中…</div>
        </el-form-item>
        <el-form-item label="标题" required class="board-form-item-static">
          <el-input v-model="editForm.title" placeholder="请输入标题" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="摘要" class="board-form-item-static">
          <el-input v-model="editForm.summary" type="textarea" :rows="2" placeholder="选填，不填则从正文自动截取" maxlength="500" show-word-limit />
        </el-form-item>
        <el-form-item label="内容" class="board-form-item-editor">
          <div class="board-dialog-editor-wrap">
            <VditorEditor
              v-model="editForm.content"
              height="100%"
              placeholder="请输入正文（支持 Markdown），支持拖拽/粘贴上传"
              class="board-vditor-editor"
            />
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditForm = false">取消</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="handleSubmitEdit">保存</el-button>
      </template>
    </el-dialog>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, ArrowLeft, Search } from '@element-plus/icons-vue'
import { marked } from 'marked'
import type { ServiceTree, UserInfo } from '@/types'
import { listPosts, getPost, createPost, updatePost, deletePost, type PostItem, type GetPostResp } from '@/api/board'
import VditorEditor from '@/components/VditorEditor.vue'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import { useAuthStore } from '@/stores/auth'
import { useUserInfoStore } from '@/stores/userInfo'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { escapeHtml, sanitizeHtml } from '@/utils/sanitizeHtml'
import PermissionDeniedView from './PermissionDeniedView.vue'

const BOARD_COVER_UPLOAD_ROUTER = 'board/cover'

interface Props {
  node: ServiceTree
}

const props = defineProps<Props>()

const userInfoStore = useUserInfoStore()
const permissionErrorStore = usePermissionErrorStore()
const permissionError = computed(() => permissionErrorStore.currentError)
const loading = ref(false)
const posts = ref<PostItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const fullCodePath = computed(() => props.node?.full_code_path || '')
/** 搜索关键词（按标题过滤） */
const searchKeyword = ref('')
/** 当前页帖子按关键词过滤后的列表 */
const filteredPosts = computed(() => {
  const kw = searchKeyword.value.trim().toLowerCase()
  if (!kw) return posts.value
  return posts.value.filter((p) => (p.title || '').toLowerCase().includes(kw))
})
/** 帖子作者 username -> UserInfo，用于展示头像、昵称 */
const authorInfoMap = ref<Record<string, UserInfo>>({})

const loadPosts = async () => {
  if (!fullCodePath.value) return
  loading.value = true
  try {
    const res = await listPosts({ full_code_path: fullCodePath.value, page: page.value, page_size: pageSize })
    posts.value = res.list || []
    total.value = res.total || 0
    permissionErrorStore.clearError()
    const authors = [...new Set((res.list || []).map((p) => p.author).filter(Boolean))]
    if (authors.length) {
      const users = await userInfoStore.batchGetUserInfo(authors)
      const map: Record<string, UserInfo> = {}
      users.forEach((u) => {
        map[u.username] = u
      })
      authorInfoMap.value = map
    } else {
      authorInfoMap.value = {}
    }
  } catch (e: any) {
    const status = e?.response?.status
    if (status === 403) {
      // 403 已由 request 拦截器写入 permissionErrorStore，会显示 PermissionDeniedView，不重复弹窗
      return
    }
    ElMessage.error('加载帖子列表失败: ' + (e?.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

/** 取作者展示信息（昵称优先，无则用户名） */
const getAuthorInfo = (author: string) => authorInfoMap.value[author] ?? null
const authorDisplayName = (author: string) => getAuthorInfo(author)?.nickname || author || '?'
const authorAvatar = (author: string) => getAuthorInfo(author)?.avatar || ''
const authorInitial = (author: string) => (authorDisplayName(author) || '?').charAt(0).toUpperCase()

const selectedPostId = ref<number | null>(null)
const postDetail = ref<GetPostResp | null>(null)
const loadPostDetail = async (id: number) => {
  try {
    postDetail.value = await getPost(id)
    const author = postDetail.value?.author
    if (author && !authorInfoMap.value[author]) {
      const user = await userInfoStore.getUserInfo(author)
      if (user) {
        authorInfoMap.value = { ...authorInfoMap.value, [author]: user }
      }
    }
  } catch (e: any) {
    if (e?.response?.status === 403) return
    ElMessage.error('加载帖子失败: ' + (e?.message || '未知错误'))
  }
}

watch(
  () => props.node?.full_code_path,
  (path) => {
    if (path) {
      permissionErrorStore.clearError()
      page.value = 1
      selectedPostId.value = null
      loadPosts()
    }
  },
  { immediate: true }
)

watch(selectedPostId, (id) => {
  if (id) loadPostDetail(id)
  else postDetail.value = null
})

const openPost = (id: number) => {
  selectedPostId.value = id
}

const formatTime = (t: string) => {
  if (!t) return ''
  try {
    const d = new Date(t)
    return d.toLocaleString('zh-CN')
  } catch {
    return t
  }
}

const onPageChange = (p: number) => {
  page.value = p
  loadPosts()
}

// 详情富文本渲染（markdown -> html）
const renderedContent = computed(() => {
  if (!postDetail.value?.content) return '（无正文）'
  if (postDetail.value.content_format === 'html') return sanitizeHtml(postDetail.value.content)
  try {
    return sanitizeHtml(marked.parse(postDetail.value.content) as string)
  } catch {
    return escapeHtml(postDetail.value.content).replace(/\n/g, '<br>')
  }
})

// 发帖：封面 + 富文本
const showCreateForm = ref(false)
const submitting = ref(false)
const coverUploading = ref(false)
const createForm = ref<{ title: string; summary: string; cover: string[]; content: string; content_format: string }>({
  title: '',
  summary: '',
  cover: [],
  content: '',
  content_format: 'markdown'
})

const coverFileList = computed(() =>
  createForm.value.cover.map((url, i) => ({ name: `封面${i + 1}`, url }))
)

const resetCreateForm = () => {
  createForm.value = { title: '', summary: '', cover: [], content: '', content_format: 'markdown' }
}

const handleCoverRemove = (_file: any, fileList: { url?: string }[]) => {
  createForm.value.cover = fileList.map((f) => f.url).filter(Boolean) as string[]
}

const handleCoverUpload = async (options: { file: File }) => {
  const file = options.file
  if (!file || !file.type.startsWith('image/')) {
    ElMessage.warning('请选择图片作为封面')
    return
  }
  coverUploading.value = true
  try {
    const uploadResult = await uploadFile(BOARD_COVER_UPLOAD_ROUTER, file, () => {})
    if (!uploadResult.fileInfo) throw new Error('封面上传失败')
    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
      upload_user: useAuthStore().userName || undefined
    })
    if (completeResult?.download_url) {
      createForm.value.cover = [...createForm.value.cover, completeResult.download_url]
      ElMessage.success('封面上传成功')
    } else {
      throw new Error('获取封面地址失败')
    }
  } catch (e: any) {
    ElMessage.error('封面上传失败: ' + (e?.message || '未知错误'))
  } finally {
    coverUploading.value = false
  }
}

const handleSubmitPost = async () => {
  if (!createForm.value.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  if (!fullCodePath.value) {
    ElMessage.error('版块路径不存在')
    return
  }
  submitting.value = true
  try {
    await createPost({
      full_code_path: fullCodePath.value,
      title: createForm.value.title.trim(),
      summary: createForm.value.summary?.trim() || undefined,
      cover: createForm.value.cover.length ? createForm.value.cover : undefined,
      content: createForm.value.content?.trim() || '',
      content_format: createForm.value.content_format || 'markdown',
      status: 'published'
    })
    ElMessage.success('发布成功')
    showCreateForm.value = false
    resetCreateForm()
    loadPosts()
  } catch (e: any) {
    ElMessage.error('发帖失败: ' + (e?.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

const handleDeletePost = async (row: PostItem) => {
  try {
    await ElMessageBox.confirm('确定删除该帖子？', '提示', { type: 'warning' })
    await deletePost(row.id)
    ElMessage.success('已删除')
    if (selectedPostId.value === row.id) selectedPostId.value = null
    loadPosts()
  } catch (e: any) {
    if (e !== 'cancel') ElMessage.error('删除失败: ' + (e?.message || '未知错误'))
  }
}

// 编辑帖子
const showEditForm = ref(false)
const editSubmitting = ref(false)
const editCoverUploading = ref(false)
const editForm = ref<{ title: string; summary: string; cover: string[]; content: string; content_format: string }>({
  title: '',
  summary: '',
  cover: [],
  content: '',
  content_format: 'markdown'
})
const editCoverFileList = computed(() =>
  editForm.value.cover.map((url, i) => ({ name: `封面${i + 1}`, url }))
)
const openEditForm = () => {
  if (!postDetail.value) return
  editForm.value = {
    title: postDetail.value.title,
    summary: postDetail.value.summary || '',
    cover: [...(postDetail.value.cover || [])],
    content: postDetail.value.content || '',
    content_format: postDetail.value.content_format || 'markdown'
  }
  showEditForm.value = true
}
const resetEditForm = () => {
  editForm.value = { title: '', summary: '', cover: [], content: '', content_format: 'markdown' }
}
const handleEditCoverRemove = (_file: any, fileList: { url?: string }[]) => {
  editForm.value.cover = fileList.map((f) => f.url).filter(Boolean) as string[]
}
const handleEditCoverUpload = async (options: { file: File }) => {
  const file = options.file
  if (!file || !file.type.startsWith('image/')) {
    ElMessage.warning('请选择图片作为封面')
    return
  }
  editCoverUploading.value = true
  try {
    const uploadResult = await uploadFile(BOARD_COVER_UPLOAD_ROUTER, file, () => {})
    if (!uploadResult.fileInfo) throw new Error('封面上传失败')
    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
      upload_user: useAuthStore().userName || undefined
    })
    if (completeResult?.download_url) {
      editForm.value.cover = [...editForm.value.cover, completeResult.download_url]
      ElMessage.success('封面上传成功')
    } else {
      throw new Error('获取封面地址失败')
    }
  } catch (e: any) {
    ElMessage.error('封面上传失败: ' + (e?.message || '未知错误'))
  } finally {
    editCoverUploading.value = false
  }
}
const handleSubmitEdit = async () => {
  if (!postDetail.value || !editForm.value.title.trim()) {
    ElMessage.warning('请输入标题')
    return
  }
  editSubmitting.value = true
  try {
    const updated = await updatePost(postDetail.value.id, {
      title: editForm.value.title.trim(),
      summary: editForm.value.summary?.trim() || undefined,
      cover: editForm.value.cover.length ? editForm.value.cover : undefined,
      content: editForm.value.content?.trim() ?? '',
      content_format: editForm.value.content_format || 'markdown'
    })
    postDetail.value = updated
    ElMessage.success('保存成功')
    showEditForm.value = false
    resetEditForm()
    loadPosts()
  } catch (e: any) {
    ElMessage.error('保存失败: ' + (e?.message || '未知错误'))
  } finally {
    editSubmitting.value = false
  }
}
</script>

<style scoped>
.board-view {
  padding: 16px;
  min-height: 200px;
}
.board-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.board-title {
  margin: 0;
  font-size: 18px;
}
.board-search {
  margin-bottom: 12px;
}
.board-search-input {
  max-width: 320px;
}
.post-list-empty {
  padding: 24px 0;
}
/* 讨论区列表：标题为主、用户信息弱化、摘要与标题分区、封面缩略 */
.post-list.board-feed {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 0;
  width: 100%;
}
.board-card {
  padding: 14px 16px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
  transition: background 0.2s;
}
.board-card:hover {
  background: var(--el-fill-color-light);
}
.board-card:last-of-type {
  border-bottom: none;
}
/* 用户信息：单行小字 */
.board-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.board-card-avatar {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 500;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}
.board-card-meta-text {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
/* 标题：突出 */
.board-card-title {
  margin: 0 0 8px;
  font-size: 17px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--el-text-color-primary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
/* 摘要：与标题留白，不连在一起 */
.board-card-summary {
  margin: 0 0 10px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
/* 封面：缩略尺寸，不占大块 */
.board-card-covers {
  display: grid;
  gap: 4px;
  margin-bottom: 8px;
  border-radius: 6px;
  overflow: hidden;
}
.board-card-covers-1 {
  grid-template-columns: 1fr;
  max-width: 120px;
  max-height: 120px;
}
.board-card-covers-2 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 160px;
}
.board-card-covers-3 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 180px;
}
.board-card-covers-4 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 160px;
}
.board-card-covers-5,
.board-card-covers-6,
.board-card-covers-7,
.board-card-covers-8,
.board-card-covers-9 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 180px;
}
.board-card-cover-img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  display: block;
  background: var(--el-fill-color-lighter);
}
.board-card-covers-1 .board-card-cover-img {
  max-height: 120px;
  aspect-ratio: auto;
}
.board-card-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.post-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}
/* ---------- 帖子详情页 ---------- */
.post-detail {
  width: 100%;
}
.post-detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.post-detail-body {
  padding: 20px 0;
  max-width: 720px;
}
/* 详情内容区居中，阅读更舒适 */
.post-detail-body-centered {
  margin-left: auto;
  margin-right: auto;
  max-width: 680px;
  padding: 24px 20px 40px;
}
.post-detail-summary {
  margin: 0 0 20px;
  font-size: 15px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
/* 作者区（与列表一致） */
.post-detail-author {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 20px;
}
.post-detail-author .detail-avatar {
  flex-shrink: 0;
  font-size: 17px;
  font-weight: 600;
  background: linear-gradient(135deg, #7dd3fc 0%, #38bdf8 100%);
}
.post-detail-user-block {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.post-detail-name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.post-detail-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.post-detail-time {
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}
.post-detail-username {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.post-detail-signature {
  font-size: 13px;
  color: var(--el-text-color-placeholder);
  line-height: 1.4;
}
/* 标题 */
.post-detail-title {
  margin: 0 0 20px;
  font-size: 22px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}
/* 详情封面九宫格（与列表一致，略大） */
.post-detail-covers {
  display: grid;
  gap: 8px;
  margin-bottom: 24px;
  border-radius: 12px;
  overflow: hidden;
}
.post-detail-covers.moments-photos-1 {
  grid-template-columns: 1fr;
  max-width: 360px;
  max-height: 360px;
}
.post-detail-covers.moments-photos-2 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 360px;
}
.post-detail-covers.moments-photos-3 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 360px;
}
.post-detail-covers.moments-photos-4 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 360px;
}
.post-detail-covers.moments-photos-5,
.post-detail-covers.moments-photos-6,
.post-detail-covers.moments-photos-7,
.post-detail-covers.moments-photos-8,
.post-detail-covers.moments-photos-9 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 360px;
}
.post-detail-cover-img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  display: block;
  background: var(--el-fill-color-lighter);
}
.post-detail-covers.moments-photos-1 .post-detail-cover-img {
  max-height: 360px;
  aspect-ratio: auto;
}
/* 正文：居中阅读 + 完善 Markdown 样式 */
.post-detail-content {
  line-height: 1.8;
  color: var(--el-text-color-regular);
  word-break: break-word;
}
.post-detail-content.markdown-content :deep(h1),
.post-detail-content.markdown-content :deep(h2),
.post-detail-content.markdown-content :deep(h3),
.post-detail-content.markdown-content :deep(h4) {
  margin-top: 1.5em;
  margin-bottom: 0.5em;
  color: var(--el-text-color-primary);
  font-weight: 600;
}
.post-detail-content.markdown-content :deep(h1) { font-size: 1.5em; }
.post-detail-content.markdown-content :deep(h2) { font-size: 1.3em; }
.post-detail-content.markdown-content :deep(h3) { font-size: 1.15em; }
.post-detail-content.markdown-content :deep(p) {
  margin-bottom: 1em;
}
.post-detail-content.markdown-content :deep(pre),
.post-detail-content.markdown-content :deep(code) {
  background: var(--el-fill-color-light);
  border-radius: 6px;
  font-family: ui-monospace, monospace;
}
.post-detail-content.markdown-content :deep(pre) {
  padding: 14px 16px;
  overflow-x: auto;
  margin: 1em 0;
}
.post-detail-content.markdown-content :deep(code) {
  padding: 0.2em 0.4em;
  font-size: 0.9em;
}
.post-detail-content.markdown-content :deep(pre code) {
  padding: 0;
  background: transparent;
}
.post-detail-content.markdown-content :deep(blockquote) {
  margin: 1em 0;
  padding: 0.5em 0 0.5em 1em;
  border-left: 4px solid var(--el-border-color);
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
  border-radius: 0 6px 6px 0;
}
.post-detail-content.markdown-content :deep(ul),
.post-detail-content.markdown-content :deep(ol) {
  margin: 0.75em 0;
  padding-left: 1.5em;
}
.post-detail-content.markdown-content :deep(li) {
  margin: 0.25em 0;
}
.post-detail-content.markdown-content :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 1em 0;
}
.post-detail-content.markdown-content :deep(th),
.post-detail-content.markdown-content :deep(td) {
  border: 1px solid var(--el-border-color);
  padding: 8px 12px;
  text-align: left;
}
.post-detail-content.markdown-content :deep(th) {
  background: var(--el-fill-color-light);
  font-weight: 600;
}
.post-detail-content.markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 6px;
}
.post-detail-content.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
  margin: 1.5em 0;
}
.post-detail-content.markdown-content :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}
.post-detail-content.markdown-content :deep(a:hover) {
  text-decoration: underline;
}
/* 发帖/编辑对话框：与 docs 一致的弹性布局，内容区自适应不溢出 */
.board-dialog-with-editor :deep(.el-dialog) {
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}
.board-dialog-with-editor :deep(.el-dialog__body) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.board-form-in-dialog {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.board-form-in-dialog :deep(.el-form-item) {
  margin-bottom: 16px;
}
.board-form-item-static {
  flex-shrink: 0;
}
.board-form-item-editor {
  flex: 1;
  min-height: 400px;
  display: flex;
  flex-direction: column;
  margin-bottom: 0;
}
.board-form-item-editor :deep(.el-form-item__content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.board-dialog-editor-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  width: 100%;
}
.board-dialog-editor-wrap .board-vditor-editor,
.board-dialog-editor-wrap :deep(.vditor-wrapper) {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.board-dialog-editor-wrap :deep(.vditor) {
  flex: 1;
  min-height: 0;
  height: 100%;
}

.cover-uploader :deep(.el-upload--picture-card) {
  width: 120px;
  height: 120px;
}
.cover-uploader-icon {
  font-size: 24px;
  color: var(--el-text-color-placeholder);
}
.cover-uploading-tip {
  font-size: 12px;
  color: var(--el-color-primary);
  margin-top: 4px;
}
.board-vditor-editor {
  width: 100%;
}
</style>
