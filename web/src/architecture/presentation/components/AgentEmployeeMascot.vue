<template>
  <span
    :class="['agent-employee-mascot', `is-${variant}`, `is-${state}`]"
    :data-agent-state="state"
    :data-agent-variant="variant"
    role="img"
    :aria-label="label || defaultLabel"
  >
    <img :src="imageSource" alt="" aria-hidden="true" draggable="false" />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import readyEmployee from '@/architecture/presentation/assets/digital-employees/employee-ready.gif'
import workingEmployee from '@/architecture/presentation/assets/digital-employees/employee-working.gif'
import pausedEmployee from '@/architecture/presentation/assets/digital-employees/employee-paused.gif'
import failedEmployee from '@/architecture/presentation/assets/digital-employees/employee-failed.gif'
import serviceEmployeeIcon from '@/architecture/presentation/assets/digital-employees/service-icon.webp'

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
