<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 表单数据
const formData = reactive({
  name: '',
  email: '',
  age: '',
  gender: '',
  description: ''
})

// 表格数据
const tableData = ref([
  {
    id: 1,
    name: '张三',
    email: 'zhangsan@example.com',
    age: 25,
    gender: '男',
    createTime: '2023-01-15'
  },
  {
    id: 2,
    name: '李四',
    email: 'lisi@example.com',
    age: 30,
    gender: '女',
    createTime: '2023-02-20'
  },
  {
    id: 3,
    name: '王五',
    email: 'wangwu@example.com',
    age: 28,
    gender: '男',
    createTime: '2023-03-10'
  }
])

// 对话框控制
const dialogVisible = ref(false)
const dialogTitle = ref('新增用户')

// 分页配置
const pagination = reactive({
  currentPage: 1,
  pageSize: 10,
  total: 100
})

// 提交表单
const submitForm = () => {
  ElMessage.success('表单提交成功！')
  console.log('表单数据:', formData)
}

// 重置表单
const resetForm = () => {
  Object.assign(formData, {
    name: '',
    email: '',
    age: '',
    gender: '',
    description: ''
  })
  ElMessage.info('表单已重置')
}

// 显示对话框
const showDialog = (title: string) => {
  dialogTitle.value = title
  dialogVisible.value = true
}

// 确认对话框
const confirmDialog = () => {
  ElMessageBox.confirm('确定要删除这条记录吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      ElMessage.success('删除成功')
    })
    .catch(() => {
      ElMessage.info('已取消删除')
    })
}

// 分页变化
const handlePageChange = (page: number) => {
  pagination.currentPage = page
  ElMessage.info(`切换到第 ${page} 页`)
}

// 显示通知
const showNotification = () => {
  ElNotification({
    title: '通知',
    message: '这是一条通知消息',
    type: 'success',
    duration: 3000
  })
}
</script>

<template>
  <div class="demo-container">
    <el-row :gutter="20">
      <!-- 基础组件展示 -->
      <el-col :span="24">
        <el-card class="demo-card">
          <template #header>
            <div class="card-header">
              <span>基础组件演示</span>
            </div>
          </template>

          <el-space wrap>
            <el-button type="primary">主要按钮</el-button>
            <el-button type="success">成功按钮</el-button>
            <el-button type="warning">警告按钮</el-button>
            <el-button type="danger">危险按钮</el-button>
            <el-button type="info">信息按钮</el-button>
          </el-space>

          <el-divider />

          <el-space wrap>
            <el-tag>标签一</el-tag>
            <el-tag type="success">成功</el-tag>
            <el-tag type="warning">警告</el-tag>
            <el-tag type="danger">危险</el-tag>
            <el-tag type="info">信息</el-tag>
          </el-space>

          <el-divider />

          <el-space wrap>
            <el-button @click="showNotification">显示通知</el-button>
            <el-button @click="confirmDialog">确认对话框</el-button>
            <el-button @click="showDialog('用户详情')">显示对话框</el-button>
          </el-space>
        </el-card>
      </el-col>

      <!-- 表单演示 -->
      <el-col :span="12">
        <el-card class="demo-card">
          <template #header>
            <div class="card-header">
              <span>表单演示</span>
            </div>
          </template>

          <el-form :model="formData" label-width="80px">
            <el-form-item label="姓名">
              <el-input v-model="formData.name" placeholder="请输入姓名" />
            </el-form-item>

            <el-form-item label="邮箱">
              <el-input v-model="formData.email" placeholder="请输入邮箱" />
            </el-form-item>

            <el-form-item label="年龄">
              <el-input-number v-model="formData.age" :min="1" :max="100" />
            </el-form-item>

            <el-form-item label="性别">
              <el-select v-model="formData.gender" placeholder="请选择性别">
                <el-option label="男" value="男" />
                <el-option label="女" value="女" />
              </el-select>
            </el-form-item>

            <el-form-item label="描述">
              <el-input
                v-model="formData.description"
                type="textarea"
                placeholder="请输入描述"
                :rows="3"
              />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="submitForm">提交</el-button>
              <el-button @click="resetForm">重置</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 表格演示 -->
      <el-col :span="12">
        <el-card class="demo-card">
          <template #header>
            <div class="card-header">
              <span>表格演示</span>
            </div>
          </template>

          <el-table :data="tableData" style="width: 100%" height="400">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="name" label="姓名" />
            <el-table-column prop="email" label="邮箱" />
            <el-table-column prop="age" label="年龄" width="60" />
            <el-table-column prop="gender" label="性别" width="60" />
            <el-table-column label="操作" width="120">
              <template #default="scope">
                <el-button size="small" @click="showDialog('编辑用户')">
                  编辑
                </el-button>
                <el-button size="small" type="danger" @click="confirmDialog">
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 分页演示 -->
      <el-col :span="24">
        <el-card class="demo-card">
          <template #header>
            <div class="card-header">
              <span>分页演示</span>
            </div>
          </template>

          <div class="pagination-container">
            <el-pagination
              v-model:current-page="pagination.currentPage"
              v-model:page-size="pagination.pageSize"
              :page-sizes="[10, 20, 50, 100]"
              :small="false"
              :disabled="false"
              :background="true"
              layout="total, sizes, prev, pager, next, jumper"
              :total="pagination.total"
              @size-change="handlePageChange"
              @current-change="handlePageChange"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 对话框 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500">
      <span>这是一个对话框内容区域，可以放置任何内容。</span>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="dialogVisible = false">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.demo-container {
  padding: 20px;
}

.demo-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>