<template>
  <div class="action-buttons">
    <!-- 链接区域：只有 1 个链接时直接显示，超过 1 个时使用下拉菜单 -->
    <template v-if="linkFields.length === 1">
      <LinkWidget
        :field="linkFields[0]"
        :value="convertToFieldValue(row[linkFields[0].code], linkFields[0])"
        :field-path="linkFields[0].code"
        mode="table-cell"
        class="action-link"
      />
    </template>
    
    <!-- 多个链接下拉菜单（超过 1 个时显示） -->
    <el-dropdown
      v-else-if="linkFields.length > 1"
      trigger="click"
      placement="bottom-end"
      @command="(fieldCode: string) => handleLinkClick(fieldCode)"
    >
      <el-button link type="primary" size="small" class="more-links-btn">
        <el-icon><More /></el-icon>
        链接
      </el-button>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="linkField in linkFields"
            :key="linkField.code"
            :command="linkField.code"
          >
            <div class="dropdown-link-content">
              <el-icon v-if="linkField.widget?.config?.icon" class="link-icon">
                <component :is="linkField.widget.config.icon" />
              </el-icon>
              <el-icon v-else class="link-icon internal-icon"><Right /></el-icon>
              <span>{{ getLinkText(linkField, row[linkField.code]) }}</span>
            </div>
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
    
    <!-- 删除按钮 -->
    <el-button 
      v-if="hasDeleteCallback"
      link 
      type="danger" 
      size="small"
      class="delete-btn"
      @click.stop="handleDeleteClick"
    >
      <el-icon><Delete /></el-icon>
      删除
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Delete, More, Right } from '@element-plus/icons-vue'
import { ElIcon, ElButton, ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus'
import { useRouter } from 'vue-router'
import { convertToFieldValue } from '@/utils/field'
import { resolveWorkspaceUrl } from '@/utils/route'
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import type { FieldConfig } from '@/core/types/field'

interface Props {
  /** 链接字段列表 */
  linkFields: FieldConfig[]
  /** 是否有删除回调 */
  hasDeleteCallback: boolean
  /** 行数据 */
  row: any
  /** 用户信息映射 */
  userInfoMap: Map<string, any>
}

const props = defineProps<Props>()

const router = useRouter()

const emit = defineEmits<{
  (e: 'link-click', fieldCode: string, row: any): void
  (e: 'delete', row: any): void
}>()

/**
 * 获取链接文本（用于下拉菜单显示）
 */
const getLinkText = (linkField: FieldConfig, rawValue: any): string => {
  const value = convertToFieldValue(rawValue, linkField)
  const url = value?.raw || ''
  if (!url) return linkField.name || '链接'
  
  // 解析 "[text]url" 格式
  const match = url.match(/^\[([^\]]+)\](.+)$/)
  if (match) {
    return match[1]  // 返回文本部分
  }
  
  // 如果没有文本，使用字段名称或配置的 text
  return linkField.widget?.config?.text || linkField.name || '链接'
}

/**
 * 处理链接点击（用于下拉菜单）
 */
const handleLinkClick = (fieldCode: string): void => {
  const linkField = props.linkFields.find((f: FieldConfig) => f.code === fieldCode)
  if (!linkField) return
  
  // 获取链接值
  const value = convertToFieldValue(props.row[fieldCode], linkField)
  const url = value?.raw || ''
  if (!url) return
  
  // 解析 "[text]url" 格式
  const match = url.match(/^\[([^\]]+)\](.+)$/)
  const actualUrl = match ? match[2] : url
  
  // 获取链接配置
  const linkConfig = linkField.widget?.config || {}
  const target = linkConfig.target || '_self'
  
  // 处理 URL，添加 /workspace 前缀
  const resolvedUrl = resolveWorkspaceUrl(actualUrl, router.currentRoute.value)
  
  // 根据 target 决定打开方式
  if (target === '_blank' || actualUrl.startsWith('http://') || actualUrl.startsWith('https://')) {
    window.open(resolvedUrl, '_blank')
  } else {
    router.push(resolvedUrl)
  }
  
  // 触发事件（如果需要）
  emit('link-click', fieldCode, props.row)
}

/**
 * 处理删除按钮点击
 */
const handleDeleteClick = (): void => {
  emit('delete', props.row)
}
</script>

<style scoped>
.action-buttons {
  position: relative;
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: nowrap;  /* 🔥 禁止换行，防止行高增加 */
  pointer-events: auto;
  width: 100%;  /* 使用 100% 宽度，确保内容完整显示 */
  min-width: 0;  /* 允许 flex 子元素收缩 */
}

.action-link {
  flex-shrink: 0;
  white-space: nowrap;  /* 防止文本换行 */
}

.more-links-btn {
  flex-shrink: 0;
  white-space: nowrap;
}

.delete-btn {
  flex-shrink: 0;  /* 🔥 防止删除按钮被压缩 */
  white-space: nowrap;  /* 防止文字换行 */
  min-width: fit-content;  /* 确保按钮内容完整显示 */
}

.dropdown-link-content {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
}

.dropdown-link-content .link-icon {
  font-size: 14px;
  color: var(--el-color-primary);
}
</style>

