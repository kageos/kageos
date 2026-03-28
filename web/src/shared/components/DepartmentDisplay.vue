<!--
  DepartmentDisplay - 通用组织架构展示组件
  功能：
  - 简单模式：只显示组织架构名称（用于列表、详情等）
  - 详细模式：点击显示完整组织架构信息卡片（使用 el-popover）
  
  显示风格：
  - horizontal：水平布局，图标在左，名称在右（适用于 table、详情字段等）
  - vertical：垂直布局，图标在上，名称在下（适用于特殊场景）
  
  使用场景：
  - Form 输出组织架构字段（horizontal）
  - Table 表格中显示组织架构（horizontal）
  - 详情中显示组织架构信息（horizontal）
  - 用户编辑页面显示组织架构（horizontal）
-->
<template>
  <div v-if="displayName" class="department-display-wrapper">
    <!-- 简单模式：只显示组织架构名称 -->
    <div v-if="mode === 'simple'" class="department-display-simple" :class="[sizeClass, layoutClass]">
      <img src="/组织架构.svg" alt="组织架构" class="department-icon" :style="{ width: iconSize + 'px', height: iconSize + 'px' }" />
      <span class="department-name">{{ displayName }}</span>
    </div>
    
    <!-- 详细模式：点击弹出组织架构信息卡片 -->
    <div v-else-if="mode === 'card'" class="department-display-card" :class="[sizeClass, layoutClass]">
      <el-popover
        v-if="actualDepartmentInfo"
        placement="bottom-start"
        :width="650"
        trigger="click"
        popper-class="department-info-popover"
      >
        <template #reference>
          <div class="department-display-simple" style="cursor: pointer;">
            <img src="/组织架构.svg" alt="组织架构" class="department-icon" :style="{ width: iconSize + 'px', height: iconSize + 'px' }" />
            <span class="department-name">{{ displayName }}</span>
          </div>
        </template>
        <DepartmentDetailCard 
          :department-info="actualDepartmentInfo" 
          :department-tree="departmentTree"
          :current-path="actualDepartmentInfo?.full_code_path || fullCodePath"
        />
      </el-popover>
      <!-- 如果没有组织架构信息，只显示图标和名称（不可点击） -->
      <div v-else class="department-display-simple" :class="[sizeClass, layoutClass]">
        <img src="/组织架构.svg" alt="组织架构" class="department-icon" :style="{ width: iconSize + 'px', height: iconSize + 'px' }" />
        <span class="department-name">{{ displayName }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref, onMounted } from 'vue'
import { ElPopover } from 'element-plus'
import type { Department } from '@/api/department'
import { getDepartmentTree } from '@/api/department'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import DepartmentDetailCard from './DepartmentDetailCard.vue'

interface Props {
  /** 组织架构信息对象 */
  departmentInfo?: Department | null
  /** 组织架构完整路径（当 departmentInfo 不存在时使用） */
  fullCodePath?: string | null
  /** 组织架构显示名称（优先使用，避免异步加载时显示英文路径） */
  displayName?: string | null
  /** 显示模式：simple（简单模式，只显示名称）或 card（详细模式，点击显示卡片） */
  mode?: 'simple' | 'card'
  /** 显示风格：horizontal（水平布局，图标在左名称在右）或 vertical（垂直布局，图标在上名称在下） */
  layout?: 'horizontal' | 'vertical'
  /** 图标大小：small(16px) | medium(20px) | large(24px) | 自定义数字 */
  size?: 'small' | 'medium' | 'large' | number
  /** 部门树（可选，如果传入则使用传入的，避免重复加载） */
  departmentTree?: Department[]
  /** 是否显示全路径名称（默认 false，显示最后一段名称，精简显示） */
  showFullPath?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  departmentInfo: null,
  fullCodePath: null,
  displayName: null,
  mode: 'simple',
  layout: 'horizontal',
  size: 'medium',
  departmentTree: () => [],
  showFullPath: false
})

// 🔥 使用部门信息缓存 Store
const departmentInfoStore = useDepartmentInfoStore()

// 使用 ref 存储组织架构信息，确保响应式更新
const cachedDepartmentInfo = ref<Department | null>(null)
// 使用 ref 存储部门树（用于详情卡片显示）
const internalDepartmentTree = ref<Department[]>([])
// 使用 computed 来获取部门树：优先使用 props.departmentTree
const departmentTree = computed(() => {
  // 如果传入了 departmentTree prop，优先使用
  if (props.departmentTree && props.departmentTree.length > 0) {
    return props.departmentTree
  }
  // 否则使用内部加载的部门树（用于 card 模式）
  return internalDepartmentTree.value
})

// 更新缓存的组织架构信息
const updateCachedDepartmentInfo = async () => {
  // 优先使用 props.departmentInfo
  if (props.departmentInfo) {
    cachedDepartmentInfo.value = props.departmentInfo
    return
  }
  
  // 🔥 如果有 fullCodePath，优先从 store 缓存读取
  if (props.fullCodePath) {
    try {
      // 🔥 先从 store 缓存中获取（如果预加载已完成，这里会命中缓存）
      const cachedDept = departmentInfoStore.departmentInfoCache.get(props.fullCodePath)
      if (cachedDept) {
        cachedDepartmentInfo.value = cachedDept
        return
      }
      
      // 🔥 如果缓存中没有，使用 getDepartmentInfo（会自动处理缓存和降级策略）
      // store 内部会调用批量接口，不需要我们手动调用 getDepartmentTree
      const dept = await departmentInfoStore.getDepartmentInfo(props.fullCodePath)
      if (dept) {
        cachedDepartmentInfo.value = dept
        return
      }
      
      // 如果 store 中也没有，说明部门不存在
      cachedDepartmentInfo.value = null
    } catch (error) {
      console.error('[DepartmentDisplay] 加载组织架构信息失败', error)
      cachedDepartmentInfo.value = null
    }
    return
  }
  
  cachedDepartmentInfo.value = null
}

// 组织架构信息（从缓存的 ref 中获取）
const actualDepartmentInfo = computed(() => {
  return cachedDepartmentInfo.value
})

// 监听 departmentInfo 和 fullCodePath 的变化，更新缓存的组织架构信息
watch([() => props.departmentInfo, () => props.fullCodePath], () => {
  updateCachedDepartmentInfo()
}, { immediate: true, deep: false })

// 计算图标大小
const iconSize = computed(() => {
  if (typeof props.size === 'number') {
    return props.size
  }
  const sizeMap: Record<'small' | 'medium' | 'large', number> = {
    small: 16,
    medium: 20,
    large: 24,
  }
  return sizeMap[props.size as 'small' | 'medium' | 'large']
})

// 计算尺寸类名
const sizeClass = computed(() => {
  if (typeof props.size === 'number') {
    return ''
  }
  return `department-display-${props.size}`
})

// 计算布局类名
const layoutClass = computed(() => {
  return `department-layout-${props.layout}`
})

// 计算显示名称
const displayName = computed(() => {
  // 优先使用传入的 displayName（避免异步加载时显示英文路径）
  if (props.displayName) {
    return props.displayName
  }
  
  const dept = actualDepartmentInfo.value
  if (dept) {
    // 根据 showFullPath 配置决定显示全路径还是最后一段
    if (props.showFullPath) {
      return dept.full_name_path || dept.name
    } else {
      // 默认显示最后一段（精简显示）
      return dept.name
    }
  }
  
  // 如果只有路径且没有显示名称，显示"加载中..."而不是英文路径
  if (props.fullCodePath) {
    return '加载中...'
  }
  
  // 没有数据时返回 null，不显示任何内容
  return null
})

// 加载部门树（用于详情卡片显示）
const loadDepartmentTreeForCard = async () => {
  // 如果已经传入了 departmentTree prop，不需要加载
  if (props.departmentTree && props.departmentTree.length > 0) {
    return
  }
  
  // 如果没有传入且需要显示详情卡片，加载部门树
  if (props.mode === 'card' && departmentTree.value.length === 0) {
    try {
      const treeRes = await getDepartmentTree()
      internalDepartmentTree.value = treeRes.departments || []
    } catch (error) {
      console.error('[DepartmentDisplay] 加载部门树失败', error)
    }
  }
}

// 组件挂载时，更新缓存信息
// 🔥 不再需要加载部门树，store 会处理所有缓存逻辑
onMounted(async () => {
  await updateCachedDepartmentInfo()
  // 如果是 card 模式，加载部门树用于详情卡片
  if (props.mode === 'card') {
    await loadDepartmentTreeForCard()
  }
})
</script>

<style scoped>
.department-display-wrapper {
  display: inline-flex;
  align-items: center;
}

/* 简单模式 */
.department-display-simple {
  display: flex;
}

/* 水平布局：图标在左，名称在右 */
.department-layout-horizontal {
  flex-direction: row;
  align-items: center;
  gap: 8px;
}

/* 垂直布局：图标在上，名称在下 */
.department-layout-vertical {
  flex-direction: column;
  align-items: center;
  gap: 6px;
  justify-content: center;
}

.department-display-simple .department-icon {
  flex-shrink: 0;
  opacity: 0.8;
}

.department-display-simple .department-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

/* 垂直布局下的名称样式 */
.department-layout-vertical .department-name {
  font-size: 12px;
  text-align: center;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.2;
  display: block;
}

.department-icon {
  flex-shrink: 0;
}
</style>

<style>
/* Popover 全局样式 */
.department-info-popover {
  padding: 0 !important;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}
</style>
