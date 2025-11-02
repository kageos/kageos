<template>
  <div class="form-renderer-test">
    <el-page-header @back="goBack" content="表单渲染器测试">
      <template #extra>
        <el-button type="primary" @click="switchTestData">切换测试数据</el-button>
      </template>
    </el-page-header>

    <el-divider />

    <FormRenderer
      v-if="currentTestData"
      :key="currentTestIndex"
      :function-detail="currentTestData"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElPageHeader, ElDivider, ElButton, ElMessage } from 'element-plus'
import FormRenderer from '@/core/renderers/FormRenderer.vue'
import type { FunctionDetail } from '@/core/types/field'

const router = useRouter()

// 测试数据集
const testDataList = ref<FunctionDetail[]>([
  // 测试1：简单表单
  {
    code: 'simple_form',
    name: '简单表单测试',
    description: '测试基础输入框',
    method: 'POST',
    router: '/test/simple',
    template_type: 'form',
    request: [
      {
        code: 'username',
        name: '用户名',
        validation: 'required,min=3,max=20',
        widget: { type: 'input' }
      },
      {
        code: 'email',
        name: '邮箱',
        validation: 'required,email',
        widget: { type: 'input' }
      },
      {
        code: 'password',
        name: '密码',
        validation: 'required,min=6',
        widget: { type: 'input' }
      },
      {
        code: 'description',
        name: '描述',
        widget: { type: 'text' }
      }
    ],
    response: []
  },
  
  // 测试2：工单表单（参考之前的例子）
  {
    code: 'ticket_form',
    name: '工单表单测试',
    description: '测试更复杂的表单',
    method: 'POST',
    router: '/test/ticket',
    template_type: 'form',
    request: [
      {
        code: 'title',
        name: '工单标题',
        validation: 'required,min=5,max=200',
        widget: { type: 'input' }
      },
      {
        code: 'description',
        name: '问题描述',
        validation: 'required,min=10',
        widget: { type: 'text' }
      },
      {
        code: 'contact',
        name: '联系电话',
        validation: 'required,min=11,max=20',
        widget: { type: 'input' }
      },
      {
        code: 'remark',
        name: '备注',
        widget: { type: 'text' }
      }
    ],
    response: []
  },

  // 🔥 测试3：List 内 Select（收银台场景）- 模拟正确的后端响应
  {
    code: 'cashier_desk',
    name: '收银台场景 - List 内 Select',
    description: '测试 List 内 Select 的复杂场景',
    method: 'POST',
    router: '/test/cashier',
    template_type: 'form',
    request: [
      {
        code: 'customer_name',
        name: '客户姓名',
        type: 'string',
        validation: 'required',
        widget: { type: 'input', config: {} }
      },
      {
        code: 'product_quantities',
        name: '商品清单',
        type: '[]struct',
        data: { type: '[]struct' },
        validation: 'required,min=1',
        // 🔥 注意：后端应该返回 "children"，不是 "properties"
        children: [
          {
            code: 'product_id',
            name: '商品',
            type: 'int',
            data: { type: 'int' },
            validation: 'required',
            callbacks: ['OnSelectFuzzy'],  // 🔥 子字段的 callbacks
            widget: {
              type: 'select',
              config: {
                placeholder: '请选择商品',
                creatable: false
              }
            }
          },
          {
            code: 'quantity',
            name: '数量',
            type: 'int',
            data: { type: 'int' },
            validation: 'required,min=1',
            widget: {
              type: 'input',
              config: {
                placeholder: '请输入数量'
              }
            }
          }
        ],
        widget: {
          type: 'table',  // 🔥 后端返回的是 "table"，前端映射为 "list"
          config: null
        }
      },
      {
        code: 'member_id',
        name: '会员卡',
        type: 'int',
        data: { type: 'int' },
        validation: 'required',
        callbacks: ['OnSelectFuzzy'],
        widget: {
          type: 'select',
          config: {
            placeholder: '请选择会员',
            creatable: false
          }
        }
      },
      {
        code: 'remarks',
        name: '备注',
        type: 'string',
        data: { type: 'string' },
        widget: {
          type: 'text_area',
          config: {
            placeholder: '请输入备注'
          }
        }
      }
    ],
    response: []
  },
  
  // 测试4：Struct 结构体
  {
    code: 'order_form',
    name: '订单表单 - Struct 测试',
    description: '测试 Struct 结构体的渲染',
    method: 'POST',
    router: '/test/order',
    template_type: 'form',
    request: [
      {
        code: 'order_no',
        name: '订单号',
        data: { type: 'string' },
        validation: 'required',
        widget: {
          type: 'input',
          config: {
            placeholder: '系统自动生成',
            disabled: true
          }
        }
      },
      {
        code: 'detail',
        name: '订单详情',
        data: { type: 'struct' },
        validation: 'required',
        widget: {
          type: 'form',
          config: null
        },
        children: [
          {
            code: 'address',
            name: '收货地址',
            data: { type: 'string' },
            validation: 'required',
            widget: {
              type: 'text_area',
              config: {
                placeholder: '请输入收货地址'
              }
            }
          },
          {
            code: 'phone',
            name: '联系电话',
            data: { type: 'string' },
            validation: 'required,min=11,max=20',
            widget: {
              type: 'input',
              config: {
                placeholder: '请输入联系电话'
              }
            }
          },
          {
            code: 'note',
            name: '备注',
            data: { type: 'string' },
            validation: '',
            widget: {
              type: 'text_area',
              config: {
                placeholder: '请输入备注信息'
              }
            }
          }
        ]
      },
      {
        code: 'payment_method',
        name: '支付方式',
        data: { type: 'string' },
        validation: 'required,oneof=现金,支付宝,微信',
        widget: {
          type: 'select',
          config: {
            options: ['现金', '支付宝', '微信'],
            placeholder: '请选择支付方式'
          }
        }
      }
    ],
    response: []
  }
])

// 当前测试索引
const currentTestIndex = ref(0)

// 当前测试数据
const currentTestData = computed(() => testDataList.value[currentTestIndex.value])

/**
 * 切换测试数据
 */
function switchTestData(): void {
  currentTestIndex.value = (currentTestIndex.value + 1) % testDataList.value.length
  ElMessage.success(`切换到测试数据 ${currentTestIndex.value + 1}`)
}

/**
 * 返回
 */
function goBack(): void {
  router.back()
}
</script>

<style scoped>
.form-renderer-test {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}
</style>

