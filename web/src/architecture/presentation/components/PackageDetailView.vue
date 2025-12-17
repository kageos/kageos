<!--
  PackageDetailView - 服务目录详情页面
  
  职责：
  - 显示服务目录信息
  - 提供"生成系统"按钮，点击后打开智能体选择对话框
-->
<template>
  <div class="package-detail-view">
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
            <el-icon class="hero-icon"><Folder /></el-icon>
          </div>
          <div class="hero-text">
            <h1 class="hero-title">{{ packageNode?.name || '服务目录' }}</h1>
            <p class="hero-subtitle" v-if="packageNode?.full_code_path">
              <el-icon class="path-icon"><Link /></el-icon>
              <span class="path-text">{{ packageNode.full_code_path }}</span>
              <el-button 
                text 
                :icon="CopyDocument" 
                @click="handleCopyPath"
                class="path-copy-btn"
                size="small"
                title="复制路径"
              />
            </p>
          </div>
        </div>
        <el-button 
          type="primary" 
          :icon="MagicStick" 
          @click="handleGenerateSystem"
          size="large"
          class="action-button"
        >
          生成系统
        </el-button>
      </div>
    </div>
    
    <div class="detail-content">
      <!-- 信息概览卡片 -->
      <div v-if="packageNode" class="overview-section">
        <div class="overview-card">
          <div class="overview-item">
            <div class="overview-icon-wrapper name-icon">
              <el-icon class="overview-icon"><Document /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">目录名称</div>
              <div class="overview-value">{{ packageNode.name }}</div>
            </div>
          </div>
          
          <div class="overview-divider"></div>
          
          <div class="overview-item">
            <div class="overview-icon-wrapper code-icon">
              <el-icon class="overview-icon"><Key /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">目录代码</div>
              <div class="overview-value code-text">{{ packageNode.code }}</div>
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
                {{ packageNode?.children?.length || 0 }} 项
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 子目录和函数列表 -->
      <div class="children-section" v-if="packageNode?.children && packageNode.children.length > 0">
        <div class="section-header">
          <h3 class="section-title">
            <el-icon class="section-icon"><Files /></el-icon>
            子目录和函数
          </h3>
          <el-tag class="section-badge" type="primary" size="small">
            {{ packageNode.children.length }}
          </el-tag>
        </div>
        
        <div class="children-grid">
          <div
            v-for="child in packageNode.children"
            :key="child.id"
            class="child-card"
          >
            <div class="child-card-header">
              <div class="child-icon-wrapper" :class="child.type === 'package' ? 'package-type' : 'function-type'">
                <el-icon class="child-icon">
                  <Folder v-if="child.type === 'package'" />
                  <Document v-else />
                </el-icon>
              </div>
              <el-tag 
                v-if="child.type === 'function'" 
                size="small" 
                :type="getTemplateTypeTag(child.template_type)"
                class="child-type-tag"
              >
                {{ getTemplateTypeText(child.template_type) }}
              </el-tag>
            </div>
            <div class="child-card-body">
              <div class="child-name">{{ child.name }}</div>
            </div>
          </div>
        </div>
      </div>
      
      <el-empty 
        v-else 
        description="该目录下暂无子目录或函数" 
        :image-size="120"
        class="empty-state"
      />
    </div>
    
    <!-- 智能体选择对话框 -->
    <AgentSelectDialog
      v-model="agentSelectDialogVisible"
      :tree-id="packageNode?.id || null"
      :package="packageNode?.code || ''"
      :current-node-name="packageNode?.name || ''"
      @confirm="handleAgentSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, MagicStick, Folder, Document, CopyDocument, Key, Link, Files } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'
import type { AgentInfo } from '@/api/agent'
import { extractWorkspacePath } from '@/utils/route'

interface Props {
  packageNode?: ServiceTree | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'generate-system': [agent: AgentInfo]
}>()

const router = useRouter()
const route = useRoute()

const agentSelectDialogVisible = ref(false)
const selectedAgent = ref<AgentInfo | null>(null)

// 返回上一级
function handleBack() {
  // 获取当前路径，去掉最后一段
  const currentPath = extractWorkspacePath(route.path)
  if (currentPath) {
    const pathSegments = currentPath.split('/').filter(Boolean)
    if (pathSegments.length > 2) {
      // 至少是 user/app/package，去掉最后一段
      pathSegments.pop()
      const parentPath = `/workspace/${pathSegments.join('/')}`
      router.push(parentPath)
    } else {
      // 回到根目录
      router.push('/workspace')
    }
  } else {
    router.push('/workspace')
  }
}

// 打开生成系统对话框
function handleGenerateSystem() {
  agentSelectDialogVisible.value = true
}

// 选择智能体后的处理
function handleAgentSelect(agent: AgentInfo) {
  selectedAgent.value = agent
  // 触发生成系统事件，让父组件处理
  emit('generate-system', agent)
  agentSelectDialogVisible.value = false
}

// 复制完整路径
async function handleCopyPath() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }
  
  try {
    await navigator.clipboard.writeText(props.packageNode.full_code_path)
    ElMessage.success('路径已复制到剪贴板')
  } catch (error) {
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = props.packageNode.full_code_path
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
function getTemplateTypeTag(templateType: string): string {
  const typeMap: Record<string, string> = {
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
</script>

<style scoped lang="scss">
.package-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
  
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
          align-items: center;
          justify-content: center;
          width: 64px;
          height: 64px;
          background: var(--el-color-primary);
          border-radius: 16px;
          box-shadow: 0 4px 12px rgba(64, 158, 255, 0.2);
          
          .hero-icon {
            font-size: 32px;
            color: #fff;
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
            margin: 0;
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
          }
        }
      }
      
      .action-button {
        flex-shrink: 0;
        padding: 12px 24px;
        font-size: 15px;
        font-weight: 500;
      }
    }
  }
  
  .detail-content {
    flex: 1;
    overflow-y: auto;
    padding: 32px 40px;
    max-width: 1400px;
    margin: 0 auto;
    width: 100%;
    
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
    
    // 子目录和函数区域
    .children-section {
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
      
      .children-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 16px;
        
        .child-card {
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 12px;
          padding: 20px;
          transition: all 0.3s ease;
          cursor: pointer;
          
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
              
              &.package-type {
                background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
                
                .child-icon {
                  font-size: 24px;
                  color: var(--el-color-primary);
                }
              }
              
              &.function-type {
                background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));
                
                .child-icon {
                  font-size: 24px;
                  color: var(--el-color-success);
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

// 响应式设计
@media (max-width: 768px) {
  .package-detail-view {
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
        
        .action-button {
          width: 100%;
        }
      }
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
      }
    }
  }
}
</style>

