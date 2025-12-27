<template>
  <div class="scripts-page">
    <div class="page-header">
      <div class="header-left">
        <el-input
          v-model="keyword"
          placeholder="搜索脚本名称"
          :prefix-icon="Search"
          clearable
          style="width: 250px"
          @input="handleSearch"
        />
      </div>
      <div class="header-right">
        <el-button type="primary" :icon="Plus" @click="handleAdd">
          添加脚本
        </el-button>
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :show-file-list="false"
          accept=".sh,.py,.ps1"
          :on-change="handleUpload"
        >
          <el-button :icon="Upload">上传脚本</el-button>
        </el-upload>
      </div>
    </div>

    <ScriptTable
      :scripts="scriptStore.scripts"
      :total="scriptStore.total"
      :loading="scriptStore.loading"
      @view="handleView"
      @edit="handleEdit"
      @delete="handleDelete"
      @test="handleTest"
      @toggle="handleToggle"
      @page-change="handlePageChange"
    />

    <ScriptForm
      v-model:visible="showForm"
      :mode="formMode"
      :script="currentScript"
      @submit="handleFormSubmit"
    />

    <!-- 测试脚本对话框 -->
    <el-dialog
      v-model="showTest"
      title="测试脚本"
      width="600px"
    >
      <div v-if="currentScript">
        <p class="test-info">脚本：{{ currentScript.name }}</p>
        <el-form
          v-if="currentScript.parameters && currentScript.parameters.length > 0"
          :model="testParams"
          label-width="120px"
        >
          <el-form-item
            v-for="param in currentScript.parameters"
            :key="param.name"
            :label="param.name"
            :required="param.required"
          >
            <el-input
              v-model="testParams[param.name]"
              :placeholder="param.description"
            />
          </el-form-item>
        </el-form>
        <p v-else class="no-params">该脚本无需参数</p>

        <div v-if="testResult" class="test-result">
          <div class="result-header">
            <span>执行结果</span>
            <el-tag :type="testResult.exitCode === 0 ? 'success' : 'danger'" size="small">
              退出码: {{ testResult.exitCode }}
            </el-tag>
          </div>
          <pre class="result-output">{{ testResult.output }}</pre>
        </div>
      </div>
      <template #footer>
        <el-button @click="showTest = false">关闭</el-button>
        <el-button type="primary" :loading="testing" @click="executeTest">
          执行测试
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus, Upload } from '@element-plus/icons-vue'
import type { UploadFile } from 'element-plus'
import { useScriptStore } from '@/stores/script'
import type { Script } from '@/api/script'
import ScriptTable from '@/components/script/ScriptTable.vue'
import ScriptForm from '@/components/script/ScriptForm.vue'

const scriptStore = useScriptStore()

const keyword = ref('')
const showForm = ref(false)
const showTest = ref(false)
const formMode = ref<'add' | 'edit' | 'view'>('add')
const currentScript = ref<Script | null>(null)
const testParams = reactive<Record<string, any>>({})
const testResult = ref<{ output: string; exitCode: number } | null>(null)
const testing = ref(false)

onMounted(() => {
  loadData()
})

const loadData = (page = 1, pageSize = 10) => {
  scriptStore.loadScripts({ page, pageSize, keyword: keyword.value })
}

const handleSearch = () => {
  loadData()
}

const handleAdd = () => {
  formMode.value = 'add'
  currentScript.value = null
  showForm.value = true
}

const handleView = (script: Script) => {
  formMode.value = 'view'
  currentScript.value = script
  showForm.value = true
}

const handleEdit = (script: Script) => {
  formMode.value = 'edit'
  currentScript.value = script
  showForm.value = true
}

const handleDelete = async (script: Script) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除脚本 "${script.name}" 吗？`,
      '提示',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await scriptStore.removeScript(script.id)
    ElMessage.success('删除成功')
  } catch {
    // 取消删除
  }
}

const handleTest = (script: Script) => {
  currentScript.value = script
  testResult.value = null
  // 重置测试参数
  Object.keys(testParams).forEach(key => delete testParams[key])
  if (script.parameters) {
    script.parameters.forEach(p => {
      testParams[p.name] = p.default || ''
    })
  }
  showTest.value = true
}

const executeTest = async () => {
  if (!currentScript.value) return

  testing.value = true
  try {
    testResult.value = await scriptStore.test(currentScript.value.id, testParams)
    ElMessage.success('测试完成')
  } catch (error) {
    ElMessage.error('测试执行失败')
  } finally {
    testing.value = false
  }
}

const handleToggle = async (id: string, enabled: boolean) => {
  try {
    await scriptStore.toggle(id, enabled)
    ElMessage.success(enabled ? '已启用' : '已禁用')
  } catch (error) {
    // 错误已在 store 中处理
  }
}

const handlePageChange = (page: number, pageSize: number) => {
  loadData(page, pageSize)
}

const handleUpload = async (file: UploadFile) => {
  if (!file.raw) return

  try {
    await scriptStore.upload(file.raw)
    ElMessage.success('上传成功')
  } catch (error) {
    // 错误已在 store 中处理
  }
}

const handleFormSubmit = async (data: Omit<Script, 'id' | 'createdAt' | 'updatedAt'>) => {
  try {
    if (formMode.value === 'edit' && currentScript.value) {
      await scriptStore.editScript(currentScript.value.id, data)
      ElMessage.success('更新成功')
    } else {
      await scriptStore.addScript(data)
      ElMessage.success('添加成功')
    }
  } catch (error) {
    // 错误已在 store 中处理
  }
}
</script>

<style scoped>
.scripts-page {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.header-right {
  display: flex;
  gap: 10px;
}

.test-info {
  font-size: 14px;
  color: #606266;
  margin-bottom: 15px;
}

.no-params {
  color: #909399;
  font-size: 14px;
}

.test-result {
  margin-top: 20px;
  border-top: 1px solid #ebeef5;
  padding-top: 15px;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-weight: 600;
  color: #303133;
}

.result-output {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 15px;
  border-radius: 6px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
