<template>
  <span v-if="!hasDuration" class="duration-empty">{{ placeholder }}</span>
  <el-tooltip v-else :content="tooltipText" placement="top" effect="light">
    <el-tag :type="tagType" size="small" effect="light">
      {{ durationText }}
    </el-tag>
  </el-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElTag, ElTooltip } from 'element-plus'
import {
  formatExecutionDuration,
  getExecutionDurationTagType,
  getExecutionDurationTip
} from '@/architecture/presentation/utils/executionLog'

const props = withDefaults(
  defineProps<{
    duration?: number | null
    placeholder?: string
  }>(),
  {
    duration: null,
    placeholder: '-'
  }
)

const hasDuration = computed(() => props.duration !== null && props.duration !== undefined && props.duration >= 0)
const durationText = computed(() => formatExecutionDuration(props.duration))
const tooltipText = computed(() => getExecutionDurationTip(props.duration))
const tagType = computed(() => getExecutionDurationTagType(props.duration))
</script>

<style scoped>
.duration-empty {
  color: var(--el-text-color-placeholder);
}
</style>
