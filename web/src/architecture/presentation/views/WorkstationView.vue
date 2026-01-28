<template>
  <div class="workstation-view">
    <WorkstationChat :full-code-path="fullCodePath" :embedded="false" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkstationChat from '../components/WorkstationChat.vue'

const route = useRoute()
const fullCodePath = computed(() => (route.query.full_code_path as string) || '')
</script>

<style scoped>
.workstation-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 100vh;
  background: var(--el-bg-color-page);
}
</style>
