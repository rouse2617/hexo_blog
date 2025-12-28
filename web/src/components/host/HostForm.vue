<template>
  <el-dialog
    :model-value="visible"
    :title="isEdit ? '编辑主机' : '添加主机'"
    width="600px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="110px"
      label-position="right"
    >
      <el-form-item label="主机名称" prop="name">
        <el-input v-model="formData.name" placeholder="请输入主机名称" />
      </el-form-item>

      <el-form-item label="主机地址" prop="host">
        <el-input v-model="formData.host" placeholder="请输入 IP 地址或域名" />
      </el-form-item>

      <el-form-item label="端口" prop="port">
        <el-input-number v-model="formData.port" :min="1" :max="65535" style="width: 100%" />
      </el-form-item>

      <el-form-item label="用户名" prop="username">
        <el-input v-model="formData.username" placeholder="请输入用户名 (root/ubuntu等)" />
      </el-form-item>

      <el-form-item label="认证方式">
        <el-select v-model="authType" placeholder="选择认证方式" style="width: 100%">
          <el-option value="auto">
            <div class="auth-option">
              <span class="auth-option-label">🔐 自动检测 (推荐)</span>
              <span class="auth-option-desc">自动尝试 SSH Agent、私钥文件、密码</span>
            </div>
          </el-option>
          <el-option value="password">
            <div class="auth-option">
              <span class="auth-option-label">🔑 密码认证</span>
              <span class="auth-option-desc">使用密码登录</span>
            </div>
          </el-option>
          <el-option value="key">
            <div class="auth-option">
              <span class="auth-option-label">🗝️ 私钥文件</span>
              <span class="auth-option-desc">使用私钥文件 (默认 ~/.ssh/id_rsa)</span>
            </div>
          </el-option>
          <el-option value="key_content">
            <div class="auth-option">
              <span class="auth-option-label">📋 私钥内容</span>
              <span class="auth-option-desc">直接粘贴私钥内容</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>

      <!-- 自动检测模式提示 -->
      <el-form-item v-if="authType === 'auto'">
        <el-alert type="info" :closable="false" show-icon>
          <template #title>
            <div class="auth-info">
              <p>系统将按以下顺序自动尝试认证：</p>
              <ol>
                <li><strong>SSH Agent</strong> - 检测系统 SSH Agent (SSH_AUTH_SOCK)</li>
                <li><strong>私钥文件</strong> - 自动尝试常见私钥：
                  <ul class="key-list">
                    <li>~/.ssh/id_rsa</li>
                    <li>~/.ssh/id_ed25519 (推荐)</li>
                    <li>~/.ssh/id_ecdsa</li>
                    <li>~/.ssh/id_dsa</li>
                  </ul>
                </li>
                <li><strong>密码</strong> - 如果配置了密码</li>
              </ol>
              <p class="tip">💡 如果已配置免密登录，选择"自动检测"即可，无需填写任何凭证</p>
            </div>
          </template>
        </el-alert>
      </el-form-item>

      <!-- 密码认证 -->
      <el-form-item v-if="authType === 'password'" label="密码" prop="password">
        <el-input
          v-model="formData.password"
          type="password"
          placeholder="请输入 SSH 登录密码"
          show-password
        />
      </el-form-item>

      <!-- 私钥文件认证 -->
      <template v-if="authType === 'key'">
        <el-form-item label="私钥路径">
          <el-input
            v-model="formData.keyPath"
            placeholder="留空则自动使用 ~/.ssh/id_rsa"
          >
            <template #append>
              <el-button @click="formData.keyPath = '~/.ssh/id_ed25519'">ed25519</el-button>
              <el-button @click="formData.keyPath = '~/.ssh/id_rsa'">rsa</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-alert type="info" :closable="false" show-icon>
            <p>支持路径格式：</p>
            <ul class="key-list">
              <li><code>~/.ssh/id_rsa</code> - 自动展开为用户主目录</li>
              <li><code>/home/user/.ssh/my_key</code> - 绝对路径</li>
              <li><code>./keys/my_key</code> - 相对路径</li>
            </ul>
          </el-alert>
        </el-form-item>
      </template>

      <!-- 私钥内容认证 -->
      <el-form-item v-if="authType === 'key_content'" label="私钥内容" prop="privateKey">
        <el-input
          v-model="formData.privateKey"
          type="textarea"
          :rows="6"
          placeholder="请粘贴完整的私钥内容，包括 BEGIN 和 END 行"
        />
        <div class="key-hint">
          💡 私钥格式示例：以 -----BEGIN PRIVATE KEY----- 或 -----BEGIN RSA PRIVATE KEY----- 开头
        </div>
      </el-form-item>

      <!-- 自动检测模式下可选的密码 -->
      <el-form-item v-if="authType === 'auto'" label="密码 (可选)">
        <el-input
          v-model="formData.password"
          type="password"
          placeholder="留空则仅使用 SSH Agent 和私钥文件"
          show-password
        />
        <div class="form-hint">
          可选：如果私钥文件都失败，将尝试使用此密码
        </div>
      </el-form-item>

      <!-- 自动检测模式下可选的私钥路径 -->
      <el-form-item v-if="authType === 'auto'" label="私钥路径 (可选)">
        <el-input
          v-model="formData.keyPath"
          placeholder="留空则自动尝试所有常见私钥文件"
        />
        <div class="form-hint">
          可选：指定私钥路径，留空则自动尝试 ~/.ssh 下的所有私钥
        </div>
      </el-form-item>

      <el-form-item label="标签">
        <el-select
          v-model="formData.tags"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="输入标签后回车添加"
          style="width: 100%"
        >
          <el-option
            v-for="tag in commonTags"
            :key="tag"
            :label="tag"
            :value="tag"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="分组">
        <el-input
          v-model="formData.group"
          placeholder="可选：主机分组名称"
        />
      </el-form-item>

      <el-form-item label="描述">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="2"
          placeholder="请输入描述信息"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { Host } from '@/api/host'

const props = defineProps<{
  visible: boolean
  host?: Host | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  submit: [data: Omit<Host, 'id' | 'createdAt' | 'updatedAt'>]
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const authType = ref<'auto' | 'password' | 'key' | 'key_content'>('auto')

const commonTags = ['生产环境', '测试环境', '开发环境', 'Web服务器', '数据库', '缓存', '负载均衡']

const isEdit = computed(() => !!props.host?.id)

const defaultFormData = {
  name: '',
  host: '',
  port: 22,
  username: 'root',
  password: '',
  privateKey: '',
  keyPath: '',
  tags: [] as string[],
  group: '',
  description: ''
}

const formData = reactive({ ...defaultFormData })

const rules = computed<FormRules>(() => {
  const baseRules: FormRules = {
    name: [{ required: true, message: '请输入主机名称', trigger: 'blur' }],
    host: [{ required: true, message: '请输入主机地址', trigger: 'blur' }],
    port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
    username: [{ required: true, message: '请输入用户名', trigger: 'blur' }]
  }

  // 根据认证类型添加验证
  if (authType.value === 'password') {
    baseRules.password = [{ required: true, message: '请输入密码', trigger: 'blur' }]
  } else if (authType.value === 'key_content') {
    baseRules.privateKey = [{ required: true, message: '请输入私钥内容', trigger: 'blur' }]
  }

  return baseRules
})

watch(
  () => props.visible,
  (val) => {
    if (val && props.host) {
      Object.assign(formData, {
        name: props.host.name,
        host: props.host.host,
        port: props.host.port,
        username: props.host.username,
        password: props.host.password || '',
        privateKey: props.host.privateKey || '',
        keyPath: props.host.keyPath || '',
        tags: props.host.tags || [],
        group: props.host.group || '',
        description: props.host.description || ''
      })
      // 推断认证类型
      if (props.host.privateKey) {
        authType.value = 'key_content'
      } else if (props.host.keyPath) {
        authType.value = 'key'
      } else if (props.host.password) {
        authType.value = 'password'
      } else {
        authType.value = 'auto'
      }
    } else if (val) {
      Object.assign(formData, defaultFormData)
      authType.value = 'auto'
    }
  }
)

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
      // 根据认证类型构建数据
      const data: any = {
        name: formData.name,
        host: formData.host,
        port: formData.port,
        username: formData.username,
        tags: formData.tags,
        group: formData.group,
        description: formData.description
      }

      // 设置认证类型和相关字段
      if (authType.value === 'auto') {
        data.authType = 'auto'
        if (formData.password) data.password = formData.password
        if (formData.keyPath) data.keyPath = formData.keyPath
      } else if (authType.value === 'password') {
        data.authType = 'password'
        data.password = formData.password
      } else if (authType.value === 'key') {
        data.authType = 'key'
        if (formData.keyPath) {
          data.keyPath = formData.keyPath
        }
      } else if (authType.value === 'key_content') {
        data.authType = 'key_content'
        data.privateKey = formData.privateKey
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
.auth-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.auth-option-label {
  font-weight: 500;
  font-size: 14px;
}

.auth-option-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.auth-info {
  font-size: 13px;
}

.auth-info p {
  margin: 4px 0;
}

.auth-info ol {
  margin: 8px 0;
  padding-left: 20px;
}

.auth-info li {
  margin: 4px 0;
}

.key-list {
  margin: 4px 0;
  padding-left: 20px;
}

.key-list li {
  margin: 2px 0;
  font-family: 'Courier New', monospace;
  font-size: 12px;
}

.tip {
  margin-top: 8px;
  padding: 8px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  font-weight: 500;
}

.key-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.form-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

:deep(.el-alert) {
  padding: 8px 12px;
}

:deep(.el-alert__title) {
  font-size: 13px;
}

:deep(.el-select-dropdown__item) {
  height: auto;
  padding: 8px 12px;
}
</style>
