<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="750px"
    @close="handleClose"
  >
    <el-form
      v-if="mode !== 'view'"
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
      label-position="right"
    >
      <el-form-item label="脚本名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入脚本名称" />
      </el-form-item>
      <el-form-item label="脚本语言" prop="language">
        <el-select v-model="formData.language" placeholder="请选择脚本语言" style="width: 100%">
          <el-option label="Bash" value="bash" />
          <el-option label="Python" value="python" />
          <el-option label="PowerShell" value="powershell" />
        </el-select>
      </el-form-item>
      <el-form-item label="描述" prop="description">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="2"
          placeholder="请输入脚本描述"
        />
      </el-form-item>
      <el-form-item label="脚本内容" prop="content">
        <div class="code-editor">
          <el-input
            v-model="formData.content"
            type="textarea"
            :rows="15"
            placeholder="请输入脚本内容"
            class="code-textarea"
          />
        </div>
      </el-form-item>
      <el-form-item label="参数定义">
        <div class="params-editor">
          <div
            v-for="(param, index) in formData.parameters"
            :key="index"
            class="param-row"
          >
            <el-input v-model="param.name" placeholder="参数名" style="width: 120px" />
            <el-select v-model="param.type" placeholder="类型" style="width: 100px">
              <el-option label="string" value="string" />
              <el-option label="number" value="number" />
              <el-option label="boolean" value="boolean" />
            </el-select>
            <el-input v-model="param.description" placeholder="描述" style="flex: 1" />
            <el-checkbox v-model="param.required">必填</el-checkbox>
            <el-button type="danger" :icon="Delete" circle size="small" @click="removeParam(index)" />
          </div>
          <el-button type="primary" plain size="small" :icon="Plus" @click="addParam">
            添加参数
          </el-button>
        </div>
      </el-form-item>
      <el-form-item label="启用状态">
        <el-switch v-model="formData.enabled" />
      </el-form-item>
    </el-form>

    <!-- 查看模式 -->
    <div v-else class="view-mode">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="脚本名称">{{ script?.name }}</el-descriptions-item>
        <el-descriptions-item label="脚本语言">
          <el-tag :type="getLanguageType(script?.language)">{{ script?.language }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="script?.enabled ? 'success' : 'info'">
            {{ script?.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="更新时间">{{ formatDate(script?.updatedAt) }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ script?.description || '-' }}</el-descriptions-item>
      </el-descriptions>
      <div class="code-preview">
        <div class="code-header">脚本内容</div>
        <pre class="code-content"><code>{{ script?.content }}</code></pre>
      </div>
      <div v-if="script?.parameters && script.parameters.length > 0" class="params-preview">
        <div class="params-header">参数列表</div>
        <el-table :data="script.parameters" size="small">
          <el-table-column prop="name" label="参数名" width="120" />
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="description" label="描述" />
          <el-table-column prop="required" label="必填" width="80">
            <template #default="{ row }">
              <el-tag :type="row.required ? 'danger' : 'info'" size="small">
                {{ row.required ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">{{ mode === 'view' ? '关闭' : '取消' }}</el-button>
      <el-button v-if="mode !== 'view'" type="primary" :loading="submitting" @click="handleSubmit">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { Script, ScriptParameter } from '@/api/script'
import { Plus, Delete } from '@element-plus/icons-vue'

const props = defineProps<{
  visible: boolean
  mode: 'add' | 'edit' | 'view'
  script?: Script | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  submit: [data: Omit<Script, 'id' | 'createdAt' | 'updatedAt'>]
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)

const dialogTitle = computed(() => {
  switch (props.mode) {
    case 'add':
      return '添加脚本'
    case 'edit':
      return '编辑脚本'
    case 'view':
      return '查看脚本'
    default:
      return ''
  }
})

const defaultFormData = {
  name: '',
  language: 'bash' as const,
  description: '',
  content: '',
  parameters: [] as ScriptParameter[],
  enabled: true
}

const formData = reactive({ ...defaultFormData })

const rules: FormRules = {
  name: [
    { required: true, message: '请输入脚本名称', trigger: 'blur' }
  ],
  language: [
    { required: true, message: '请选择脚本语言', trigger: 'change' }
  ],
  content: [
    { required: true, message: '请输入脚本内容', trigger: 'blur' }
  ]
}

watch(
  () => props.visible,
  (val) => {
    if (val && props.script && props.mode !== 'add') {
      Object.assign(formData, {
        name: props.script.name,
        language: props.script.language,
        description: props.script.description || '',
        content: props.script.content,
        parameters: props.script.parameters ? [...props.script.parameters] : [],
        enabled: props.script.enabled
      })
    } else if (val && props.mode === 'add') {
      Object.assign(formData, { ...defaultFormData, parameters: [] })
    }
  }
)

const getLanguageType = (language?: string) => {
  switch (language) {
    case 'bash':
      return 'success'
    case 'python':
      return 'primary'
    case 'powershell':
      return 'warning'
    default:
      return 'info'
  }
}

const formatDate = (date?: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const addParam = () => {
  formData.parameters.push({
    name: '',
    type: 'string',
    description: '',
    required: false
  })
}

const removeParam = (index: number) => {
  formData.parameters.splice(index, 1)
}

const handleClose = () => {
  emit('update:visible', false)
  formRef.value?.resetFields()
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      const data = {
        name: formData.name,
        language: formData.language,
        description: formData.description,
        content: formData.content,
        parameters: formData.parameters.filter(p => p.name),
        enabled: formData.enabled
      }
      emit('submit', data)
      handleClose()
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
.code-editor {
  width: 100%;
}

.code-textarea :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.params-editor {
  width: 100%;
}

.param-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.view-mode {
  color: #606266;
}

.code-preview {
  margin-top: 20px;
}

.code-header,
.params-header {
  font-weight: 600;
  color: #303133;
  margin-bottom: 10px;
}

.code-content {
  background: #1e1e1e;
  color: #d4d4d4;
  padding: 15px;
  border-radius: 6px;
  overflow-x: auto;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.5;
  max-height: 300px;
  overflow-y: auto;
}

.params-preview {
  margin-top: 20px;
}
</style>
