<!--
  HubDirectoryDetail - Hub 目录详情页面
  
  参考 OS 的 PackageDetailView 样式和布局
-->
<template>
  <div class="hub-directory-detail-view">
    <!-- 顶部横幅区域 -->
    <div class="hero-section">
      <div class="hero-content">
        <el-button
          @click="handleBack"
          :icon="ArrowLeft"
          circle
          class="back-button"
          size="large"
        />
        <div class="hero-info">
          <div class="hero-icon-wrapper">
            <img
              src="/service-tree/custom-folder.svg"
              alt="目录"
              class="hero-icon-img"
            />
          </div>
          <div class="hero-text">
            <h1 class="hero-title">{{ directoryDetail?.name || 'Hub 目录' }}</h1>
            <p class="hero-subtitle" v-if="directoryDetail?.full_code_path">
              <el-icon class="path-icon"><Link /></el-icon>
              <span class="path-text">{{ directoryDetail.full_code_path }}</span>
              <el-button
                text
                :icon="CopyDocument"
                @click="handleCopyPath"
                class="path-copy-btn"
                size="small"
                title="复制路径"
              />
              <span class="action-divider">|</span>
              <el-button
                text
                type="primary"
                @click="handleTryDirectory"
                class="action-link-inline"
                size="small"
              >
                <el-icon><Operation /></el-icon>
                试用
              </el-button>
              <el-button
                text
                type="success"
                @click="handleInstallDirectory"
                class="action-link-inline"
                size="small"
                title="复制 Hub 链接，可在工作空间中粘贴使用"
              >
                <el-icon><CopyDocument /></el-icon>
                复制使用
              </el-button>
            </p>
            <p class="hero-description" v-if="directoryDetail?.description">
              <div
                class="description-html"
                v-html="directoryDetail.description"
              />
            </p>
            <div class="hero-badges" v-if="directoryDetail">
              <el-tag v-if="directoryDetail.category" type="info" size="large">
                {{ directoryDetail.category }}
              </el-tag>
              <el-tag
                v-if="directoryDetail.service_fee_personal > 0"
                type="warning"
                size="large"
              >
                ¥{{ directoryDetail.service_fee_personal }} / 个人
              </el-tag>
              <el-tag v-else type="success" size="large">免费</el-tag>
              <el-tag type="info" size="large">
                版本 {{ directoryDetail.version }}
              </el-tag>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域：左右分栏 -->
    <div class="main-content">
      <!-- 左侧：目录树 -->
      <div class="tree-sidebar" v-if="directoryDetail?.directory_tree">
        <div class="sidebar-header">
          <h3 class="sidebar-title">
            <el-icon class="sidebar-icon"><Files /></el-icon>
            目录结构
          </h3>
        </div>
        <div class="tree-container">
          <el-tree
            :data="treeData"
            :props="{ children: 'children', label: 'name' }"
            node-key="path"
            :default-expand-all="true"
            :expand-on-click-node="false"
            :highlight-current="true"
            @node-click="handleTreeNodeClick"
          >
            <template #default="{ node, data }">
              <span class="tree-node">
                <!-- package 类型：显示自定义文件夹图标 -->
                <img 
                  v-if="data.type === 'package'" 
                  src="/service-tree/custom-folder.svg" 
                  alt="目录" 
                  class="node-icon package-icon-img"
                />
                <!-- function 类型：根据 template_type 显示不同图标 -->
                <template v-else-if="data.type === 'function'">
                  <!-- 表单类型：使用自定义 SVG -->
                  <img 
                    v-if="data.template_type === 'form'"
                    src="/service-tree/表单 (3).svg" 
                    alt="表单" 
                    class="node-icon form-icon-img"
                  />
                  <!-- 其他类型：使用组件图标 -->
                  <el-icon v-else class="node-icon">
                    <TableIcon v-if="data.template_type === 'table'" />
                    <ChartIcon v-else-if="data.template_type === 'chart'" />
                    <Operation v-else />
                  </el-icon>
                </template>
                <!-- 文件类型 -->
                <el-icon v-else-if="data.type === 'file'" class="node-icon">
                  <Document />
                </el-icon>
                <span class="node-label">{{ node.label }}</span>
              </span>
            </template>
          </el-tree>
        </div>
      </div>

      <!-- 右侧：详情内容 -->
      <div class="detail-content">
        <!-- 信息概览卡片 -->
        <div v-if="directoryDetail" class="overview-section">
          <div class="overview-card">
            <div class="overview-item">
              <div class="overview-icon-wrapper name-icon">
                <el-icon class="overview-icon"><Document /></el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-label">目录名称</div>
                <div class="overview-value">{{ directoryDetail.name }}</div>
              </div>
            </div>

            <div class="overview-divider"></div>

            <div class="overview-item">
              <div class="overview-icon-wrapper code-icon">
                <el-icon class="overview-icon"><Key /></el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-label">完整路径</div>
                <div class="overview-value code-text">{{ directoryDetail.full_code_path }}</div>
              </div>
            </div>

            <div class="overview-divider"></div>

            <div class="overview-item">
              <div class="overview-icon-wrapper count-icon">
                <el-icon class="overview-icon"><Files /></el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-label">子项数量</div>
                <div class="overview-value">
                  {{ getTotalChildrenCount() }} 项
                </div>
              </div>
            </div>

            <div class="overview-divider"></div>

            <div class="overview-item">
              <div class="overview-icon-wrapper stats-icon">
                <el-icon class="overview-icon"><DataAnalysis /></el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-label">统计信息</div>
                <div class="overview-value stats-text">
                  目录: {{ directoryDetail.directory_count }} | 
                  文件: {{ directoryDetail.file_count }} | 
                  函数: {{ directoryDetail.function_count }}
                </div>
              </div>
            </div>

            <div class="overview-divider" v-if="directoryDetail?.publisher_username"></div>

            <div class="overview-item" v-if="directoryDetail?.publisher_username">
              <div class="overview-icon-wrapper user-icon">
                <el-icon class="overview-icon"><User /></el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-label">发布者</div>
                <div class="overview-value">
                  <UserDisplay 
                    :username="directoryDetail.publisher_username" 
                    layout="horizontal" 
                    size="small"
                  />
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 标签区域 -->
        <div class="tags-section" v-if="directoryDetail?.tags && directoryDetail.tags.length > 0">
          <h3 class="section-title">
            <el-icon class="section-icon"><CollectionTag /></el-icon>
            标签
          </h3>
          <div class="tags-list">
            <el-tag
              v-for="tag in directoryDetail.tags"
              :key="tag"
              type="info"
              size="default"
              class="tag-item"
            >
              {{ tag }}
            </el-tag>
          </div>
        </div>

        <!-- 目录树结构（嵌套展示） -->
        <div class="children-section" v-if="directoryDetail?.directory_tree">
          <div class="section-header">
            <h3 class="section-title">
              <el-icon class="section-icon"><Files /></el-icon>
              目录结构
            </h3>
            <el-tag class="section-badge" type="primary" size="small">
              {{ getTotalChildrenCount() }}
            </el-tag>
          </div>

          <!-- 递归渲染目录树 -->
          <div class="children-grid">
            <DirectoryNodeWrapper
              v-for="subdir in directoryDetail.directory_tree.subdirectories"
              :key="subdir.path"
              :node="subdir"
            />
            <!-- 根目录下的函数和文件 -->
            <div
              v-for="func in directoryDetail.directory_tree.functions"
              :key="func.full_code_path"
              class="child-card"
              @click.stop="handleFunctionClick(func)"
            >
              <div class="child-card-header">
                <div class="child-icon-wrapper function-type">
                  <img
                    v-if="func.template_type === 'form'"
                    src="/service-tree/表单 (3).svg"
                    alt="表单"
                    class="child-icon-img"
                  />
                  <TableIcon v-else-if="func.template_type === 'table'" class="child-icon" />
                  <ChartIcon v-else-if="func.template_type === 'chart'" class="child-icon" />
                  <el-icon v-else class="child-icon"><Operation /></el-icon>
                </div>
                <el-tag
                  size="small"
                  :type="getTemplateTypeTag(func.template_type)"
                  class="child-type-tag"
                >
                  {{ getTemplateTypeText(func.template_type) }}
                </el-tag>
              </div>
              <div class="child-card-body">
                <div class="child-name">{{ func.name }}</div>
                <div class="child-description" v-if="func.description">
                  {{ func.description }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <el-empty
          v-else-if="!loading"
          description="该目录下暂无子目录或函数"
          :image-size="120"
          class="empty-state"
        />
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="detailDialogTitle"
      width="700px"
      :close-on-click-modal="false"
      class="detail-dialog"
    >
      <div v-if="selectedItem" class="detail-dialog-content">
        <!-- 目录详情 -->
        <template v-if="selectedItem.type === 'package'">
          <div class="detail-header">
            <div class="detail-icon-wrapper package-icon-wrapper">
              <img src="/service-tree/custom-folder.svg" alt="目录" class="detail-icon-img" />
            </div>
            <div class="detail-header-info">
              <h3 class="detail-title">{{ selectedItem.name }}</h3>
              <p class="detail-path">{{ selectedItem.path }}</p>
            </div>
          </div>
          <div class="detail-section" v-if="selectedItem.description">
            <h4 class="section-title">描述</h4>
            <p class="section-content">{{ selectedItem.description }}</p>
          </div>
        </template>

        <!-- 函数详情 -->
        <template v-else-if="selectedItem.type === 'function'">
          <div class="detail-header">
            <div class="detail-icon-wrapper function-icon-wrapper">
              <img
                v-if="selectedItem.template_type === 'form'"
                src="/service-tree/表单 (3).svg"
                alt="表单"
                class="detail-icon-img"
              />
              <TableIcon v-else-if="selectedItem.template_type === 'table'" class="detail-icon" />
              <ChartIcon v-else-if="selectedItem.template_type === 'chart'" class="detail-icon" />
              <el-icon v-else class="detail-icon"><Operation /></el-icon>
            </div>
            <div class="detail-header-info">
              <div class="detail-title-row">
                <h3 class="detail-title">{{ selectedItem.name }}</h3>
                <el-tag :type="getTemplateTypeTag(selectedItem.template_type)" size="small" class="type-tag">
                  {{ getTemplateTypeText(selectedItem.template_type) }}
                </el-tag>
              </div>
              <p class="detail-path">{{ selectedItem.full_code_path || selectedItem.path }}</p>
            </div>
          </div>
          <div class="detail-section" v-if="selectedItem.description">
            <h4 class="section-title">描述</h4>
            <p class="section-content">{{ selectedItem.description }}</p>
          </div>
          <div class="detail-section" v-if="selectedItem.tags && selectedItem.tags.length > 0">
            <h4 class="section-title">标签</h4>
            <div class="tags-list">
              <el-tag
                v-for="tag in selectedItem.tags"
                :key="tag"
                type="info"
                size="default"
                class="tag-item"
              >
                {{ tag }}
              </el-tag>
            </div>
          </div>
        </template>

        <!-- 文件详情 -->
        <template v-else-if="selectedItem.type === 'file'">
          <div class="detail-header">
            <div class="detail-icon-wrapper file-icon-wrapper">
              <el-icon class="detail-icon"><Document /></el-icon>
            </div>
            <div class="detail-header-info">
              <h3 class="detail-title">{{ selectedItem.name || selectedItem.relative_path }}</h3>
              <p class="detail-path">{{ selectedItem.relative_path }}</p>
            </div>
          </div>
          <div class="detail-section" v-if="selectedItem.file_type">
            <h4 class="section-title">文件类型</h4>
            <p class="section-content">{{ selectedItem.file_type }}</p>
          </div>
        </template>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <el-button @click="detailDialogVisible = false">关闭</el-button>
          <el-button
            v-if="selectedItem && (selectedItem.type === 'package' || selectedItem.type === 'function')"
            type="primary"
            @click="handleTryIt"
            size="large"
          >
            <el-icon><Operation /></el-icon>
            试用
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, defineComponent, h, computed } from 'vue'
import type { PropType } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  Document,
  CopyDocument,
  Key,
  Link,
  Files,
  DataAnalysis,
  CollectionTag,
  Operation,
  User
} from '@element-plus/icons-vue'
import { ElMessage, ElTag } from 'element-plus'
import { getHubDirectoryDetail, type HubDirectoryDetail, type DirectoryTreeNode } from '@/api/hub'
import ChartIcon from '@/components/icons/ChartIcon.vue'
import TableIcon from '@/components/icons/TableIcon.vue'
import UserDisplay from '@/components/UserDisplay.vue'
import { useUserInfoStore } from '@/stores/userInfo'

const route = useRoute()
const router = useRouter()
const userInfoStore = useUserInfoStore()

const loading = ref(false)
const directoryDetail = ref<HubDirectoryDetail | null>(null)

// 详情对话框相关
const detailDialogVisible = ref(false)
const selectedItem = ref<any>(null)
const detailDialogTitle = computed(() => {
  if (!selectedItem.value) return '详情'
  if (selectedItem.value.type === 'package') return '目录详情'
  if (selectedItem.value.type === 'function') return '函数详情'
  if (selectedItem.value.type === 'file') return '文件详情'
  return '详情'
})

// 加载目录详情
const loadDirectoryDetail = async () => {
  const directoryId = Number(route.params.id)
  if (!directoryId) {
    ElMessage.error('目录ID无效')
    return
  }

  loading.value = true
  try {
    const detail = await getHubDirectoryDetail(directoryId, true)
    directoryDetail.value = detail
    
    // 🔥 预加载发布者的用户信息（使用缓存）
    if (detail.publisher_username) {
      userInfoStore.getUserInfo(detail.publisher_username).catch((error: any) => {
        console.warn('[HubDirectoryDetail] 预加载用户信息失败:', error)
      })
    }
  } catch (error: any) {
    ElMessage.error(`加载目录详情失败: ${error.message || '未知错误'}`)
    console.error('加载目录详情失败:', error)
  } finally {
    loading.value = false
  }
}

// 返回
const handleBack = () => {
  router.push({ name: 'hub-market' })
}

// 复制完整路径
async function handleCopyPath() {
  if (!directoryDetail.value?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }

  try {
    await navigator.clipboard.writeText(directoryDetail.value.full_code_path)
    ElMessage.success('路径已复制到剪贴板')
  } catch (error) {
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = directoryDetail.value.full_code_path
    textArea.style.position = 'fixed'
    textArea.style.opacity = '0'
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('路径已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  }
}

// 获取模板类型标签类型
function getTemplateTypeTag(templateType: string): 'success' | 'primary' | 'warning' | 'info' {
  const typeMap: Record<string, 'success' | 'primary' | 'warning' | 'info'> = {
    'table': 'success',
    'form': 'primary',
    'chart': 'warning'
  }
  return typeMap[templateType] || 'info'
}

// 获取模板类型文本
function getTemplateTypeText(templateType: string): string {
  const typeMap: Record<string, string> = {
    'table': '表格',
    'form': '表单',
    'chart': '图表'
  }
  return typeMap[templateType] || '函数'
}

// 递归计算总子项数量
function countChildren(node: DirectoryTreeNode): number {
  let count = 0
  if (node.functions) count += node.functions.length
  // ⭐ 不再计算 files（已移除）
  if (node.subdirectories) {
    for (const subdir of node.subdirectories) {
      count += countChildren(subdir)
    }
  }
  return count
}

// 获取总子项数量
function getTotalChildrenCount(): number {
  if (!directoryDetail.value?.directory_tree) {
    return 0
  }
  return countChildren(directoryDetail.value.directory_tree)
}

// 将 DirectoryTreeNode 转换为 el-tree 需要的格式
function convertToTreeData(node: DirectoryTreeNode): any {
  const children: any[] = []
  
  // 添加子目录
  if (node.subdirectories && node.subdirectories.length > 0) {
    children.push(...node.subdirectories.map(subdir => convertToTreeData(subdir)))
  }
  
  // 添加函数
  if (node.functions && node.functions.length > 0) {
    children.push(...node.functions.map(func => ({
      name: func.name,
      path: func.full_code_path,
      type: 'function',
      template_type: func.template_type,
      description: func.description
    })))
  }
  
  // ⭐ 不再添加文件（已移除）
  
  return {
    name: node.name,
    path: node.path,
    type: node.type || 'package',
    children: children.length > 0 ? children : undefined
  }
}

// 计算树形数据
const treeData = computed(() => {
  if (!directoryDetail.value?.directory_tree) {
    return []
  }
  return [convertToTreeData(directoryDetail.value.directory_tree)]
})

// 目录节点包装组件（递归）
const DirectoryNodeWrapper = defineComponent({
  name: 'DirectoryNodeWrapper',
  props: {
    node: {
      type: Object as PropType<DirectoryTreeNode>,
      required: true
    }
  },
  setup(props, { emit }): () => any {
    return () => h('div', {
      class: 'directory-wrapper-card',
      onClick: (e: Event) => {
        // 如果点击的是子元素（函数或文件卡片），不触发目录点击
        const target = e.target as HTMLElement
        if (target.closest('.child-card')) {
          return
        }
        handleDirectoryClick(props.node)
      }
    }, [
      h('div', { class: 'directory-header' }, [
        h('div', { class: 'directory-header-left' }, [
          h('img', {
            src: '/service-tree/custom-folder.svg',
            alt: '目录',
            class: 'directory-icon'
          }),
          h('div', { class: 'directory-info' }, [
            h('div', { class: 'directory-name' }, props.node.name),
            h('div', { class: 'directory-path' }, props.node.path)
          ])
        ]),
        h(ElTag, {
          type: 'primary',
          size: 'small'
        }, () => `目录 · ${countChildren(props.node)} 项`)
      ]),
      h('div', { class: 'directory-content children-grid' }, [
        // 子目录
        ...(props.node.subdirectories || []).map((subdir: DirectoryTreeNode) =>
          h(DirectoryNodeWrapper, {
            key: subdir.path,
            node: subdir
          })
        ),
        // 函数
        ...(props.node.functions || []).map((func: any) =>
          h('div', {
            key: func.full_code_path,
            class: 'child-card',
            onClick: (e: Event) => {
              e.stopPropagation() // 阻止事件冒泡
              handleFunctionClick(func)
            }
          }, [
            h('div', { class: 'child-card-header' }, [
              h('div', { class: 'child-icon-wrapper function-type' }, [
                func.template_type === 'form'
                  ? h('img', { src: '/service-tree/表单 (3).svg', alt: '表单', class: 'child-icon-img' })
                  : func.template_type === 'table'
                  ? h(TableIcon, { class: 'child-icon' })
                  : func.template_type === 'chart'
                  ? h(ChartIcon, { class: 'child-icon' })
                  : h(Operation, { class: 'child-icon' })
              ]),
              h(ElTag, {
                size: 'small',
                type: getTemplateTypeTag(func.template_type),
                class: 'child-type-tag'
              }, () => getTemplateTypeText(func.template_type))
            ]),
            h('div', { class: 'child-card-body' }, [
              h('div', { class: 'child-name' }, func.name),
              func.description && h('div', { class: 'child-description' }, func.description)
            ])
          ])
        ),
        // ⭐ 不再渲染文件（已移除）
      ])
    ])
  }
})

// 获取 OS 的 base URL（用于跳转到 OS 的工作空间）
function getOSBaseURL(): string {
  // 从环境变量获取配置
  const osBaseURL = import.meta.env.VITE_OS_BASE_URL
  
  if (osBaseURL) {
    return osBaseURL
  }
  
  // 如果没有配置，从当前域名推断（Hub 和 OS 通常在同一域名下）
  const currentHost = window.location.host
  const currentProtocol = window.location.protocol
  
  // 如果是开发环境，默认使用 localhost:5173
  if (import.meta.env.DEV) {
    return `${currentProtocol}//${currentHost.replace(/:\d+$/, ':5173')}`
  }
  
  // 生产环境，使用当前域名
  return `${currentProtocol}//${currentHost}`
}

// 处理树节点点击 - 显示详情对话框
const handleTreeNodeClick = (data: any) => {
  if (!data) {
    return
  }
  
  // 设置选中的项目并显示对话框
  selectedItem.value = {
    ...data,
    type: data.type || 'package'
  }
  detailDialogVisible.value = true
}

// 处理目录卡片点击 - 显示详情对话框
const handleDirectoryClick = (node: DirectoryTreeNode) => {
  if (!node) {
    return
  }
  
  selectedItem.value = {
    ...node,
    type: 'package'
  }
  detailDialogVisible.value = true
}

// 处理函数卡片点击 - 显示详情对话框
const handleFunctionClick = (func: any) => {
  if (!func) {
    console.warn('Function data is empty')
    return
  }
  
  selectedItem.value = {
    ...func,
    type: 'function'
  }
  detailDialogVisible.value = true
}

// 试用目录 - 跳转到 OS 工作空间
const handleTryDirectory = () => {
  if (!directoryDetail.value?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }
  
  const osBaseURL = getOSBaseURL()
  const workspacePath = `/workspace${directoryDetail.value.full_code_path}`
  const targetURL = `${osBaseURL}${workspacePath}`
  
  // 在新窗口打开
  window.open(targetURL, '_blank')
}

// 复制 Hub 链接 - 生成 Hub 链接并复制到剪贴板
const handleInstallDirectory = async () => {
  if (!directoryDetail.value) {
    ElMessage.warning('目录信息不可用')
    return
  }
  
  // 获取当前 Hub 的 host
  const hubHost = getHubHost()
  const fullCodePath = directoryDetail.value.full_code_path
  const version = directoryDetail.value.version || ''
  
  // 生成 Hub 链接：hub://{host}/{full_code_path}@version
  let hubLink = `hub://${hubHost}${fullCodePath}`
  if (version) {
    hubLink += `@${version}`
  }
  
  try {
    // 复制到剪贴板
    await navigator.clipboard.writeText(hubLink)
    ElMessage.success({
      message: `Hub 链接已复制到剪贴板：\n${hubLink}\n\n请在 OS 工作空间中使用"从应用中心安装"功能粘贴此链接`,
      duration: 5000,
      showClose: true
    })
  } catch (error) {
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = hubLink
    textArea.style.position = 'fixed'
    textArea.style.opacity = '0'
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success({
        message: `Hub 安装链接已复制到剪贴板！\n\n链接：${hubLink}\n\n下一步：\n1. 打开 OS 工作空间\n2. 点击"从应用中心安装"\n3. 粘贴此链接即可安装`,
        duration: 6000,
        showClose: true
      })
    } catch (err) {
      ElMessage.error('复制失败，请手动复制：' + hubLink)
    }
    document.body.removeChild(textArea)
  }
}

// 获取 Hub 的 host
function getHubHost(): string {
  // 从环境变量获取配置
  const hubHost = import.meta.env.VITE_HUB_HOST
  
  if (hubHost) {
    return hubHost
  }
  
  // 如果没有配置，从当前域名推断
  const isDev = import.meta.env.DEV || import.meta.env.MODE === 'development'
  if (isDev) {
    // 开发环境：使用 localhost
    return 'localhost:5174'
  } else {
    // 生产环境：使用当前域名
    return window.location.host
  }
}

// 处理试用按钮点击 - 跳转到 OS
const handleTryIt = () => {
  if (!selectedItem.value) {
    return
  }
  
  let fullCodePath = ''
  
  if (selectedItem.value.type === 'package') {
    fullCodePath = selectedItem.value.path
  } else if (selectedItem.value.type === 'function') {
    fullCodePath = selectedItem.value.full_code_path || selectedItem.value.path
  }
  
  if (!fullCodePath) {
    ElMessage.warning('路径信息不可用')
    return
  }
  
  const osBaseURL = getOSBaseURL()
  const workspacePath = `/workspace${fullCodePath}`
  const targetURL = `${osBaseURL}${workspacePath}`
  
  // 在新窗口打开
  window.open(targetURL, '_blank')
  
  // 关闭对话框
  detailDialogVisible.value = false
}

// 处理文件卡片点击 - 显示详情对话框
const handleFileClick = (file: any) => {
  if (!file) {
    return
  }
  
  selectedItem.value = {
    ...file,
    type: 'file'
  }
  detailDialogVisible.value = true
}

onMounted(() => {
  loadDirectoryDetail()
})
</script>

<style scoped lang="scss">
.hub-directory-detail-view {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
  overflow: hidden;

  // 顶部横幅区域
  .hero-section {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 32px 40px;

    .hero-content {
      max-width: 1400px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      gap: 24px;

      .back-button {
        flex-shrink: 0;
        background: var(--el-bg-color);
        border-color: var(--el-border-color);
        color: var(--el-text-color-regular);

        &:hover {
          background: var(--el-color-primary-light-9);
          border-color: var(--el-color-primary);
          color: var(--el-color-primary);
        }
      }

      .hero-info {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 20px;
        min-width: 0;

        .hero-icon-wrapper {
          flex-shrink: 0;
          display: flex;
          align-items: flex-start;
          justify-content: center;
          padding-top: 4px;

          .hero-icon-img {
            width: 48px;
            height: 48px;
            object-fit: contain;
          }
        }

        .hero-text {
          flex: 1;
          min-width: 0;

          .hero-title {
            margin: 0 0 8px 0;
            font-size: 28px;
            font-weight: 700;
            color: var(--el-text-color-primary);
            line-height: 1.2;
          }

          .hero-subtitle {
            margin: 0 0 8px 0;
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            color: var(--el-text-color-secondary);

            .path-icon {
              font-size: 16px;
              color: var(--el-color-primary);
            }

            .path-text {
              flex: 1;
              font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
              color: var(--el-text-color-regular);
              word-break: break-all;
            }

            .path-copy-btn {
              flex-shrink: 0;
              color: var(--el-text-color-secondary);

              &:hover {
                color: var(--el-color-primary);
              }
            }
            
            .action-divider {
              margin: 0 8px;
              color: var(--el-border-color);
              flex-shrink: 0;
            }
            
            .action-link-inline {
              flex-shrink: 0;
              padding: 2px 6px;
              font-size: 13px;
              color: var(--el-text-color-regular);
              transition: all 0.2s;
              
              .el-icon {
                margin-right: 3px;
                font-size: 13px;
              }
              
              &:hover {
                color: var(--el-color-primary);
                background-color: var(--el-color-primary-light-9);
              }
              
              &.el-button--primary {
                color: var(--el-color-primary);
                
                &:hover {
                  background-color: var(--el-color-primary-light-9);
                }
              }
              
              &.el-button--success {
                color: var(--el-color-success);
                
                &:hover {
                  background-color: var(--el-color-success-light-9);
                }
              }
            }
          }

          .hero-description {
            margin: 8px 0;
            font-size: 15px;
            color: var(--el-text-color-regular);
            line-height: 1.6;
            padding: 12px 16px;
            background: var(--el-fill-color-lighter);
            border-radius: 8px;
            border-left: 3px solid var(--el-color-primary);

            .description-html {
              :deep(p) {
                margin: 0 0 8px 0;
                &:last-child {
                  margin-bottom: 0;
                }
              }
            }
          }

          .hero-badges {
            display: flex;
            gap: 8px;
            flex-wrap: wrap;
            margin-top: 12px;
          }
          
        }
      }
    }
  }

  // 主要内容区域：左右分栏
  .main-content {
    flex: 1;
    min-height: 0; // 确保 flex 子元素可以收缩
    display: flex;
    overflow: hidden;

    // 左侧：目录树
    .tree-sidebar {
      width: 320px;
      flex-shrink: 0;
      background: var(--el-bg-color);
      border-right: 1px solid var(--el-border-color-lighter);
      display: flex;
      flex-direction: column;

      .sidebar-header {
        padding: 20px;
        border-bottom: 1px solid var(--el-border-color-lighter);

        .sidebar-title {
          margin: 0;
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);

          .sidebar-icon {
            font-size: 18px;
            color: var(--el-color-primary);
          }
        }
      }

      .tree-container {
        flex: 1;
        overflow-y: auto;
        padding: 8px;
        padding-bottom: 100px;

        .tree-node {
          display: flex;
          align-items: center;
          gap: 8px;
          flex: 1;
          width: 100%;
          
          .node-icon {
            width: 16px;
            height: 16px;
            margin-right: 8px;
            color: #6366f1;
            opacity: 0.8;
            flex-shrink: 0;
            transition: color 0.2s ease;
            
            &.package-icon-img {
              width: 16px;
              height: 16px;
              object-fit: contain;
              opacity: 0.9;
            }
            
            &.form-icon-img {
              width: 16px;
              height: 16px;
              object-fit: contain;
              opacity: 0.9;
            }
          }
          
          .node-label {
            font-size: 14px;
            color: var(--el-text-color-primary);
            flex: 1;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }

        :deep(.el-tree-node__content) {
          height: 32px;
          padding: 0 8px;
          display: flex;
          align-items: center;
          
          &:hover {
            background-color: var(--el-fill-color-light);
          }
        }

        :deep(.el-tree-node.is-current > .el-tree-node__content) {
          background-color: var(--el-fill-color-lighter);
          border-left: 2px solid #6366f1;
          
          .tree-node {
            .node-label {
              color: var(--el-text-color-primary);
              font-weight: 500;
            }
            
            .node-icon {
              color: #6366f1;
              opacity: 0.8;
            }
          }
        }

        :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
          background-color: transparent;
          border-left: none;
        }
      }
    }

    .detail-content {
      flex: 1;
      min-height: 0; // 确保可以收缩
      overflow-y: auto;
      overflow-x: hidden;
      display: flex;
      flex-direction: column;
      padding: 32px 40px;
      min-width: 0;
      width: 100%;
      max-width: 1400px;
      margin: 0 auto;

      // 信息概览卡片
      .overview-section {
        margin-bottom: 32px;

        .overview-card {
          display: flex;
          align-items: center;
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 16px;
          padding: 24px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

          .overview-item {
            flex: 1;
            display: flex;
            align-items: center;
            gap: 16px;

            .overview-icon-wrapper {
              flex-shrink: 0;
              display: flex;
              align-items: center;
              justify-content: center;
              width: 48px;
              height: 48px;
              border-radius: 12px;

              &.name-icon {
                background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-primary);
                }
              }

              &.code-icon {
                background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-success);
                }
              }

              &.count-icon {
                background: linear-gradient(135deg, var(--el-color-warning-light-8), var(--el-color-warning-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-warning);
                }
              }

              &.stats-icon {
                background: linear-gradient(135deg, var(--el-color-info-light-8), var(--el-color-info-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-info);
                }
              }

              &.user-icon {
                background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-success);
                }
              }
            }
            
            // UserDisplay 组件样式调整
            .overview-content {
              :deep(.user-display-wrapper) {
                .user-name {
                  font-size: 14px;
                  color: var(--el-text-color-primary);
                }
              }
            }

            .overview-content {
              flex: 1;
              min-width: 0;

              .overview-label {
                font-size: 13px;
                color: var(--el-text-color-secondary);
                margin-bottom: 4px;
                font-weight: 500;
              }

              .overview-value {
                font-size: 18px;
                font-weight: 600;
                color: var(--el-text-color-primary);

                &.code-text {
                  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
                  color: var(--el-color-success);
                  font-size: 16px;
                }

                &.stats-text {
                  font-size: 14px;
                  color: var(--el-text-color-regular);
                }
              }
            }
          }

          .overview-divider {
            width: 1px;
            height: 48px;
            background: var(--el-border-color-lighter);
            margin: 0 24px;
          }
        }
      }

      // 标签区域
      .tags-section {
        margin-bottom: 32px;

        .section-title {
          margin: 0 0 16px 0;
          display: flex;
          align-items: center;
          gap: 10px;
          font-size: 20px;
          font-weight: 600;
          color: var(--el-text-color-primary);

          .section-icon {
            font-size: 22px;
            color: var(--el-color-primary);
          }
        }

        .tags-list {
          display: flex;
          flex-wrap: wrap;
          gap: 8px;

          .tag-item {
            font-size: 14px;
            padding: 6px 12px;
          }
        }
      }

      // 子目录和函数区域
      .children-section {
        margin-top: 32px;

        .section-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 20px;

          .section-title {
            margin: 0;
            display: flex;
            align-items: center;
            gap: 10px;
            font-size: 20px;
            font-weight: 600;
            color: var(--el-text-color-primary);

            .section-icon {
              font-size: 22px;
              color: var(--el-color-primary);
            }
          }

          .section-badge {
            font-weight: 600;
            padding: 4px 12px;
          }
        }

        // DirectoryNodeWrapper 的样式 - 每个目录是一个完整的卡片，占据网格的一列
        :deep(.directory-wrapper-card) {
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 12px;
          padding: 20px;
          transition: all 0.3s ease;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
          width: 100%;
          box-sizing: border-box;

          &:hover {
            border-color: var(--el-color-primary-light-7);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
            transform: translateY(-2px);
          }

          .directory-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 16px;
            padding-bottom: 12px;
            border-bottom: 1px solid var(--el-border-color-lighter);

            .directory-header-left {
              display: flex;
              align-items: center;
              gap: 12px;
              flex: 1;
              min-width: 0;

              .directory-icon {
                width: 32px;
                height: 32px;
                object-fit: contain;
                flex-shrink: 0;
              }

              .directory-info {
                flex: 1;
                min-width: 0;

                .directory-name {
                  font-size: 18px;
                  font-weight: 600;
                  color: var(--el-text-color-primary);
                  margin-bottom: 4px;
                  line-height: 1.4;
                }

                .directory-path {
                  font-size: 13px;
                  color: var(--el-text-color-secondary);
                  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
                  word-break: break-all;
                }
              }
            }
          }

          // 目录内部的 content 使用网格布局，让函数和子目录横向排列
          .directory-content {
            display: grid !important;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)) !important;
            gap: 16px !important;
            width: 100% !important;
            margin-top: 16px;
          }
        }

        // 统一的 child-card 样式（用于目录容器内和根目录下）
        :deep(.directory-content .child-card),
        :deep(.children-grid .child-card),
        .children-grid .child-card {
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 12px;
          padding: 20px;
          transition: all 0.3s ease;
          cursor: pointer;
          width: 100%;
          box-sizing: border-box;

          &:hover {
            border-color: var(--el-color-primary-light-7);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
            transform: translateY(-2px);
          }

          .child-card-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 16px;

            .child-icon-wrapper {
              display: flex;
              align-items: center;
              justify-content: center;
              width: 48px;
              height: 48px;
              border-radius: 12px;
              flex-shrink: 0;

              &.package-type {
                background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                .child-icon-img {
                  width: 32px;
                  height: 32px;
                  object-fit: contain;
                }
              }

              &.function-type {
                background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                .child-icon {
                  font-size: 24px;
                  color: var(--el-color-success);
                }

                .child-icon-img {
                  width: 32px;
                  height: 32px;
                  object-fit: contain;
                }
              }

              &.file-type {
                background: linear-gradient(135deg, var(--el-color-info-light-8), var(--el-color-info-light-9));

                .child-icon {
                  font-size: 24px;
                  color: var(--el-color-info);
                }
              }
            }

            .child-type-tag {
              font-weight: 500;
            }
          }

          .child-card-body {
            .child-name {
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
              line-height: 1.5;
              word-break: break-word;
              margin-bottom: 8px;
            }

            .child-description {
              font-size: 13px;
              color: var(--el-text-color-secondary);
              line-height: 1.6;
              word-break: break-word;
              padding-top: 8px;
              border-top: 1px solid var(--el-border-color-lighter);
            }
          }
        }

        .children-grid {
          display: grid !important;
          grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)) !important;
          gap: 16px !important;
          width: 100% !important;
          
          // 目录卡片占据整行
          :deep(.directory-wrapper-card) {
            grid-column: 1 / -1; // 占据整行
            width: 100%;
          }

          .child-card {
            background: var(--el-bg-color);
            border: 1px solid var(--el-border-color-lighter);
            border-radius: 12px;
            padding: 20px;
            transition: all 0.3s ease;
            cursor: pointer;
            width: 100%;
            box-sizing: border-box;

            &:hover {
              border-color: var(--el-color-primary-light-7);
              box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
              transform: translateY(-2px);
            }

            .child-card-header {
              display: flex;
              align-items: center;
              justify-content: space-between;
              margin-bottom: 16px;

              .child-icon-wrapper {
                display: flex;
                align-items: center;
                justify-content: center;
                width: 48px;
                height: 48px;
                border-radius: 12px;
                flex-shrink: 0;

                &.package-type {
                  background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                  .child-icon-img {
                    width: 32px;
                    height: 32px;
                    object-fit: contain;
                  }
                }

                &.function-type {
                  background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                  .child-icon {
                    font-size: 24px;
                    color: var(--el-color-success);
                  }

                  .child-icon-img {
                    width: 32px;
                    height: 32px;
                    object-fit: contain;
                  }
                }

                &.file-type {
                  background: linear-gradient(135deg, var(--el-color-info-light-8), var(--el-color-info-light-9));

                  .child-icon {
                    font-size: 24px;
                    color: var(--el-color-info);
                  }
                }
              }

              .child-type-tag {
                font-weight: 500;
              }
            }

            .child-card-body {
              .child-name {
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
                line-height: 1.5;
                word-break: break-word;
                margin-bottom: 8px;
              }

              .child-description {
                font-size: 13px;
                color: var(--el-text-color-secondary);
                line-height: 1.6;
                word-break: break-word;
                padding-top: 8px;
                border-top: 1px solid var(--el-border-color-lighter);
              }
            }
          }
        }
      }

      .empty-state {
        margin-top: 60px;
      }
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .hub-directory-detail-view {
    .hero-section {
      padding: 24px 20px;

      .hero-content {
        flex-direction: column;
        align-items: stretch;
        gap: 16px;

        .hero-info {
          flex-direction: column;
          align-items: flex-start;
          gap: 16px;
        }
      }
    }

    .main-content {
      flex-direction: column;

      .tree-sidebar {
        width: 100%;
        border-right: none;
        border-bottom: 1px solid var(--el-border-color-lighter);
        max-height: 300px;
      }

      .detail-content {
        padding: 24px 20px;

        .overview-section {
          .overview-card {
            flex-direction: column;
            gap: 20px;

            .overview-divider {
              width: 100%;
              height: 1px;
              margin: 0;
            }
          }
        }

        .children-section {
          .children-grid {
            grid-template-columns: 1fr;
          }

          :deep(.directory-content) {
            grid-template-columns: 1fr;
          }
        }
      }
    }
  }
}

// 详情对话框样式
:deep(.detail-dialog) {
  .el-dialog__header {
    padding: 24px 24px 0;
    border-bottom: none;
  }

  .el-dialog__body {
    padding: 24px;
  }

  .el-dialog__footer {
    padding: 16px 24px 24px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

.detail-dialog-content {
  .detail-header {
    display: flex;
    align-items: flex-start;
    gap: 20px;
    padding-bottom: 24px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    margin-bottom: 24px;

    .detail-icon-wrapper {
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      width: 64px;
      height: 64px;
      border-radius: 16px;

      &.package-icon-wrapper {
        background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
      }

      &.function-icon-wrapper {
        background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));
      }

      &.file-icon-wrapper {
        background: linear-gradient(135deg, var(--el-color-info-light-8), var(--el-color-info-light-9));
      }

      .detail-icon-img {
        width: 40px;
        height: 40px;
        object-fit: contain;
      }

      .detail-icon {
        font-size: 32px;
        color: var(--el-color-primary);
      }
    }

    .detail-header-info {
      flex: 1;
      min-width: 0;

      .detail-title-row {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 8px;

        .detail-title {
          margin: 0;
          font-size: 20px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          line-height: 1.4;
        }

        .type-tag {
          flex-shrink: 0;
        }
      }

      .detail-path {
        margin: 0;
        font-size: 13px;
        color: var(--el-text-color-secondary);
        font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
        word-break: break-all;
        background: var(--el-fill-color-lighter);
        padding: 8px 12px;
        border-radius: 6px;
      }
    }
  }

  .detail-section {
    margin-bottom: 24px;

    &:last-child {
      margin-bottom: 0;
    }

    .section-title {
      margin: 0 0 12px 0;
      font-size: 14px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .section-content {
      margin: 0;
      font-size: 14px;
      color: var(--el-text-color-regular);
      line-height: 1.6;
      word-break: break-word;
    }

    .tags-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;

      .tag-item {
        margin: 0;
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
