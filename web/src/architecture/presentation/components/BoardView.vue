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

    <!-- 帖子列表（朋友圈风格：头像+昵称+时间在上，九宫格图片） -->
    <div class="post-list moments-feed" v-if="!selectedPostId">
      <div
        v-for="row in filteredPosts"
        :key="row.id"
        class="moments-card"
        @click="openPost(row.id)"
      >
        <div class="moments-header">
          <el-avatar :size="44" :src="authorAvatar(row.author)" class="moments-avatar-el">
            {{ authorInitial(row.author) }}
          </el-avatar>
          <div class="moments-header-right">
            <div class="moments-name-row">
              <div class="moments-user-block">
                <span class="moments-name">{{ authorDisplayName(row.author) }}</span>
                <span v-if="getAuthorInfo(row.author)?.nickname" class="moments-username">@{{ row.author }}</span>
                <span v-if="getAuthorInfo(row.author)?.signature" class="moments-signature">{{ getAuthorInfo(row.author)!.signature }}</span>
              </div>
              <span class="moments-time">{{ formatTime(row.created_at) }}</span>
            </div>
            <div class="moments-text">{{ row.title }}</div>
          </div>
        </div>
        <div
          v-if="row.cover && (row.cover as string[]).length"
          :class="['moments-photos', 'moments-photos-' + Math.min((row.cover as string[]).length, 9)]"
        >
          <img
            v-for="(url, i) in (row.cover as string[]).slice(0, 9)"
            :key="i"
            :src="url"
            alt=""
            class="moments-photo"
          />
        </div>
        <div class="moments-actions" @click.stop>
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

    <!-- 帖子详情（与列表风格一致：作者区 + 标题 + 九宫格封面 + 正文） -->
    <div v-else class="post-detail">
      <div class="post-detail-header">
        <el-button link :icon="ArrowLeft" @click="selectedPostId = null">返回列表</el-button>
      </div>
      <div v-if="postDetail" class="post-detail-body">
        <!-- 作者区（与列表一致） -->
        <div class="post-detail-author">
          <el-avatar :size="44" :src="authorAvatar(postDetail.author)" class="detail-avatar">{{ authorInitial(postDetail.author) }}</el-avatar>
          <div class="post-detail-user-block">
            <div class="post-detail-name-row">
              <span class="post-detail-name">{{ authorDisplayName(postDetail.author) }}</span>
              <span class="post-detail-time">{{ formatTime(postDetail.created_at) }}</span>
            </div>
            <span v-if="getAuthorInfo(postDetail.author)?.nickname" class="post-detail-username">@{{ postDetail.author }}</span>
            <span v-if="getAuthorInfo(postDetail.author)?.signature" class="post-detail-signature">{{ getAuthorInfo(postDetail.author)!.signature }}</span>
          </div>
        </div>
        <!-- 标题 -->
        <h1 class="post-detail-title">{{ postDetail.title }}</h1>
        <!-- 封面九宫格（与列表一致） -->
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
        <!-- 正文（限制宽度便于阅读） -->
        <div class="post-detail-content markdown-content" v-html="renderedContent" />
      </div>
    </div>

    <!-- 发帖对话框（封面 + 富文本） -->
    <el-dialog v-model="showCreateForm" title="发帖" width="720" @close="resetCreateForm" class="board-create-dialog">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="封面（可多图）">
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
        <el-form-item label="标题" required>
          <el-input v-model="createForm.title" placeholder="请输入标题" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="内容">
          <VditorEditor
            v-model="createForm.content"
            height="320"
            placeholder="请输入正文（支持 Markdown）"
            class="board-vditor-editor"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateForm = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmitPost">发布</el-button>
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
import { listPosts, getPost, createPost, deletePost, type PostItem, type GetPostResp } from '@/api/board'
import VditorEditor from '@/components/VditorEditor.vue'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import { useAuthStore } from '@/stores/auth'
import { useUserInfoStore } from '@/stores/userInfo'
import { usePermissionErrorStore } from '@/stores/permissionError'
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
  if (postDetail.value.content_format === 'html') return postDetail.value.content
  try {
    return marked(postDetail.value.content)
  } catch {
    return postDetail.value.content
  }
})

// 发帖：封面 + 富文本
const showCreateForm = ref(false)
const submitting = ref(false)
const coverUploading = ref(false)
const createForm = ref<{ title: string; cover: string[]; content: string; content_format: string }>({
  title: '',
  cover: [],
  content: '',
  content_format: 'markdown'
})

const coverFileList = computed(() =>
  createForm.value.cover.map((url, i) => ({ name: `封面${i + 1}`, url }))
)

const resetCreateForm = () => {
  createForm.value = { title: '', cover: [], content: '', content_format: 'markdown' }
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
/* 朋友圈风格列表（宽度拉满） */
.post-list.moments-feed {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 0;
  width: 100%;
}
.moments-card {
  padding: 12px 16px 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
  transition: background 0.2s;
}
.moments-card:hover {
  background: var(--el-fill-color-light);
}
.moments-card:last-of-type {
  border-bottom: none;
}
.moments-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 8px;
}
.moments-avatar-el {
  flex-shrink: 0;
  font-size: 17px;
  font-weight: 600;
  background: linear-gradient(135deg, #7dd3fc 0%, #38bdf8 100%);
}
.moments-header-right {
  flex: 1;
  min-width: 0;
}
.moments-name-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}
.moments-user-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.moments-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.moments-username {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.moments-signature {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.moments-time {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.moments-text {
  font-size: 15px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
/* 朋友圈九宫格：1张大、2-3横排、4为2x2、5-9为3列网格 */
.moments-photos {
  margin: 8px 0 8px 56px; /* 与头像+文字左对齐（44+12） */
  display: grid;
  gap: 4px;
  border-radius: 8px;
  overflow: hidden;
}
.moments-photos-1 {
  grid-template-columns: 1fr;
  max-width: 240px;
  max-height: 240px;
}
.moments-photos-2 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 240px;
}
.moments-photos-3 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 240px;
}
.moments-photos-4 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 240px;
}
.moments-photos-5,
.moments-photos-6,
.moments-photos-7,
.moments-photos-8,
.moments-photos-9 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 240px;
}
.moments-photo {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  display: block;
  background: var(--el-fill-color-lighter);
}
.moments-photos-1 .moments-photo {
  max-height: 240px;
  aspect-ratio: auto;
}
.moments-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-left: 56px;
  padding-top: 6px;
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
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.post-detail-body {
  padding: 20px 0;
  max-width: 720px;
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
/* 正文（限制宽度 + 舒适行高） */
.post-detail-content {
  line-height: 1.75;
  color: var(--el-text-color-regular);
}
.post-detail-content.markdown-content :deep(h1),
.post-detail-content.markdown-content :deep(h2),
.post-detail-content.markdown-content :deep(h3) {
  margin-top: 1.25em;
  margin-bottom: 0.5em;
  color: var(--el-text-color-primary);
}
.post-detail-content.markdown-content :deep(p) {
  margin-bottom: 1em;
}
.post-detail-content.markdown-content :deep(pre),
.post-detail-content.markdown-content :deep(code) {
  background: var(--el-fill-color-light);
  border-radius: 6px;
}
.post-detail-content.markdown-content :deep(pre) {
  padding: 14px;
  overflow-x: auto;
}
.post-detail-content.markdown-content :deep(blockquote) {
  margin: 1em 0;
  padding-left: 1em;
  border-left: 4px solid var(--el-border-color);
  color: var(--el-text-color-secondary);
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
