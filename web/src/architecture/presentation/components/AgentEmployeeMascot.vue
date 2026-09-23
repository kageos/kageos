<template>
  <span
    :class="['agent-employee-mascot', `is-${variant}`, `is-${state}`]"
    :data-agent-state="state"
    :data-agent-variant="variant"
    role="img"
    :aria-label="label || defaultLabel"
    :title="label || defaultLabel"
  >
    <img :src="imageSource" alt="" aria-hidden="true" draggable="false" />
    <span v-if="variant === 'mark'" class="employee-status" aria-hidden="true" :data-status="state">
      <component :is="stateIcons[state]" />
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Loading, Clock, VideoPause, WarningFilled } from '@element-plus/icons-vue'
import readyEmployee from '@/architecture/presentation/assets/digital-employees/employee-ready.gif'
import workingEmployee from '@/architecture/presentation/assets/digital-employees/employee-working.gif'
import pausedEmployee from '@/architecture/presentation/assets/digital-employees/employee-paused.gif'
import failedEmployee from '@/architecture/presentation/assets/digital-employees/employee-failed.gif'
import serviceEmployeeIcon from '@/architecture/presentation/assets/digital-employees/service-icon-portrait.webp'

type AgentEmployeeState = 'working' | 'ready' | 'paused' | 'failed'
type AgentEmployeeVariant = 'mark' | 'employee'

const props = withDefaults(defineProps<{
  state?: AgentEmployeeState
  variant?: AgentEmployeeVariant
  label?: string
}>(), {
  state: 'ready',
  variant: 'employee',
  label: '',
})

const stateLabels: Record<AgentEmployeeState, string> = {
  working: '数字员工正在处理',
  ready: '数字员工正在待命',
  paused: '数字员工已暂停',
  failed: '数字员工需要关注',
}

const stateIcons = { working: Loading, ready: Clock, paused: VideoPause, failed: WarningFilled }

const employeeImages: Record<AgentEmployeeState, string> = {
  ready: readyEmployee,
  working: workingEmployee,
  paused: pausedEmployee,
  failed: failedEmployee,
}

const defaultLabel = computed(() => stateLabels[props.state])
const imageSource = computed(() => props.variant === 'mark' ? serviceEmployeeIcon : employeeImages[props.state])
</script>

<style scoped lang="scss">
.agent-employee-mascot {
  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  overflow: visible;
  line-height: 1;
  vertical-align: middle;
}

.agent-employee-mascot.is-mark {
  width: 24px;
  height: 24px;
}

.agent-employee-mascot.is-employee {
  width: 68px;
  height: 62px;
}

.agent-employee-mascot img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  user-select: none;
}

.agent-employee-mascot.is-mark img {
  border-radius: 50%;
}

.employee-status {
  position: absolute;
  right: -3px;
  bottom: -2px;
  display: grid;
  place-items: center;
  width: 12px;
  height: 12px;
  border: 2px solid var(--el-bg-color, #fff);
  border-radius: 50%;
  background: #18794e;
  color: #fff;
}

.employee-status svg {
  width: 10px;
  height: 10px;
}

.is-mark.is-working .employee-status { background: #3758db; }
.is-mark.is-working .employee-status svg { animation: employee-status-spin 1.2s linear infinite; }
.is-mark.is-paused .employee-status { background: #64748b; }
.is-mark.is-paused img { filter: saturate(0.35); }
.is-mark.is-failed .employee-status { background: #d92d20; }
.is-mark.is-failed img { outline: 2px solid #d92d20; outline-offset: 1px; }

@keyframes employee-status-spin {
  to { transform: rotate(360deg); }
}

@media (prefers-reduced-motion: reduce) {
  .is-mark.is-working .employee-status svg { animation: none; }
}

.agent-employee-mascot.is-employee.is-working {
  filter: drop-shadow(0 8px 15px rgba(59, 130, 246, 0.22));
}

.agent-employee-mascot.is-employee.is-ready {
  filter: drop-shadow(0 8px 15px rgba(16, 185, 129, 0.18));
}

.agent-employee-mascot.is-employee.is-paused {
  filter: drop-shadow(0 8px 15px rgba(245, 158, 11, 0.18));
}

.agent-employee-mascot.is-employee.is-failed {
  filter: drop-shadow(0 8px 15px rgba(239, 68, 68, 0.2));
}
</style>
