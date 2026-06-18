<template>
  <aside
    class="mini-artifact-panel"
    :class="{ 'is-empty': artifactItems.length === 0, 'is-maximized': maximized }"
    aria-label="当前产物"
  >
    <div class="mini-artifact-head">
      <span>{{ maximized ? '当前产物' : '产物' }}</span>
      <strong>{{ artifactItems.length }} 项</strong>
      <span class="mini-artifact-actions">
        <el-dropdown
          v-if="panelHasContent"
          ref="keyInfoDropdownRef"
          trigger="click"
          placement="top-end"
          popper-class="mini-files-dropdown-popper"
          :hide-on-click="false"
          @visible-change="onKeyInfoDropdownVisibleChange"
        >
          <button type="button" class="mini-icon-action" title="查看关键信息">
            <el-icon :size="14"><DocumentIcon /></el-icon>
            <span class="mini-header-files-count">{{ panelItemCount }}</span>
          </button>
          <template #dropdown>
            <div class="mini-files-dropdown-panel">
              <div class="mini-files-dropdown-title">关键信息</div>
              <MiniWorkstationKeyInfoSection
                compact
                :uploaded-files="uploadedFiles"
                :output-files="outputFiles"
                :display-fields="displayFields"
                @preview-file="$emit('previewFile', $event)"
                @download-file="$emit('downloadFile', $event)"
                @preview-field="$emit('previewField', $event)"
                @copy-field="$emit('copyField', $event)"
              />
            </div>
          </template>
        </el-dropdown>
      </span>
    </div>

    <button
      v-for="item in artifactItems"
      :key="item.key"
      type="button"
      class="mini-artifact-item"
      :class="`is-${item.tone}`"
      @click="$emit('artifactClick', item)"
    >
      <span class="mini-artifact-preview" :class="`is-${item.tone}`">
        <img v-if="item.previewUrl" :src="item.previewUrl" :alt="item.name" loading="lazy" />
        <template v-else>
          <el-icon :size="maximized ? 22 : 16">
            <component :is="item.iconComponent" />
          </el-icon>
          <span v-if="item.ext" class="mini-artifact-ext">{{ item.ext }}</span>
        </template>
      </span>
      <span class="mini-artifact-copy">
        <span class="mini-artifact-name">{{ item.name }}</span>
        <span class="mini-artifact-meta">{{ item.meta }}</span>
      </span>
      <span class="mini-artifact-tag">
        {{ item.tag }}
        <span v-if="!maximized && artifactItems.length > 1" class="mini-artifact-mini-count">{{ artifactItems.length }}项</span>
      </span>
    </button>

    <div v-if="artifactItems.length === 0" class="mini-artifact-empty">
      <span>暂无产物</span>
      <em>等待</em>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Document as DocumentIcon } from '@element-plus/icons-vue'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { FilePanelItem } from '../composables/useMiniWorkstationPanel'
import type { MiniArtifactItem } from '../composables/useMiniWorkstationArtifacts'
import MiniWorkstationKeyInfoSection from './MiniWorkstationKeyInfoSection.vue'

const props = defineProps<{
  artifactItems: MiniArtifactItem[]
  maximized: boolean
  panelHasContent: boolean
  panelItemCount: number
  uploadedFiles: FilePanelItem[]
  outputFiles: FilePanelItem[]
  displayFields: OutputDisplayField[]
  displayFieldPreviewVisible: boolean
}>()

defineEmits<{
  (e: 'artifactClick', item: MiniArtifactItem): void
  (e: 'previewFile', file: FilePanelItem): void
  (e: 'downloadFile', file: FilePanelItem): void
  (e: 'previewField', field: OutputDisplayField): void
  (e: 'copyField', field: OutputDisplayField): void
}>()

const keyInfoDropdownRef = ref<any>(null)

function onKeyInfoDropdownVisibleChange(visible: boolean) {
  if (!visible && props.displayFieldPreviewVisible) {
    setTimeout(() => { keyInfoDropdownRef.value?.handleOpen?.() }, 50)
  }
}
</script>

<style scoped>
.mini-artifact-panel {
  min-height: 36px;
  overflow: hidden;
  padding-left: 12px;
  border-left: 1px solid rgba(130, 153, 190, 0.18);
}

.mini-artifact-panel.is-maximized {
  overflow: auto;
  padding: 12px;
  border: 1px solid rgba(130, 153, 190, 0.18);
  border-radius: 12px;
  background: rgba(12, 20, 35, 0.48);
}

.mini-artifact-head {
  min-height: 24px;
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: flex-start;
  margin-bottom: 8px;
  color: var(--mini-cyber-muted);
  font-size: 12px;
}

.mini-artifact-head strong {
  color: #8ed0ff;
  white-space: nowrap;
}

.mini-artifact-actions {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.mini-artifact-item {
  width: 100%;
  height: 42px;
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border: 1px solid rgba(83, 174, 255, 0.18);
  border-radius: 8px;
  background: rgba(14, 27, 45, 0.62);
  color: inherit;
  text-align: left;
}

.mini-artifact-item:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(24, 51, 83, 0.52);
}

.mini-artifact-item + .mini-artifact-item {
  margin-top: 8px;
}

.mini-artifact-panel.is-maximized .mini-artifact-item {
  height: auto;
  min-height: 66px;
  padding: 10px;
  grid-template-columns: 48px minmax(0, 1fr) auto;
}

.mini-artifact-preview {
  width: 30px;
  height: 30px;
  min-width: 0;
  position: relative;
  display: inline-grid;
  place-items: center;
  overflow: hidden;
  border: 1px solid rgba(130, 153, 190, 0.18);
  border-radius: 8px;
  background: rgba(10, 16, 29, 0.62);
  color: #8ed0ff;
}

.mini-artifact-panel.is-maximized .mini-artifact-preview {
  width: 48px;
  height: 48px;
  border-radius: 10px;
}

.mini-artifact-preview img {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

.mini-artifact-preview.is-image {
  border-color: rgba(83, 174, 255, 0.28);
  background: rgba(24, 48, 77, 0.46);
}

.mini-artifact-preview.is-data {
  color: #7df5c4;
  background: rgba(21, 54, 50, 0.42);
}

.mini-artifact-preview.is-document {
  color: #bcb7ff;
  background: rgba(41, 38, 76, 0.46);
}

.mini-artifact-preview.is-media {
  color: #ffd78d;
  background: rgba(58, 45, 24, 0.46);
}

.mini-artifact-preview.is-archive,
.mini-artifact-preview.is-file {
  color: #b9c9e4;
  background: rgba(41, 48, 64, 0.46);
}

.mini-artifact-preview.is-field {
  color: #8ed0ff;
  background: rgba(24, 51, 83, 0.46);
}

.mini-artifact-ext {
  position: absolute;
  right: 2px;
  bottom: 2px;
  max-width: calc(100% - 4px);
  overflow: hidden;
  padding: 1px 3px;
  border-radius: 4px;
  background: rgba(2, 7, 15, 0.72);
  color: #ffffff;
  font-size: 8px;
  font-weight: 900;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-artifact-panel.is-maximized .mini-artifact-ext {
  font-size: 9px;
}

.mini-artifact-copy,
.mini-artifact-name,
.mini-artifact-meta {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-artifact-name {
  color: #dce9fb;
  font-size: 12px;
  font-weight: 760;
}

.mini-artifact-meta {
  margin-top: 2px;
  color: #8a9ab6;
  font-size: 11px;
}

.mini-artifact-tag {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border-radius: 7px;
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
  font-size: 11px;
  font-weight: 800;
  white-space: nowrap;
}

.mini-artifact-item.is-data .mini-artifact-tag {
  background: rgba(43, 213, 159, 0.14);
  color: #7df5c4;
}

.mini-artifact-item.is-document .mini-artifact-tag {
  background: rgba(119, 107, 255, 0.16);
  color: #bcb7ff;
}

.mini-artifact-item.is-media .mini-artifact-tag {
  background: rgba(246, 189, 77, 0.15);
  color: #ffd78d;
}

.mini-artifact-item.is-archive .mini-artifact-tag,
.mini-artifact-item.is-file .mini-artifact-tag {
  background: rgba(142, 159, 187, 0.14);
  color: #b9c9e4;
}

.mini-artifact-item.is-field .mini-artifact-tag {
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
}

.mini-artifact-mini-count {
  margin-left: 6px;
  padding-left: 6px;
  border-left: 1px solid rgba(125, 245, 196, 0.24);
  color: #dff7ef;
}

.mini-artifact-empty {
  height: 32px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 1px solid rgba(83, 174, 255, 0.14);
  border-radius: 8px;
  background: rgba(14, 27, 45, 0.42);
  color: #8a9ab6;
  font-size: 12px;
}

.mini-artifact-empty em {
  border-radius: 7px;
  padding: 3px 7px;
  background: rgba(142, 159, 187, 0.12);
  color: #9fb0cb;
  font-style: normal;
  font-weight: 800;
}

.mini-files-dropdown-panel {
  min-width: 260px;
  max-width: 320px;
  color: var(--mini-cyber-text);
  background:
    radial-gradient(circle at 12% 0%, rgba(55, 163, 255, 0.16), transparent 34%),
    linear-gradient(150deg, rgba(8, 13, 24, 0.98), rgba(17, 25, 45, 0.96));
  border: 1px solid rgba(130, 153, 190, 0.24);
  border-radius: 14px;
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.36), 0 0 24px rgba(34, 211, 238, 0.1);
  overflow: hidden;
}

.mini-files-dropdown-title {
  padding: 11px 12px;
  font-size: 13px;
  font-weight: 700;
  color: var(--mini-cyber-text);
  border-bottom: 1px solid rgba(96, 231, 255, 0.16);
  background: rgba(34, 211, 238, 0.06);
}

@media (max-width: 820px) {
  .mini-artifact-panel {
    display: none;
  }
}
</style>
