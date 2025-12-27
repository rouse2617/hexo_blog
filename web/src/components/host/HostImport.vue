<template>
  <el-dialog
    :model-value="visible"
    title="批量导入主机"
    width="650px"
    @close="handleClose"
  >
    <el-tabs v-model="activeTab">
      <el-tab-pane label="文本导入" name="text">
        <div class="import-tips">
          <p>请按以下格式输入主机信息，每行一个主机：</p>
          <code>名称,地址,端口,用户名,密码</code>
          <p>示例：</p>
          <code>Web服务器1,192.168.1.10,22,root,password123</code>
        </div>
        <el-input
          v-model="textContent"
          type="textarea"
          :rows="10"
          placeholder="请输入主机信息..."
        />
      </el-tab-pane>
      <el-tab-pane label="文件导入" name="file">
        <div class="import-tips">
          <p>支持 CSV 或 JSON 格式文件</p>
          <p>CSV 格式：名称,地址,端口,用户名,密码</p>
          <p>JSON 格式：[{"name": "", "host": "", "port": 22, "username": "", "password": ""}]</p>
        </div>
        <el-upload
          ref="uploadRef"
          class="upload-area"
          drag
          :auto-upload="false"
          :limit="1"
          accept=".csv,.json"
          :on-change="handleFileChange"
        >
          <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
          <div class="el-upload__text">
            将文件拖到此处，或<em>点击上传</em>
          </div>
          <template #tip>
            <div class="el-upload__tip">
              支持 .csv 或 .json 文件
            </div>
          </template>
        </el-upload>
      </el-tab-pane>
    </el-tabs>

    <div v-if="parsedHosts.length > 0" class="preview-section">
      <h4>预览（共 {{ parsedHosts.length }} 台主机）</h4>
      <el-table :data="parsedHosts" max-height="200" size="small">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="host" label="地址" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column prop="username" label="用户名" />
        <el-table-column label="密码" width="80">
          <template #default>******</template>
        </el-table-column>
      </el-table>
    </div>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button @click="handleParse" :disabled="!canParse">
        解析
      </el-button>
      <el-button
        type="primary"
        :loading="submitting"
        :disabled="parsedHosts.length === 0"
        @click="handleSubmit"
      >
        导入 ({{ parsedHosts.length }})
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'
import type { UploadFile } from 'element-plus'
import type { Host } from '@/api/host'

defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  submit: [hosts: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>[]]
}>()

const activeTab = ref('text')
const textContent = ref('')
const fileContent = ref('')
const parsedHosts = ref<Omit<Host, 'id' | 'createdAt' | 'updatedAt'>[]>([])
const submitting = ref(false)

const canParse = computed(() => {
  if (activeTab.value === 'text') {
    return textContent.value.trim().length > 0
  }
  return fileContent.value.length > 0
})

const handleFileChange = (file: UploadFile) => {
  if (!file.raw) return

  const reader = new FileReader()
  reader.onload = (e) => {
    fileContent.value = e.target?.result as string
  }
  reader.readAsText(file.raw)
}

const handleParse = () => {
  const content = activeTab.value === 'text' ? textContent.value : fileContent.value

  if (!content.trim()) {
    ElMessage.warning('请输入或上传主机信息')
    return
  }

  try {
    // 尝试 JSON 解析
    if (content.trim().startsWith('[')) {
      const data = JSON.parse(content)
      parsedHosts.value = data.map((item: any) => ({
        name: item.name || '',
        host: item.host || item.ip || '',
        port: item.port || 22,
        username: item.username || item.user || 'root',
        password: item.password || '',
        privateKey: item.privateKey || '',
        tags: item.tags || [],
        description: item.description || ''
      }))
    } else {
      // CSV 解析
      const lines = content.trim().split('\n')
      parsedHosts.value = lines
        .filter(line => line.trim())
        .map(line => {
          const parts = line.split(',').map(p => p.trim())
          return {
            name: parts[0] || '',
            host: parts[1] || '',
            port: parseInt(parts[2]) || 22,
            username: parts[3] || 'root',
            password: parts[4] || '',
            tags: [],
            description: ''
          }
        })
        .filter(h => h.name && h.host)
    }

    if (parsedHosts.value.length === 0) {
      ElMessage.warning('未解析到有效的主机信息')
    } else {
      ElMessage.success(`成功解析 ${parsedHosts.value.length} 台主机`)
    }
  } catch (error) {
    console.error('解析失败:', error)
    ElMessage.error('解析失败，请检查格式是否正确')
  }
}

const handleClose = () => {
  emit('update:visible', false)
  textContent.value = ''
  fileContent.value = ''
  parsedHosts.value = []
}

const handleSubmit = async () => {
  if (parsedHosts.value.length === 0) return

  submitting.value = true
  try {
    emit('submit', parsedHosts.value)
    handleClose()
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.import-tips {
  background: #f5f7fa;
  padding: 12px 15px;
  border-radius: 6px;
  margin-bottom: 15px;
  font-size: 13px;
  color: #606266;
}

.import-tips p {
  margin: 5px 0;
}

.import-tips code {
  background: #e4e7ed;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

.upload-area {
  width: 100%;
}

.preview-section {
  margin-top: 20px;
  padding-top: 15px;
  border-top: 1px solid #ebeef5;
}

.preview-section h4 {
  margin: 0 0 10px;
  font-size: 14px;
  color: #303133;
}
</style>
