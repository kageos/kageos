<template>
  <span
    :class="['mini-resource-identity', `is-${variant}`, iconMeta.className]"
    :title="identityTitle"
  >
    <span class="mini-resource-identity__icon" aria-hidden="true">
      <img
        v-if="iconMeta.src"
        :src="iconMeta.src"
        :alt="iconMeta.alt"
        class="mini-resource-identity__img"
      />
      <component
        :is="iconMeta.component"
        v-else
        class="mini-resource-identity__svg"
        :size="iconSize"
        :color="iconMeta.color"
      />
    </span>
    <span v-if="showName" class="mini-resource-identity__name">{{ displayName }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed, markRaw, type Component } from 'vue'
import { Document } from '@element-plus/icons-vue'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'

type ResourceIdentityVariant = 'pill' | 'message'

const RAW_TABLE_ICON = markRaw(TableIcon)
const RAW_CHART_ICON = markRaw(ChartIcon)
const RAW_DOCUMENT_ICON = markRaw(Document)

const props = withDefaults(defineProps<{
  name?: string
  fullCodePath?: string
  resourceType?: string
  resourceTemplateType?: string
  variant?: ResourceIdentityVariant
  showName?: boolean
}>(), {
  name: '',
  fullCodePath: '',
  resourceType: '',
  resourceTemplateType: '',
  variant: 'pill',
  showName: true,
})

const displayName = computed(() => {
  const name = props.name.trim()
  if (name) return name
  return resourceDisplayName(props.fullCodePath)
})

const normalizedTemplateType = computed(() => {
  const value = props.resourceTemplateType.trim()
  if (value) return value
  const path = props.fullCodePath.trim()
  if (path.endsWith('.form')) return TEMPLATE_TYPE.FORM
  if (path.endsWith('.table')) return TEMPLATE_TYPE.TABLE
  if (path.endsWith('.chart')) return TEMPLATE_TYPE.CHART
  return ''
})

const normalizedResourceType = computed(() => {
  const value = props.resourceType.trim()
  if (value) return value
  const path = props.fullCodePath.trim()
  if (path.endsWith('.docs') || path.includes('/docs/')) return 'docs'
  if (normalizedTemplateType.value) return 'function'
  return 'package'
})

const iconSize = computed(() => props.variant === 'message' ? 17 : 15)
const identityTitle = computed(() => props.fullCodePath || displayName.value)

const iconMeta = computed<{
  src?: string
  component?: Component
  alt: string
  className: string
  color?: string
}>(() => {
  if (normalizedResourceType.value === 'package') {
    return {
      src: '/service-tree/custom-folder.svg',
      alt: 'directory',
      className: 'is-package',
    }
  }
  if (normalizedResourceType.value === 'docs') {
    return {
      src: '/文档.svg',
      alt: 'docs',
      className: 'is-docs',
    }
  }
  if (normalizedTemplateType.value === TEMPLATE_TYPE.FORM) {
    return {
      src: '/service-tree/编辑.svg',
      alt: 'form',
      className: 'is-form',
    }
  }
  if (normalizedTemplateType.value === TEMPLATE_TYPE.TABLE) {
    return {
      component: RAW_TABLE_ICON,
      alt: 'table',
      className: 'is-table',
      color: '#10b981',
    }
  }
  if (normalizedTemplateType.value === TEMPLATE_TYPE.CHART) {
    return {
      component: RAW_CHART_ICON,
      alt: 'chart',
      className: 'is-chart',
      color: '#8b5cf6',
    }
  }
  return {
    component: RAW_DOCUMENT_ICON,
    alt: 'tool',
    className: 'is-function',
    color: '#6366f1',
  }
})

function resourceDisplayName(path: string) {
  const normalized = String(path || '').replace(/[<>]/g, '').replace(/\/+$/g, '')
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || normalized || 'Current'
}
</script>

<style scoped>
.mini-resource-identity {
  --mini-resource-icon-size: 18px;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.mini-resource-identity__icon {
  width: var(--mini-resource-icon-size);
  height: var(--mini-resource-icon-size);
  flex: 0 0 var(--mini-resource-icon-size);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
}

.mini-resource-identity__img,
.mini-resource-identity__svg {
  width: 100%;
  height: 100%;
  object-fit: contain;
  display: block;
}

.mini-resource-identity__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-resource-identity.is-message {
  --mini-resource-icon-size: 22px;
  max-width: min(240px, 50vw);
  padding: 2px 7px 2px 4px;
  border: 1px solid rgba(var(--color-primary-rgb), 0.18);
  border-radius: 999px;
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--text-primary);
  font-size: 11px;
  font-weight: 700;
  line-height: 1.25;
}

.mini-resource-identity.is-pill {
  color: var(--color-primary);
}

.mini-resource-identity.is-table .mini-resource-identity__icon {
  color: #10b981;
}

.mini-resource-identity.is-chart .mini-resource-identity__icon {
  color: #8b5cf6;
}
</style>
