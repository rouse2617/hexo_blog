<template>
  <el-dialog
    :model-value="visible"
    :title="isEdit ? '编辑主机' : '添加主机'"
    width="550px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
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
        <el-input v-model="formData.username" placeholder="请输入用户名" />
      </el-form-item>
      <el-form-item label="认证方式">
        <el-radio-group v-model="authType">
          <el-radio value="password">密码</el-radio>
          <el-radio value="privateKey">私钥</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="authType === 'password'" label="密码" prop="password">
        <el-input
          v-model="formData.password"
          type="password"
          placeholder="请输入密码"
          show-password
        />
      </el-form-item>
      <el-form-item v-if="authType === 'privateKey'" label="私钥" prop="privateKey">
        <el-input
          v-model="formData.privateKey"
          type="textarea"
          :rows="4"
          placeholder="请粘贴私钥内容"
        />
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
const authType = ref<'password' | 'privateKey'>('password')

const commonTags = ['生产环境', '测试环境', '开发环境', 'Web服务器', '数据库', '缓存']

const isEdit = computed(() => !!props.host?.id)

const defaultFormData = {
  name: '',
  host: '',
  port: 22,
  username: 'root',
  password: '',
  privateKey: '',
  tags: [] as string[],
  description: ''
}

const formData = reactive({ ...defaultFormData })

const rules = computed<FormRules>(() => ({
  name: [
    { required: true, message: '请输入主机名称', trigger: 'blur' }
  ],
  host: [
    { required: true, message: '请输入主机地址', trigger: 'blur' }
  ],
  port: [
    { required: true, message: '请输入端口', trigger: 'blur' }
  ],
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: authType.value === 'password' ? [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ] : [],
  privateKey: authType.value === 'privateKey' ? [
    { required: true, message: '请输入私钥', trigger: 'blur' }
  ] : []
}))

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
        tags: props.host.tags || [],
        description: props.host.description || ''
      })
      authType.value = props.host.privateKey ? 'privateKey' : 'password'
    } else if (val) {
      Object.assign(formData, defaultFormData)
      authType.value = 'password'
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
      const data = {
        name: formData.name,
        host: formData.host,
        port: formData.port,
        username: formData.username,
        password: authType.value === 'password' ? formData.password : undefined,
        privateKey: authType.value === 'privateKey' ? formData.privateKey : undefined,
        tags: formData.tags,
        description: formData.description
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
</style>
