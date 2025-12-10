<template>
  <div class="home-container">
    <!-- 视角切换器 -->
    <div class="perspective-switcher">
      <div class="switcher-container">
        <div 
          class="switcher-option"
          :class="{ active: perspective === 'non-tech' }"
          @click="switchPerspective('non-tech')"
        >
          <span class="switcher-icon">👤</span>
          <span class="switcher-label">普通用户</span>
        </div>
        <div 
          class="switcher-option"
          :class="{ active: perspective === 'tech' }"
          @click="switchPerspective('tech')"
        >
          <span class="switcher-icon">👨‍💻</span>
          <span class="switcher-label">技术用户</span>
        </div>
      </div>
    </div>

    <!-- 背景装饰 -->
    <div class="background-decoration">
      <div class="decoration-circle circle-1 animate-float"></div>
      <div class="decoration-circle circle-2 animate-float animate-delay-500"></div>
      <div class="decoration-circle circle-3 animate-float animate-delay-1000"></div>
      <div class="decoration-grid"></div>
    </div>

    <!-- 主要内容 -->
    <div class="main-content">
      <!-- Hero 区域 -->
      <div class="hero-section">
        <!-- 非技术视角 -->
        <template v-if="perspective === 'non-tech'">
          <div class="hero-badge animate-fade-in">
            <span class="badge-icon">✨</span>
            <span class="badge-text">用描述生成应用，用克隆复制应用</span>
          </div>
          
          <h1 class="hero-title animate-fade-in-up">
            <span class="title-line">一句话</span>
            <span class="title-line title-gradient">生成专业应用</span>
          </h1>
          
          <p class="hero-subtitle animate-fade-in-up animate-delay-200">
            <span class="highlight-text">描述即生成，克隆即复制</span>
          </p>
          
          <p class="hero-description animate-fade-in-up animate-delay-300">
            不需要懂编程，只需要用自然语言描述你的需求，AI就能帮你生成专业应用。
            <br />
            看到别人好的应用？一键克隆，立即拥有。还可以根据自己的需求修改，生成新的变体。
          </p>
        </template>
        
        <!-- 技术视角 -->
        <template v-else>
          <div class="hero-badge animate-fade-in">
            <span class="badge-icon">🚀</span>
            <span class="badge-text">下一代软件构建与运行系统</span>
          </div>
        
        <h1 class="hero-title animate-fade-in-up">
          <span class="title-line">AI原生</span>
          <span class="title-line title-gradient">一站式应用构建与运行平台</span>
        </h1>
        
        <p class="hero-subtitle animate-fade-in-up animate-delay-200">
          <span class="highlight-text">描述即生成，所述即所得</span>
        </p>
        
        <p class="hero-description animate-fade-in-up animate-delay-300">
          让每个人都能成为创造者。用自然语言描述需求，AI瞬间生成专业应用。
          <br />
          生成的软件像Docker镜像一样，一次构建，到处运行。支持私有化部署，统一登录，无需重复注册。
        </p>
        
        <!-- GitHub 开源标识 -->
        <div class="github-badge animate-fade-in-up animate-delay-300">
          <el-button 
            size="default" 
            @click="openGitHub"
            class="github-btn"
          >
            <span class="github-icon">🔓</span>
            <span>完全开源 - GitHub</span>
          </el-button>
        </div>
        
        <!-- 功能按钮 -->
        <div class="action-buttons animate-fade-in-up animate-delay-400">
          <el-button 
            type="primary" 
            size="large" 
            @click="goToWorkspace"
            class="action-btn primary-btn hover-lift"
          >
            <el-icon><Cpu /></el-icon>
            开始构建应用
          </el-button>
          <el-button 
            size="large" 
            @click="scrollToFeatures"
            class="action-btn secondary-btn hover-lift"
          >
            <el-icon><View /></el-icon>
            探索功能
          </el-button>
        </div>
        
        <!-- 核心能力展示 -->
        <div class="core-capabilities animate-fade-in-up animate-delay-500">
          <div class="capability-item">
            <div class="capability-icon">🤖</div>
            <div class="capability-text">
              <div class="capability-label">AI驱动</div>
              <div class="capability-desc">智能理解需求</div>
            </div>
          </div>
          <div class="capability-item">
            <div class="capability-icon">📦</div>
            <div class="capability-text">
              <div class="capability-label">Docker镜像</div>
              <div class="capability-desc">像容器一样部署</div>
            </div>
          </div>
          <div class="capability-item">
            <div class="capability-icon">🔐</div>
            <div class="capability-text">
              <div class="capability-label">统一登录</div>
              <div class="capability-desc">无需重复注册</div>
            </div>
          </div>
          <div class="capability-item">
            <div class="capability-icon">🏠</div>
            <div class="capability-text">
              <div class="capability-label">私有化部署</div>
              <div class="capability-desc">完全自主可控</div>
            </div>
          </div>
          <div class="capability-item">
            <div class="capability-icon">🌐</div>
            <div class="capability-text">
              <div class="capability-label">一站式</div>
              <div class="capability-desc">构建到运行</div>
            </div>
          </div>
          <div class="capability-item">
            <div class="capability-icon">🔓</div>
            <div class="capability-text">
              <div class="capability-label">完全开源</div>
              <div class="capability-desc">GitHub开源</div>
            </div>
          </div>
        </div>
        </template>
      </div>

      <!-- 闪电克隆流程展示（仅非技术视角） -->
      <div v-if="perspective === 'non-tech'" class="clone-flow-section">
        <div class="section-header">
          <h2 class="section-title">⚡ 闪电克隆，应用生态</h2>
          <p class="section-subtitle">一次描述开发，无限次克隆变体，形成强大的应用生态</p>
        </div>
        
        <div class="clone-flow-container">
          <el-steps 
            :active="cloneFlowSteps.length - 1" 
            direction="vertical"
            finish-status="success"
            class="clone-flow-steps"
          >
            <el-step 
              v-for="(step, index) in cloneFlowSteps" 
              :key="index"
              :title="step.title"
              :description="step.description"
              :status="index === 0 ? 'success' : index === cloneFlowSteps.length - 2 ? 'success' : 'process'"
            >
            </el-step>
          </el-steps>
        </div>
      </div>

      <!-- 真实案例展示 -->
      <div class="demo-showcase animate-fade-in-up animate-delay-600" ref="demoSection">
        <div class="demo-container">
          <div class="demo-header">
            <h3 class="demo-title">真实案例，瞬间生成</h3>
            <p class="demo-subtitle">看看普通人如何用一句话创造专业应用</p>
          </div>
          
          <!-- 真实案例轮播展示 -->
          <div 
            class="demo-carousel"
            @mouseenter="stopDemoAutoPlay"
            @mouseleave="startDemoAutoPlay"
          >
            <div class="demo-carousel-track" :style="{ transform: `translateX(-${currentDemoSlide * 100}%)` }">
              <div 
                v-for="(demoGroup, index) in demoSlides" 
                :key="index"
                class="demo-carousel-slide"
              >
                <div class="demo-examples">
                  <div 
                    v-for="(demo, idx) in demoGroup" 
                    :key="idx"
                    class="demo-item hover-lift"
                  >
                    <div class="demo-speaker">
                      <div class="speaker-avatar">{{ demo.avatar }}</div>
                      <div class="speaker-info">
                        <span class="speaker-name">{{ demo.name }}</span>
                        <span class="speaker-role">{{ demo.role }}</span>
                      </div>
                    </div>
                    <div class="demo-request">
                      "{{ demo.request }}"
                    </div>
                    <div class="demo-result">
                      <div class="result-icon">✨</div>
                      <div class="result-text">{{ demo.result }}</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 轮播控制按钮 -->
            <div class="demo-carousel-controls">
              <button 
                class="carousel-btn prev" 
                @click="prevDemoSlide"
              >
                <el-icon><ArrowLeft /></el-icon>
              </button>
              <button 
                class="carousel-btn next" 
                @click="nextDemoSlide"
              >
                <el-icon><ArrowRight /></el-icon>
              </button>
            </div>
            
            <!-- 轮播指示器 -->
            <div class="demo-carousel-indicators">
              <div class="indicators-container">
                <span 
                  v-for="(_, index) in demoSlides" 
                  :key="index"
                  class="demo-indicator"
                  :class="{ active: currentDemoSlide === index }"
                  @click="goToDemoSlide(index)"
                ></span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 核心特性 -->
      <div class="features-section" ref="featuresSection">
        <!-- 非技术视角 -->
        <template v-if="perspective === 'non-tech'">
          <div class="section-header">
            <h2 class="section-title">三大核心功能</h2>
            <p class="section-subtitle">简单易用，无需编程基础</p>
          </div>
          
          <div class="features-grid">
            <div 
              v-for="(feature, index) in nonTechFeatures" 
              :key="feature.title"
              class="feature-card card hover-lift animate-fade-in-up"
              :class="`animate-delay-${700 + index * 100}`"
            >
              <div class="feature-icon-wrapper">
                <span class="feature-emoji">{{ feature.emoji }}</span>
              </div>
              <h3 class="feature-title">{{ feature.title }}</h3>
              <p class="feature-description">{{ feature.description }}</p>
              <div class="feature-highlights">
                <div 
                  v-for="highlight in feature.highlights" 
                  :key="highlight"
                  class="feature-highlight"
                >
                  <el-icon><Check /></el-icon>
                  <span>{{ highlight }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>
        
        <!-- 技术视角 -->
        <template v-else>
          <div class="section-header">
            <h2 class="section-title">核心特性</h2>
            <p class="section-subtitle">下一代软件构建与运行系统的强大能力</p>
          </div>
          
          <div class="features-grid">
            <div 
              v-for="(feature, index) in techFeatures" 
              :key="feature.title"
              class="feature-card card hover-lift animate-fade-in-up"
              :class="`animate-delay-${700 + index * 100}`"
            >
              <div class="feature-icon-wrapper">
                <el-icon :size="40" :color="feature.color">
                  <component :is="feature.icon" />
                </el-icon>
              </div>
              <h3 class="feature-title">{{ feature.title }}</h3>
              <p class="feature-description">{{ feature.description }}</p>
              <div class="feature-highlights">
                <div 
                  v-for="highlight in feature.highlights" 
                  :key="highlight"
                  class="feature-highlight"
                >
                  <el-icon><Check /></el-icon>
                  <span>{{ highlight }}</span>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- 技术架构（仅技术视角显示） -->
      <div v-if="perspective === 'tech'" class="architecture-section">
        <div class="section-header">
          <h2 class="section-title">技术架构</h2>
          <p class="section-subtitle">AI原生的一站式构建与运行平台，支持私有化部署</p>
        </div>
        
        <div class="architecture-diagram">
          <div class="arch-layer">
            <div class="arch-label">应用层</div>
            <div class="arch-items">
              <div class="arch-item">Web应用</div>
              <div class="arch-item">API服务</div>
              <div class="arch-item">数据看板</div>
            </div>
          </div>
          <div class="arch-arrow">↓</div>
          <div class="arch-layer">
            <div class="arch-label">构建层</div>
            <div class="arch-items">
              <div class="arch-item">AI代码生成</div>
              <div class="arch-item">Docker镜像打包</div>
              <div class="arch-item">智能组件库</div>
            </div>
          </div>
          <div class="arch-arrow">↓</div>
          <div class="arch-layer">
            <div class="arch-label">运行层</div>
            <div class="arch-items">
              <div class="arch-item">容器化运行</div>
              <div class="arch-item">统一登录系统</div>
              <div class="arch-item">自动扩缩容</div>
            </div>
          </div>
          <div class="arch-arrow">↓</div>
          <div class="arch-layer">
            <div class="arch-label">部署层</div>
            <div class="arch-items">
              <div class="arch-item">私有化部署</div>
              <div class="arch-item">云端部署</div>
              <div class="arch-item">边缘计算</div>
            </div>
          </div>
        </div>
        
        <!-- 核心优势说明 -->
        <div class="architecture-advantages">
          <div class="advantage-item">
            <div class="advantage-icon">📦</div>
            <div class="advantage-content">
              <h4 class="advantage-title">Docker镜像化</h4>
              <p class="advantage-desc">生成的软件像Docker镜像一样，一次构建，到处运行，支持私有化部署</p>
            </div>
          </div>
          <div class="advantage-item">
            <div class="advantage-icon">🔐</div>
            <div class="advantage-content">
              <h4 class="advantage-title">统一登录系统</h4>
              <p class="advantage-desc">所有应用共享用户信息，无论fork多少应用都无需新注册用户</p>
            </div>
          </div>
          <div class="advantage-item">
            <div class="advantage-icon">🔓</div>
            <div class="advantage-content">
              <h4 class="advantage-title">完全开源</h4>
              <p class="advantage-desc">系统完全开源，代码透明，可自由定制和二次开发</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 统计数据 -->
      <div class="stats-section">
        <div class="stats-grid">
          <div 
            v-for="(stat, index) in stats" 
            :key="stat.label"
            class="stat-item animate-fade-in-up"
            :class="`animate-delay-${800 + index * 100}`"
          >
            <div class="stat-icon">{{ stat.icon }}</div>
            <div class="stat-value">{{ stat.value }}</div>
            <div class="stat-label">{{ stat.label }}</div>
          </div>
        </div>
      </div>

      <!-- 快速开始 -->
      <div class="quick-start-section">
        <div class="quick-start-card card animate-fade-in-up animate-delay-900">
          <!-- 非技术视角 -->
          <template v-if="perspective === 'non-tech'">
            <div class="section-header">
              <h2 class="section-title">快速开始</h2>
              <p class="section-subtitle">4步完成应用创建，无需编程基础</p>
            </div>
            
            <div class="steps-container">
              <div 
                v-for="(step, index) in nonTechQuickStartSteps" 
                :key="step.title"
                class="step-item"
              >
                <div class="step-number">{{ index + 1 }}</div>
                <div class="step-content">
                  <h4 class="step-title">{{ step.title }}</h4>
                  <p class="step-description">{{ step.description }}</p>
                </div>
              </div>
            </div>
          </template>
          
          <!-- 技术视角 -->
          <template v-else>
            <div class="section-header">
              <h2 class="section-title">快速开始</h2>
              <p class="section-subtitle">4步完成应用构建，开启AI原生开发之旅</p>
            </div>
            
            <div class="steps-container">
              <div 
                v-for="(step, index) in techQuickStartSteps" 
                :key="step.title"
                class="step-item"
              >
                <div class="step-number">{{ index + 1 }}</div>
                <div class="step-content">
                  <h4 class="step-title">{{ step.title }}</h4>
                  <p class="step-description">{{ step.description }}</p>
                </div>
              </div>
            </div>
          </template>
          
          <div class="quick-start-actions">
            <el-button 
              type="primary" 
              size="large" 
              @click="goToWorkspace"
              class="action-btn primary-btn"
            >
              <el-icon><Cpu /></el-icon>
              立即开始
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Cpu, 
  View, 
  Lightning, 
  Setting, 
  DataAnalysis, 
  Monitor,
  Upload,
  Lock,
  ArrowLeft,
  ArrowRight,
  Check
} from '@element-plus/icons-vue'

const router = useRouter()

// 视角切换：非技术用户 vs 技术用户
const perspective = ref<'non-tech' | 'tech'>('non-tech')

const switchPerspective = (newPerspective: 'non-tech' | 'tech') => {
  perspective.value = newPerspective
  // 保存到 localStorage
  localStorage.setItem('home-perspective', newPerspective)
}

// 从 localStorage 恢复视角
onMounted(() => {
  const saved = localStorage.getItem('home-perspective')
  if (saved === 'tech' || saved === 'non-tech') {
    perspective.value = saved
  }
})

// 真实案例数据
const demos = [
  {
    avatar: '🏛️',
    name: '北京市政府',
    role: '市民服务热线',
    request: '我需要一个百姓呼声工单系统，能受理市民提交的各类诉求，自动分类派单，跟踪处理进度，还要能统计分析',
    result: '8分钟后，智能工单系统部署完成，被全国100+城市fork使用，工单处理效率提升50%'
  },
  {
    avatar: '👨‍🔬',
    name: '科研人员',
    role: '华清大学物理系',
    request: '我需要一个计算斐波那契数列的工具，能输入参数，计算前N项，还要能绘制数列增长曲线图',
    result: '3分钟后，专业的数学分析工具生成完成，被全球500+科研院所fork'
  },
  {
    avatar: '👩‍🏫',
    name: '王老师',
    role: '春光小学五年级',
    request: '我需要一个互动式分数学习游戏，要有图形化演示，能随机生成题目，还要有闯关模式激励学生',
    result: '5分钟后，生动的教学游戏制作完成，被fork 5万次，惠及全球数千万儿童'
  },
  {
    avatar: '👩‍⚕️',
    name: '张医生',
    role: '北京协和医院心血管科',
    request: '我需要一个心血管疾病风险评估系统，能根据患者数据计算风险指数，生成个性化健康建议',
    result: '5分钟后，专业医疗诊断辅助系统上线，被全国2000+医院fork使用'
  },
  {
    avatar: '🌾',
    name: '农民马丁',
    role: '肯尼亚东非农场',
    request: '我需要一个农作物病虫害识别工具，拍照就能诊断并给出治疗建议，支持离线使用',
    result: '8分钟后，AI病虫害识别系统部署完成，被fork十万次，惠及整个东非'
  },
  {
    avatar: '💇‍♂️',
    name: '理发师张师傅',
    role: '江南小镇理发店',
    request: '我需要一个收银系统，能记录客户信息，计算不同服务的价格，还能生成小票',
    result: '2分钟后，完整的收银管理系统上线，被fork 100万次，服务全球小商户'
  },
  {
    avatar: '🏦',
    name: '风控经理',
    role: '某大型银行',
    request: '我需要一个智能风控系统，能实时分析交易数据，识别异常行为，自动预警',
    result: '12分钟后，AI风控系统部署完成，被全球500+银行fork，风险识别准确率提升40%'
  },
  {
    avatar: '🚗',
    name: '交通调度员',
    role: '某城市交通局',
    request: '我需要一个智能交通调度系统，能根据实时路况优化信号灯配时，缓解交通拥堵',
    result: '9分钟后，AI交通调度系统上线，被全球100+城市fork，交通效率提升35%'
  }
]

// 将案例分组，每组3个
const demoSlides = computed(() => {
  const itemsPerSlide = 3
  const slides = []
  for (let i = 0; i < demos.length; i += itemsPerSlide) {
    slides.push(demos.slice(i, i + itemsPerSlide))
  }
  return slides
})

// 真实案例轮播状态
const currentDemoSlide = ref(0)
const demoAutoPlayInterval = ref<NodeJS.Timeout | null>(null)

// 真实案例轮播控制方法
const goToDemoSlide = (index: number) => {
  currentDemoSlide.value = index
}

const nextDemoSlide = () => {
  if (currentDemoSlide.value < demoSlides.value.length - 1) {
    currentDemoSlide.value++
  } else {
    currentDemoSlide.value = 0
  }
}

const prevDemoSlide = () => {
  if (currentDemoSlide.value > 0) {
    currentDemoSlide.value--
  } else {
    currentDemoSlide.value = demoSlides.value.length - 1
  }
}

const startDemoAutoPlay = () => {
  demoAutoPlayInterval.value = setInterval(nextDemoSlide, 5000)
}

const stopDemoAutoPlay = () => {
  if (demoAutoPlayInterval.value) {
    clearInterval(demoAutoPlayInterval.value)
    demoAutoPlayInterval.value = null
  }
}

// 闪电克隆流程步骤（非技术视角）
const cloneFlowSteps = [
  {
    title: '北京百姓呼声工单管理系统',
    description: '北京市政府描述开发，上传到克隆中心并设置价格为100元进行售卖'
  },
  {
    title: '南京市政府付费闪电克隆',
    description: '南京市政府支付100元，闪电克隆北京系统，重命名为南京市民工单管理系统，新增了"附件上传"功能。北京市政府收到100元克隆费，轻松赚钱'
  },
  {
    title: '南京上传到克隆中心并设置价格',
    description: '南京将改进后的系统上传到克隆中心，设置价格为150元。克隆费分配：北京（原始开发者）获得原始价格100元，南京（改进者）获得超出部分50元'
  },
  {
    title: '深圳市政府付费克隆',
    description: '深圳市政府支付150元，克隆南京系统，新增了"富文本编辑"功能。北京获得100元，南京获得50元'
  },
  {
    title: '深圳上传到克隆中心并设置价格',
    description: '深圳将改进后的系统上传到克隆中心，设置价格为200元。克隆费分配：北京（原始开发者）获得原始价格100元，南京（中间改进者）获得50元（150-100），深圳（当前改进者）获得增值部分50元（200-150）'
  },
  {
    title: '上海市政府付费克隆',
    description: '上海市政府支付200元，克隆深圳系统，立即拥有附件上传+富文本编辑。北京获得100元，南京获得50元，深圳获得50元。每个改进者都获得自己贡献的增值部分'
  },
  {
    title: '北京也可以克隆变体版本',
    description: '北京也可以付费克隆变体后的版本，立即拥有别人帮忙升级的所有功能。这样原始开发者也能从改进中受益'
  },
  {
    title: '形成应用生态和商业闭环',
    description: '大家互相升级应用，通过智能收益分配机制，原始开发者、改进者和当前所有者都能获得合理收益，形成强大的应用生态和商业闭环'
  }
]

// 非技术视角特性
const nonTechFeatures = [
  {
    emoji: '💬',
    title: '描述生成应用',
    description: '不需要懂编程，只需要用自然语言描述你的需求，AI就能帮你生成专业应用',
    highlights: ['用自然语言描述', 'AI自动理解', '秒级生成应用', '无需编程基础']
  },
  {
    emoji: '⚡',
    title: '闪电克隆',
    description: '看到别人好的应用？一键克隆，立即拥有，无需重新描述和开发',
    highlights: ['一键复制应用', '无需重新开发', '立即可以使用', '保留所有功能']
  },
  {
    emoji: '✏️',
    title: '自由修改',
    description: '克隆后的应用可以自由修改，根据自己的需求调整功能',
    highlights: ['自由编辑', '个性化定制', '修改功能', '调整界面']
  },
  {
    emoji: '🔄',
    title: '生成新变体',
    description: '修改后的应用可以生成新的变体，创造出属于你自己的应用，新变体可以继续被别人克隆',
    highlights: ['生成新版本', '创造变体', '独立运行', '分享给他人']
  },
  {
    emoji: '🌐',
    title: '统一登录',
    description: '所有应用使用同一个账号登录，无需重复注册，方便快捷',
    highlights: ['一个账号', '所有应用', '无需重复注册', '方便管理']
  },
  {
    emoji: '🚀',
    title: '立即使用',
    description: '生成或克隆的应用立即可以使用，无需等待，马上开始工作',
    highlights: ['即时可用', '无需等待', '马上开始', '随时使用']
  },
  {
    emoji: '💰',
    title: '克隆中心售卖',
    description: '上传应用到官方克隆中心，设置价格进行售卖。别人无需描述，直接付费克隆即可使用，轻松赚钱',
    highlights: ['上传到克隆中心', '设置价格售卖', '别人付费克隆', '轻松赚钱']
  }
]

// 技术视角特性
const techFeatures = [
  {
    icon: Lightning,
    title: 'AI原生构建',
    description: '基于大语言模型，用自然语言描述需求，AI自动生成完整应用代码',
    color: '#f59e0b',
    highlights: ['自然语言理解', '智能代码生成', '自动架构设计']
  },
  {
    icon: Setting,
    title: '一站式平台',
    description: '从需求描述到应用部署，全流程自动化，构建与运行一体化',
    color: '#6366f1',
    highlights: ['需求分析', '代码生成', '自动部署', '运行管理']
  },
  {
    icon: Upload,
    title: 'Docker镜像化',
    description: '生成的软件像Docker镜像一样，一次构建，到处运行，支持私有化部署',
    color: '#10b981',
    highlights: ['容器化打包', '镜像分发', '私有化部署', '环境隔离']
  },
  {
    icon: Lock,
    title: '统一登录系统',
    description: '所有应用共享用户信息，无论fork多少应用都无需新注册用户',
    color: '#8b5cf6',
    highlights: ['单点登录', '用户共享', '权限统一', '无需重复注册']
  },
  {
    icon: Monitor,
    title: '自动运行系统',
    description: '容器化部署，自动扩缩容，监控告警，保障应用稳定运行',
    color: '#06b6d4',
    highlights: ['容器化运行', '自动扩缩容', '监控告警', '日志管理']
  },
  {
    icon: DataAnalysis,
    title: '完全开源',
    description: '系统完全开源，代码透明，可自由定制和二次开发',
    color: '#ef4444',
    highlights: ['GitHub开源', '代码透明', '自由定制', '社区驱动']
  }
]

// 统计数据
const stats = [
  { icon: '👥', value: '10万+', label: '活跃开发者' },
  { icon: '🚀', value: '100万+', label: '已生成应用' },
  { icon: '⚡', value: '秒级', label: '生成速度' },
  { icon: '🌍', value: '全球', label: '服务覆盖' }
]

// 非技术视角快速开始步骤
const nonTechQuickStartSteps = [
  {
    title: '描述你的需求',
    description: '用简单的话描述你想要的应用，比如"我需要一个记账本"'
  },
  {
    title: 'AI自动生成',
    description: 'AI理解你的需求，自动生成完整的应用，几秒钟就能完成'
  },
  {
    title: '或者克隆应用',
    description: '看到别人做的好应用？点击克隆，立即拥有，无需重新描述'
  },
  {
    title: '修改和变体',
    description: '克隆的应用可以自由修改，生成属于你自己的新版本'
  }
]

// 技术视角快速开始步骤
const techQuickStartSteps = [
  {
    title: '描述需求',
    description: '用自然语言描述你想要的应用功能，AI将理解并分析你的需求'
  },
  {
    title: 'AI生成Docker镜像',
    description: '基于大语言模型，AI自动生成完整的应用代码，并打包成Docker镜像'
  },
  {
    title: '预览调整',
    description: '实时预览生成的应用，支持在线调整和优化功能'
  },
  {
    title: '一键部署',
    description: '确认无误后，一键部署到云端或私有化环境，统一登录，无需重复注册'
  }
]

// 引用
const demoSection = ref<HTMLElement>()
const featuresSection = ref<HTMLElement>()

const goToWorkspace = () => {
  router.push('/workspace')
}

const scrollToFeatures = () => {
  featuresSection.value?.scrollIntoView({ behavior: 'smooth' })
}

const openGitHub = () => {
  window.open('https://github.com/ai-agent-os/ai-agent-os', '_blank')
}

// 生命周期管理
onMounted(() => {
  startDemoAutoPlay()
  // 允许页面滚动 - 强制覆盖全局样式
  // 使用 nextTick 确保 DOM 已渲染
  nextTick(() => {
    // 强制覆盖全局样式，允许滚动
    document.body.style.setProperty('overflow', 'auto', 'important')
    document.body.style.setProperty('height', 'auto', 'important')
    document.body.style.setProperty('position', 'relative', 'important')
    
    document.documentElement.style.setProperty('overflow', 'auto', 'important')
    document.documentElement.style.setProperty('height', 'auto', 'important')
    
    const app = document.getElementById('app')
    if (app) {
      app.style.setProperty('overflow', 'visible', 'important')
      app.style.setProperty('height', 'auto', 'important')
      app.style.setProperty('min-height', '100vh', 'important')
    }
  })
})

onUnmounted(() => {
  stopDemoAutoPlay()
  // 恢复默认样式
  document.body.style.overflow = ''
  document.body.style.height = ''
  document.body.style.position = ''
  document.documentElement.style.overflow = ''
  document.documentElement.style.height = ''
  
  const app = document.getElementById('app')
  if (app) {
    app.style.overflow = ''
    app.style.height = ''
    app.style.minHeight = ''
  }
})
</script>

<style lang="scss" scoped>
.home-container {
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
  overflow-y: visible;
  background: linear-gradient(
    135deg,
    #0f172a 0%,
    #1e293b 50%,
    #0f172a 100%
  );
  color: #e2e8f0;
  width: 100%;
  height: auto;
  display: block;
}

// 视角切换器
.perspective-switcher {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 1000;
  
  .switcher-container {
    display: flex;
    gap: 8px;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    padding: 4px;
    backdrop-filter: blur(10px);
    
    .switcher-option {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 16px;
      border-radius: 8px;
      cursor: pointer;
      transition: all 0.3s ease;
      color: #94a3b8;
      font-size: 14px;
      font-weight: 500;
      
      &:hover {
        background: rgba(255, 255, 255, 0.1);
        color: #e2e8f0;
      }
      
      &.active {
        background: rgba(99, 102, 241, 0.3);
        color: #e2e8f0;
        border: 1px solid rgba(99, 102, 241, 0.5);
      }
      
      .switcher-icon {
        font-size: 18px;
      }
      
      .switcher-label {
        white-space: nowrap;
      }
    }
  }
}

.background-decoration {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
  overflow: hidden;
}

.decoration-circle {
  position: absolute;
  border-radius: 50%;
  background: linear-gradient(45deg, #6366f1, #8b5cf6, #ec4899);
  opacity: 0.1;
  filter: blur(60px);
  
  &.circle-1 {
    width: 500px;
    height: 500px;
    top: -250px;
    right: -250px;
    animation: float 20s ease-in-out infinite;
  }
  
  &.circle-2 {
    width: 400px;
    height: 400px;
    bottom: -200px;
    left: -200px;
    animation: float 25s ease-in-out infinite;
  }
  
  &.circle-3 {
    width: 300px;
    height: 300px;
    top: 50%;
    left: 10%;
    transform: translateY(-50%);
    animation: float 30s ease-in-out infinite;
  }
}

.decoration-grid {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image: 
    linear-gradient(rgba(99, 102, 241, 0.1) 1px, transparent 1px),
    linear-gradient(90deg, rgba(99, 102, 241, 0.1) 1px, transparent 1px);
  background-size: 50px 50px;
  opacity: 0.3;
}

.main-content {
  position: relative;
  z-index: 1;
  max-width: 1400px;
  margin: 0 auto;
  padding: 80px 24px;
}

.hero-section {
  text-align: center;
  margin-bottom: 120px;
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(99, 102, 241, 0.1);
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 24px;
  margin-bottom: 24px;
  backdrop-filter: blur(10px);
  
  .badge-icon {
    font-size: 20px;
  }
  
  .badge-text {
    font-size: 14px;
    color: #a5b4fc;
    font-weight: 500;
  }
}

.hero-title {
  font-size: clamp(2.5rem, 6vw, 5rem);
  font-weight: 800;
  margin: 0 0 24px 0;
  line-height: 1.1;
  
  .title-line {
    display: block;
    
    &.title-gradient {
      background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 50%, #ec4899 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
  }
}

.hero-subtitle {
  font-size: clamp(1.25rem, 2vw, 1.75rem);
  margin: 0 0 16px 0;
  
  .highlight-text {
    background: linear-gradient(135deg, #a5b4fc 0%, #c084fc 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
    font-weight: 700;
  }
}

.hero-description {
  font-size: clamp(1rem, 1.5vw, 1.25rem);
  color: #94a3b8;
  margin: 0 0 32px 0;
  line-height: 1.8;
  max-width: 800px;
  margin-left: auto;
  margin-right: auto;
}

.github-badge {
  margin-bottom: 48px;
  
  .github-btn {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: #e2e8f0;
    backdrop-filter: blur(10px);
    transition: all 0.3s ease;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 12px 24px;
    font-size: 14px;
    font-weight: 500;
    
    &:hover {
      background: rgba(255, 255, 255, 0.15);
      transform: translateY(-2px);
      box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
      border-color: rgba(255, 255, 255, 0.3);
    }
    
    .github-icon {
      font-size: 18px;
    }
  }
}

.action-buttons {
  display: flex;
  gap: 16px;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 64px;
}

.action-btn {
  padding: 16px 32px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  transition: all 0.3s ease;
  
  &.primary-btn {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    border: none;
    color: white;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 12px 24px rgba(99, 102, 241, 0.4);
    }
  }
  
  &.secondary-btn {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: #e2e8f0;
    backdrop-filter: blur(10px);
    
    &:hover {
      background: rgba(255, 255, 255, 0.15);
      transform: translateY(-2px);
    }
  }
}

.core-capabilities {
  display: flex;
  justify-content: center;
  gap: 32px;
  flex-wrap: wrap;
  margin-top: 48px;
}

.capability-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 24px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
  
  &:hover {
    background: rgba(255, 255, 255, 0.08);
    transform: translateY(-2px);
  }
  
  .capability-icon {
    font-size: 32px;
  }
  
  .capability-text {
    .capability-label {
      font-size: 16px;
      font-weight: 600;
      color: #e2e8f0;
      margin-bottom: 4px;
    }
    
    .capability-desc {
      font-size: 12px;
      color: #94a3b8;
    }
  }
}

// 案例展示样式
.demo-showcase {
  margin-bottom: 120px;
  
  .demo-container {
    max-width: 1200px;
    margin: 0 auto;
  }
  
  .demo-header {
    text-align: center;
    margin-bottom: 48px;
    
    .demo-title {
      font-size: 2.5rem;
      font-weight: 700;
      margin-bottom: 12px;
      background: linear-gradient(135deg, #6366f1, #8b5cf6);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
    
    .demo-subtitle {
      font-size: 1.25rem;
      color: #94a3b8;
    }
  }
  
  .demo-carousel {
    position: relative;
    overflow: hidden;
    border-radius: 24px;
    background: rgba(255, 255, 255, 0.05);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 32px;
    
    .demo-carousel-track {
      display: flex;
      transition: transform 0.5s cubic-bezier(0.4, 0, 0.2, 1);
    }
    
    .demo-carousel-slide {
      min-width: 100%;
      flex-shrink: 0;
    }
    
    .demo-examples {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 24px;
    }
    
    .demo-item {
      background: rgba(255, 255, 255, 0.05);
      border-radius: 16px;
      padding: 24px;
      border: 1px solid rgba(255, 255, 255, 0.1);
      transition: all 0.3s ease;
      
      &:hover {
        background: rgba(255, 255, 255, 0.08);
        transform: translateY(-4px);
        box-shadow: 0 12px 24px rgba(0, 0, 0, 0.2);
      }
      
      .demo-speaker {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
        
        .speaker-avatar {
          font-size: 2.5rem;
          width: 64px;
          height: 64px;
          border-radius: 50%;
          background: linear-gradient(135deg, #6366f1, #8b5cf6);
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;
        }
        
        .speaker-info {
          .speaker-name {
            display: block;
            font-size: 1.125rem;
            font-weight: 600;
            color: #e2e8f0;
            margin-bottom: 4px;
          }
          
          .speaker-role {
            font-size: 0.875rem;
            color: #94a3b8;
          }
        }
      }
      
      .demo-request {
        background: rgba(99, 102, 241, 0.1);
        border-radius: 12px;
        padding: 16px;
        margin-bottom: 16px;
        font-style: italic;
        color: #cbd5e1;
        border-left: 4px solid #6366f1;
        line-height: 1.6;
      }
      
      .demo-result {
        display: flex;
        align-items: center;
        gap: 8px;
        color: #10b981;
        font-weight: 500;
        
        .result-icon {
          font-size: 1.25rem;
          animation: sparkle 2s infinite;
        }
      }
    }
    
    .demo-carousel-controls {
      position: absolute;
      top: 50%;
      transform: translateY(-50%);
      width: 100%;
      display: flex;
      justify-content: space-between;
      pointer-events: none;
      padding: 0 16px;
      
      .carousel-btn {
        background: rgba(255, 255, 255, 0.1);
        border: 1px solid rgba(255, 255, 255, 0.2);
        border-radius: 50%;
        width: 48px;
        height: 48px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #e2e8f0;
        cursor: pointer;
        transition: all 0.3s ease;
        pointer-events: auto;
        backdrop-filter: blur(10px);
        
        &:hover {
          background: rgba(255, 255, 255, 0.2);
          transform: scale(1.1);
        }
      }
    }
    
    .demo-carousel-indicators {
      margin-top: 32px;
      display: flex;
      justify-content: center;
      gap: 8px;
      
      .demo-indicator {
        width: 12px;
        height: 12px;
        border-radius: 50%;
        background: rgba(255, 255, 255, 0.3);
        cursor: pointer;
        transition: all 0.3s ease;
        
        &:hover {
          background: rgba(255, 255, 255, 0.5);
          transform: scale(1.2);
        }
        
        &.active {
          background: #6366f1;
          transform: scale(1.3);
        }
      }
    }
  }
}

// 特性区域
.features-section {
  margin-bottom: 120px;
  
  .section-header {
    text-align: center;
    margin-bottom: 64px;
    
    .section-title {
      font-size: 2.5rem;
      font-weight: 700;
      margin-bottom: 16px;
      color: #e2e8f0;
    }
    
    .section-subtitle {
      font-size: 1.25rem;
      color: #94a3b8;
    }
  }
  
  .features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
    gap: 32px;
  }
  
  .feature-card {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 24px;
    padding: 32px;
    transition: all 0.3s ease;
    backdrop-filter: blur(10px);
    
    &:hover {
      background: rgba(255, 255, 255, 0.08);
      transform: translateY(-8px);
      box-shadow: 0 16px 32px rgba(0, 0, 0, 0.3);
    }
    
    .feature-icon-wrapper {
      width: 80px;
      height: 80px;
      border-radius: 20px;
      background: rgba(99, 102, 241, 0.1);
      display: flex;
      align-items: center;
      justify-content: center;
      margin-bottom: 24px;
      transition: all 0.3s ease;
      
      .el-icon {
        transition: all 0.3s ease;
      }
      
      .feature-emoji {
        font-size: 40px;
        line-height: 1;
      }
    }
    
    .feature-title {
      font-size: 1.5rem;
      font-weight: 700;
      color: #e2e8f0;
      margin-bottom: 12px;
    }
    
    .feature-description {
      color: #94a3b8;
      line-height: 1.6;
      margin-bottom: 24px;
    }
    
    .feature-highlights {
      display: flex;
      flex-direction: column;
      gap: 12px;
      
      .feature-highlight {
        display: flex;
        align-items: center;
        gap: 8px;
        color: #cbd5e1;
        font-size: 0.875rem;
        
        .el-icon {
          color: #10b981;
        }
      }
    }
    
    .feature-example {
      margin-top: 24px;
      padding: 20px;
      background: rgba(99, 102, 241, 0.1);
      border: 1px solid rgba(99, 102, 241, 0.2);
      border-radius: 12px;
      border-left: 4px solid #6366f1;
      
      .example-title {
        font-size: 0.875rem;
        font-weight: 600;
        color: #6366f1;
        margin-bottom: 16px;
        text-align: center;
      }
      
      .example-steps {
        display: flex;
        flex-direction: column;
        gap: 12px;
        
        .example-step {
          padding: 10px 16px;
          background: rgba(255, 255, 255, 0.05);
          border-radius: 8px;
          transition: all 0.3s ease;
          
          &.step-highlight {
            background: rgba(99, 102, 241, 0.2);
            border: 1px solid rgba(99, 102, 241, 0.3);
            font-weight: 600;
            color: #e2e8f0;
          }
          
          .step-text {
            font-size: 0.875rem;
            color: #cbd5e1;
            line-height: 1.5;
            display: block;
          }
          
          &:hover {
            background: rgba(255, 255, 255, 0.08);
            transform: translateX(4px);
          }
        }
      }
    }
  }
}

// 闪电克隆流程展示
.clone-flow-section {
  margin-bottom: 80px;
  
  .section-header {
    text-align: center;
    margin-bottom: 32px;
    
    .section-title {
      font-size: 1.75rem;
      font-weight: 700;
      margin-bottom: 12px;
      background: linear-gradient(135deg, #f59e0b, #ef4444);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }
    
    .section-subtitle {
      font-size: 1rem;
      color: #94a3b8;
    }
  }
  
  .clone-flow-container {
    max-width: 800px;
    margin: 0 auto;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 16px;
    padding: 32px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(20px);
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.2);
  }
  
  .clone-flow-steps {
    :deep(.el-step__title) {
      font-size: 1rem;
      font-weight: 600;
      color: #e2e8f0;
      line-height: 1.4;
    }
    
    :deep(.el-step__description) {
      font-size: 0.8125rem;
      color: #94a3b8;
      line-height: 1.5;
      margin-top: 8px;
    }
    
    :deep(.el-step__head) {
      .el-step__icon {
        width: 32px;
        height: 32px;
        border-color: rgba(99, 102, 241, 0.5);
        background: rgba(99, 102, 241, 0.1);
        color: #6366f1;
      }
      
      &.is-success .el-step__icon {
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        border-color: #6366f1;
        color: white;
      }
      
      &.is-process .el-step__icon {
        background: rgba(99, 102, 241, 0.2);
        border-color: #6366f1;
        color: #6366f1;
      }
    }
    
    :deep(.el-step__line) {
      background: linear-gradient(180deg, rgba(99, 102, 241, 0.5), rgba(99, 102, 241, 0.2));
    }
    
    :deep(.el-step__line-inner) {
      background: linear-gradient(180deg, #6366f1, #8b5cf6);
    }
  }
  
  .clone-flow-steps {
    :deep(.el-step) {
      margin-bottom: 24px;
    }
    
    :deep(.el-step__main) {
      padding-left: 16px;
    }
    
    :deep(.el-step__title) {
      font-size: 1rem;
      font-weight: 600;
      color: #e2e8f0;
      line-height: 1.4;
      margin-bottom: 0;
      display: block;
    }
    
    :deep(.el-step__description) {
      font-size: 0.8125rem;
      color: #94a3b8;
      line-height: 1.5;
      margin-top: 6px;
      margin-bottom: 0;
      padding-right: 0;
      display: block;
    }
    
    :deep(.el-step__body) {
      margin-top: 0;
      margin-bottom: 0;
    }
    
    :deep(.el-step__line) {
      top: 40px;
    }
    
    :deep(.el-step__head) {
      .el-step__icon {
        width: 32px;
        height: 32px;
        border-color: rgba(99, 102, 241, 0.5);
        background: rgba(99, 102, 241, 0.1);
        color: #6366f1;
      }
      
      &.is-success .el-step__icon {
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        border-color: #6366f1;
        color: white;
      }
      
      &.is-process .el-step__icon {
        background: rgba(99, 102, 241, 0.2);
        border-color: #6366f1;
        color: #6366f1;
      }
    }
    
    :deep(.el-step__line) {
      background: linear-gradient(180deg, rgba(99, 102, 241, 0.5), rgba(99, 102, 241, 0.2));
    }
    
    :deep(.el-step__line-inner) {
      background: linear-gradient(180deg, #6366f1, #8b5cf6);
    }
  }
}

@keyframes bounce {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-8px);
  }
}

// 架构图
.architecture-section {
  margin-bottom: 120px;
  
  .section-header {
    text-align: center;
    margin-bottom: 64px;
    
    .section-title {
      font-size: 2.5rem;
      font-weight: 700;
      margin-bottom: 16px;
      color: #e2e8f0;
    }
    
    .section-subtitle {
      font-size: 1.25rem;
      color: #94a3b8;
    }
  }
  
  .architecture-diagram {
    max-width: 800px;
    margin: 0 auto;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 24px;
    padding: 48px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    backdrop-filter: blur(10px);
    
    .arch-layer {
      text-align: center;
      margin-bottom: 24px;
      
      .arch-label {
        font-size: 1.25rem;
        font-weight: 600;
        color: #6366f1;
        margin-bottom: 16px;
      }
      
      .arch-items {
        display: flex;
        justify-content: center;
        gap: 16px;
        flex-wrap: wrap;
        
        .arch-item {
          padding: 12px 24px;
          background: rgba(99, 102, 241, 0.1);
          border: 1px solid rgba(99, 102, 241, 0.3);
          border-radius: 12px;
          color: #cbd5e1;
          font-size: 0.875rem;
        }
      }
    }
    
    .arch-arrow {
      text-align: center;
      font-size: 2rem;
      color: #6366f1;
      margin: 16px 0;
    }
  }
  
  .architecture-advantages {
    margin-top: 64px;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 32px;
    
    .advantage-item {
      display: flex;
      align-items: flex-start;
      gap: 20px;
      padding: 32px;
      background: rgba(255, 255, 255, 0.05);
      border: 1px solid rgba(255, 255, 255, 0.1);
      border-radius: 20px;
      backdrop-filter: blur(10px);
      transition: all 0.3s ease;
      
      &:hover {
        background: rgba(255, 255, 255, 0.08);
        transform: translateY(-4px);
        box-shadow: 0 12px 24px rgba(0, 0, 0, 0.2);
      }
      
      .advantage-icon {
        font-size: 3rem;
        flex-shrink: 0;
      }
      
      .advantage-content {
        flex: 1;
        
        .advantage-title {
          font-size: 1.25rem;
          font-weight: 600;
          color: #e2e8f0;
          margin-bottom: 12px;
        }
        
        .advantage-desc {
          font-size: 1rem;
          color: #94a3b8;
          line-height: 1.6;
          margin: 0;
        }
      }
    }
  }
}

// 统计数据
.stats-section {
  margin-bottom: 120px;
  
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 32px;
  }
  
  .stat-item {
    text-align: center;
    padding: 32px;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 20px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    transition: all 0.3s ease;
    backdrop-filter: blur(10px);
    
    &:hover {
      transform: translateY(-4px);
      background: rgba(255, 255, 255, 0.08);
    }
    
    .stat-icon {
      font-size: 3rem;
      margin-bottom: 16px;
    }
    
    .stat-value {
      font-size: 2.5rem;
      font-weight: 700;
      color: #6366f1;
      margin-bottom: 8px;
    }
    
    .stat-label {
      color: #94a3b8;
      font-size: 1rem;
    }
  }
}

// 快速开始
.quick-start-section {
  margin-bottom: 80px;
  
  .quick-start-card {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 24px;
    padding: 64px;
    backdrop-filter: blur(10px);
    
    .section-header {
      text-align: center;
      margin-bottom: 48px;
    }
    
    .steps-container {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 32px;
      margin-bottom: 48px;
    }
    
    .step-item {
      display: flex;
      align-items: flex-start;
      gap: 16px;
      
      .step-number {
        flex-shrink: 0;
        width: 40px;
        height: 40px;
        background: linear-gradient(135deg, #6366f1, #8b5cf6);
        color: white;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 1.125rem;
      }
      
      .step-content {
        .step-title {
          font-size: 1.25rem;
          font-weight: 600;
          color: #e2e8f0;
          margin-bottom: 8px;
        }
        
        .step-description {
          color: #94a3b8;
          line-height: 1.6;
        }
      }
    }
    
    .quick-start-actions {
      text-align: center;
    }
  }
}

// 动画
@keyframes float {
  0%, 100% { transform: translate(0, 0) rotate(0deg); }
  33% { transform: translate(30px, -30px) rotate(120deg); }
  66% { transform: translate(-20px, 20px) rotate(240deg); }
}

@keyframes sparkle {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.2); opacity: 0.8; }
}

.animate-fade-in {
  animation: fadeIn 0.6s ease-out;
}

.animate-fade-in-up {
  animation: fadeInUp 0.8s ease-out;
}

.animate-delay-200 { animation-delay: 0.2s; }
.animate-delay-300 { animation-delay: 0.3s; }
.animate-delay-400 { animation-delay: 0.4s; }
.animate-delay-500 { animation-delay: 0.5s; }
.animate-delay-600 { animation-delay: 0.6s; }
.animate-delay-700 { animation-delay: 0.7s; }
.animate-delay-800 { animation-delay: 0.8s; }
.animate-delay-900 { animation-delay: 0.9s; }

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.hover-lift {
  transition: transform 0.3s ease;
  
  &:hover {
    transform: translateY(-4px);
  }
}

.card {
  transition: all 0.3s ease;
}

// 响应式设计
@media (max-width: 768px) {
  .main-content {
    padding: 40px 16px;
  }
  
  .hero-section {
    margin-bottom: 80px;
  }
  
  .core-capabilities {
    flex-direction: column;
    align-items: stretch;
  }
  
  .demo-examples {
    grid-template-columns: 1fr !important;
  }
  
  .features-grid {
    grid-template-columns: 1fr !important;
  }
  
  .steps-container {
    grid-template-columns: 1fr !important;
  }
  
  .quick-start-card {
    padding: 32px !important;
  }
}
</style>

