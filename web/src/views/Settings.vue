<template>
  <div class="settings-page">
    <div class="page-header">
      <div class="header-left">
        <h3>安全设置</h3>
      </div>
      <div class="header-right">
        <el-switch v-model="policy.enabled" active-text="白名单启用" inactive-text="白名单关闭" />
        <el-button type="primary" @click="savePolicy" :loading="saving">保存</el-button>
        <el-button @click="loadAll" :loading="loading">刷新</el-button>
      </div>
    </div>

    <el-card class="card" header="命令白名单（只读）">
      <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center;">
        <el-button @click="addRule">新增规则</el-button>
        <div style="color: #909399; font-size: 12px;">
          规则使用正则表达式匹配整条命令；默认拒绝包含 `;` / 重定向 / `&&` / `||` 等语法。
        </div>
      </div>

      <el-table :data="policy.rules" style="width: 100%" border>
        <el-table-column label="启用" width="90">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" />
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" width="180">
          <template #default="{ row }">
            <el-input v-model="row.name" placeholder="rule name" />
          </template>
        </el-table-column>
        <el-table-column prop="pattern" label="Pattern (regex)">
          <template #default="{ row }">
            <el-input v-model="row.pattern" placeholder="^docker\\s+stats...$" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ $index }">
            <el-button type="danger" link @click="removeRule($index)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="card" header="命令审计（最近）">
      <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center;">
        <el-input-number v-model="auditLimit" :min="10" :max="1000" />
        <el-button @click="loadAudit" :loading="loading">刷新审计</el-button>
      </div>

      <el-table :data="audit.events" style="width: 100%" border>
        <el-table-column prop="time" label="时间" width="200" />
        <el-table-column prop="host" label="Host" width="120" />
        <el-table-column prop="allowed" label="Allowed" width="110">
          <template #default="{ row }">
            <el-tag :type="row.allowed ? 'success' : 'danger'">{{ row.allowed ? 'YES' : 'NO' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="Reason" width="160" />
        <el-table-column prop="elapsed_ms" label="耗时(ms)" width="110" />
        <el-table-column prop="command" label="Command" />
        <el-table-column prop="error" label="Error" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getCommandAudit, getCommandPolicy, updateCommandPolicy, type CommandPolicy } from '@/api/system'

const loading = ref(false)
const saving = ref(false)

const policy = reactive<CommandPolicy>({
  enabled: true,
  rules: []
})

const audit = reactive<{ events: any[] }>({ events: [] })
const auditLimit = ref(200)

const loadPolicy = async () => {
  const res = await getCommandPolicy()
  policy.enabled = res.enabled
  policy.rules = (res.rules || []).map(r => ({ ...r }))
}

const loadAudit = async () => {
  const res = await getCommandAudit(auditLimit.value)
  audit.events = res.events || []
}

const loadAll = async () => {
  loading.value = true
  try {
    await Promise.all([loadPolicy(), loadAudit()])
  } finally {
    loading.value = false
  }
}

const addRule = () => {
  policy.rules.unshift({ name: '', pattern: '', enabled: true })
}

const removeRule = (idx: number) => {
  policy.rules.splice(idx, 1)
}

const savePolicy = async () => {
  saving.value = true
  try {
    await updateCommandPolicy({ enabled: policy.enabled, rules: policy.rules })
    ElMessage.success('保存成功')
    await loadPolicy()
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadAll()
})
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card {
  width: 100%;
}
</style>
