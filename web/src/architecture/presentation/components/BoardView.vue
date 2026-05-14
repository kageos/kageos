<template>
  <div class="board-view" v-loading="loading">
    <div class="board-main">
    <div class="board-header">
      <h1 class="board-title">{{ node?.name || '讨论区' }}</h1>
      <el-button
        type="primary"
        :icon="Plus"
        @click="handleCreateButtonClick"
        size="large"
      >
        发帖
      </el-button>
    </div>

    <!-- 搜索（仅列表页显示） -->
    <div v-if="!selectedPostId" class="board-search">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索帖子标题…"
        clearable
        class="board-search-input"
        :prefix-icon="Search"
        size="large"
      />
    </div>

    <!-- 帖子列表：左侧大头像，右侧内容区（现代社区流式排版） -->
    <div class="post-list board-feed" v-if="!selectedPostId">
      <div
        v-for="row in filteredPosts"
        :key="row.id"
        class="board-card"
        @click="openPost(row.id)"
      >
        <!-- 左侧：固定头像区 -->
        <div class="board-card-left">
          <el-avatar :size="44" :src="authorAvatar(row.author)" class="board-card-avatar">
            {{ authorInitial(row.author) }}
          </el-avatar>
        </div>
        
        <!-- 右侧：主要内容区 -->
        <div class="board-card-right">
          <!-- 作者信息与时间 -->
          <div class="board-card-meta">
            <span class="board-card-author">{{ authorDisplayName(row.author) }}</span>
            <span class="board-card-time">{{ formatTime(row.created_at) }}</span>
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
          
          <!-- 底部操作区 -->
          <div class="board-card-actions" @click.stop>
            <div class="action-item" @click="openPost(row.id)">
              <el-button link type="primary" size="small">查看讨论</el-button>
            </div>
            <div class="action-item" @click="handleDeletePost(row)">
              <el-button link type="danger" size="small">删除</el-button>
            </div>
          </div>
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
        <el-button
          v-if="postDetail"
          link
          type="primary"
          @click="openEditForm"
        >
          编辑
        </el-button>
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
import { ref, computed, watch, defineAsyncComponent } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, ArrowLeft, Search } from '@element-plus/icons-vue'
import type { ServiceTree, UserInfo } from '@/architecture/domain/types'
import { listPosts, getPost, createPost, updatePost, deletePost, type PostItem, type GetPostResp } from '@/architecture/infrastructure/api/board'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import { useUserInfoStore } from '@/architecture/infrastructure/stores/userInfo'
import { sanitizeHtml } from '@/utils/sanitizeHtml'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'

const VditorEditor = defineAsyncComponent(() => import('@/shared/components/VditorEditor.vue'))
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const BOARD_COVER_UPLOAD_ROUTER = 'board/cover'

interface Props {
  node: ServiceTree
}

const props = defineProps<Props>()

const router = useRouter()
const userInfoStore = useUserInfoStore()
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

const handleCreateButtonClick = () => {
  showCreateForm.value = true
}

// 详情富文本渲染（markdown -> html）
const renderedContent = computed(() => {
  if (!postDetail.value?.content) return '（无正文）'
  if (postDetail.value.content_format === 'html') return sanitizeHtml(postDetail.value.content)
  return renderMarkdown(postDetail.value.content)
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
  padding: 24px; /* 增加内边距 */
  min-height: 200px;
  box-sizing: border-box;
}
.board-main {
  width: 100%;
  max-width: 1024px;
  margin: 0 auto;
}
.board-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px; /* 增加间距 */
  width: 100%;
}
.board-title {
  margin: 0;
  font-size: 24px; /* 标题变大，更加醒目 */
  font-weight: 600;
}
.board-search {
  margin-bottom: 20px; /* 增加搜索框与下方列表的间距 */
  width: 100%;
}
.board-search-input {
  max-width: 480px; /* 搜索框变宽更易用 */
}
.post-list-empty {
  padding: 40px 0;
  background: var(--app-shell-panel-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.03);
}
/* 讨论区列表：标题为主、用户信息弱化、摘要与标题分区、封面缩略 */
.post-list.board-feed {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 16px; /* 卡片之间的间距改用 gap */
  width: 100%;
  max-width: 1024px; /* 列表也限制最大宽度，与详情页一致 */
  margin-left: auto;
  margin-right: auto;
}
.board-card {
  display: flex;
  gap: 16px;
  padding: 24px; /* 增大内边距，让卡片更加舒展 */
  border: 1px solid transparent; /* 默认透明边框 */
  border-bottom: 1px solid var(--el-border-color-extra-light, #f3f4f6); /* 列表底部分割线 */
  border-radius: 16px; /* 圆角增加 */
  background: transparent; /* 默认无背景 */
  margin-bottom: 4px; 
  cursor: pointer;
  transition: all 0.2s ease;
}
.board-card:hover {
  background: var(--el-fill-color-light, #f9fafb); /* 悬浮时微妙的背景色 */
  border-color: var(--el-border-color-extra-light, #f3f4f6);
  transform: none; /* 移除之前的上浮，更现代 */
  box-shadow: none; /* 移除之前的阴影 */
}
.board-card:last-of-type {
  border-bottom-color: transparent;
}
/* 左侧大头像 */
.board-card-left {
  flex-shrink: 0;
}
.board-card-avatar {
  font-size: 18px;
  font-weight: 600;
  background: linear-gradient(135deg, #7dd3fc 0%, #38bdf8 100%); /* 渐变头像背景更好看 */
  color: #fff;
  box-shadow: 0 2px 4px rgba(56, 189, 248, 0.2);
}
/* 右侧内容区 */
.board-card-right {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
/* 头部信息：作者和时间 */
.board-card-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}
.board-card-author {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary, #111827);
}
.board-card-time {
  font-size: 13px;
  color: var(--el-text-color-placeholder, #9ca3af);
}
/* 标题：突出 */
.board-card-title {
  margin: 0 0 8px;
  font-size: 18px; /* 增大字号 */
  font-weight: 700;
  line-height: 1.4;
  color: var(--el-text-color-primary, #111827);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  letter-spacing: -0.01em;
}
/* 摘要：与标题留白，不连在一起 */
.board-card-summary {
  margin: 0 0 12px;
  font-size: 15px;
  color: var(--el-text-color-regular, #4b5563); 
  line-height: 1.6;
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
  margin-bottom: 24px;
  padding-bottom: 12px;
}
.post-detail-body {
  padding: 20px 0;
  max-width: 1024px; /* 调整为1024px，提供更宽广的阅读区 */
  width: 100%;
  box-sizing: border-box;
}
/* 详情内容区居中，阅读更舒适 */
.post-detail-body-centered {
  margin-left: auto;
  margin-right: auto;
  padding: 48px 56px; /* 增加内边距，呼应文档样式 */
  background: var(--app-shell-panel-bg, #fff); /* 添加卡片背景 */
  border-radius: 16px;
  border: 1px solid var(--el-border-color-extra-light, #f0f2f5);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.02), 0 20px 40px -10px rgba(0, 0, 0, 0.03); /* 高级弥散阴影 */
}
/* 标题 */
.post-detail-title {
  margin: 0 0 24px;
  font-size: 32px;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary, #111827);
}
/* 作者区（放置在标题下方，更现代） */
.post-detail-author {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--el-border-color-lighter, #f3f4f6);
}
.post-detail-author .detail-avatar {
  flex-shrink: 0;
  font-size: 18px;
  font-weight: 600;
  background: linear-gradient(135deg, #7dd3fc 0%, #38bdf8 100%);
  box-shadow: 0 2px 4px rgba(56, 189, 248, 0.2);
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
  gap: 12px;
}
.post-detail-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary, #111827);
}
.post-detail-time {
  font-size: 14px;
  color: var(--el-text-color-placeholder, #9ca3af);
}
.post-detail-username {
  font-size: 14px;
  color: var(--el-text-color-secondary, #6b7280);
}
.post-detail-signature {
  font-size: 14px;
  color: var(--el-text-color-placeholder, #9ca3af);
  line-height: 1.4;
  margin-top: 2px;
}
.post-detail-summary {
  margin: 0 0 32px;
  padding: 16px 20px;
  font-size: 15px;
  color: var(--el-text-color-regular, #374151);
  line-height: 1.6;
  background: var(--el-fill-color-light, #f9fafb);
  border-left: 4px solid var(--el-color-primary);
  border-radius: 4px 8px 8px 4px;
}
/* 详情封面九宫格 */
.post-detail-covers {
  display: grid;
  gap: 8px;
  margin-bottom: 32px;
  border-radius: 12px;
  overflow: hidden;
}
.post-detail-covers.moments-photos-1 {
  grid-template-columns: 1fr;
  max-width: 480px;
  max-height: 480px;
}
.post-detail-covers.moments-photos-2 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 480px;
}
.post-detail-covers.moments-photos-3 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 480px;
}
.post-detail-covers.moments-photos-4 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 480px;
}
.post-detail-covers.moments-photos-5,
.post-detail-covers.moments-photos-6,
.post-detail-covers.moments-photos-7,
.post-detail-covers.moments-photos-8,
.post-detail-covers.moments-photos-9 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 480px;
}
.post-detail-cover-img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  display: block;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  box-shadow: inset 0 0 0 1px rgba(0,0,0,0.05);
}
.post-detail-covers.moments-photos-1 .post-detail-cover-img {
  max-height: 480px;
  aspect-ratio: auto;
}
/* 正文：居中阅读 + Tailwind Typography 高级排版 */
.post-detail-content {
  font-size: 16px;
  line-height: 1.75;
  color: var(--el-text-color-regular, #374151);
  word-break: break-word;
}
.post-detail-content.markdown-content :deep(h1),
.post-detail-content.markdown-content :deep(h2),
.post-detail-content.markdown-content :deep(h3),
.post-detail-content.markdown-content :deep(h4),
.post-detail-content.markdown-content :deep(h5),
.post-detail-content.markdown-content :deep(h6) {
  color: var(--el-text-color-primary, #111827);
  font-weight: 600;
  line-height: 1.3;
  margin-top: 2em;
  margin-bottom: 1em;
}
.post-detail-content.markdown-content :deep(h1) {
  font-size: 2.25em;
  font-weight: 700;
  margin-top: 0;
  padding-bottom: 0.3em;
  border-bottom: 1px solid var(--el-border-color-lighter, #e5e7eb);
}
.post-detail-content.markdown-content :deep(h2) {
  font-size: 1.5em;
  padding-bottom: 0.3em;
  border-bottom: 1px solid var(--el-border-color-lighter, #e5e7eb);
}
.post-detail-content.markdown-content :deep(h3) { font-size: 1.25em; }
.post-detail-content.markdown-content :deep(h4) { font-size: 1em; }
.post-detail-content.markdown-content :deep(p) {
  margin-top: 1.25em;
  margin-bottom: 1.25em;
}
.post-detail-content.markdown-content :deep(code) {
  background: var(--el-fill-color-light, #f3f4f6);
  color: var(--el-color-primary, #0369a1);
  padding: 0.2em 0.4em;
  border-radius: 6px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.875em;
  font-weight: 500;
}
.post-detail-content.markdown-content :deep(pre) {
  background: #1f2937;
  color: #e5e7eb;
  padding: 1.25em 1.5em;
  border-radius: 8px;
  overflow-x: auto;
  margin-top: 1.7em;
  margin-bottom: 1.7em;
  font-size: 0.875em;
  line-height: 1.7142857;
  max-width: 100%;
  box-shadow: inset 0 0 0 1px rgba(255,255,255,0.1);
}
.post-detail-content.markdown-content :deep(pre code) {
  background: transparent;
  color: inherit;
  padding: 0;
  font-weight: 400;
  font-size: inherit;
  border-radius: 0;
}
.post-detail-content.markdown-content :deep(blockquote) {
  font-weight: 400;
  font-style: normal;
  color: var(--el-text-color-secondary, #4b5563);
  border-left: 4px solid var(--el-border-color, #d1d5db);
  padding-left: 1rem;
  margin-top: 1.6em;
  margin-bottom: 1.6em;
  background: transparent;
}
.post-detail-content.markdown-content :deep(ul),
.post-detail-content.markdown-content :deep(ol) {
  margin-top: 1.25em;
  margin-bottom: 1.25em;
  padding-left: 1.625em;
}
.post-detail-content.markdown-content :deep(li) {
  margin-top: 0.5em;
  margin-bottom: 0.5em;
}
.post-detail-content.markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-top: 2em;
  margin-bottom: 2em;
  font-size: 0.875em;
  display: block;
  overflow-x: auto;
  white-space: nowrap;
}
.post-detail-content.markdown-content :deep(th),
.post-detail-content.markdown-content :deep(td) {
  border: 1px solid var(--el-border-color-lighter, #e5e7eb);
  padding: 0.75em 1em;
  text-align: left;
}
.post-detail-content.markdown-content :deep(th) {
  background: var(--el-fill-color-light, #f9fafb);
  color: var(--el-text-color-primary, #111827);
  font-weight: 600;
}
.post-detail-content.markdown-content :deep(tr:nth-child(even)) {
  background: var(--el-fill-color-blank, #ffffff);
}
.post-detail-content.markdown-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
  margin-top: 2em;
  margin-bottom: 2em;
  display: block;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}
.post-detail-content.markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--el-border-color-lighter, #e5e7eb);
  margin-top: 3em;
  margin-bottom: 3em;
}
.post-detail-content.markdown-content :deep(a) {
  color: var(--el-color-primary);
  text-decoration: underline;
  text-decoration-color: transparent;
  font-weight: 500;
  transition: text-decoration-color 0.2s ease;
}
.post-detail-content.markdown-content :deep(a:hover) {
  text-decoration-color: currentColor;
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
