<template>
  <div class="host-tree">
    <div class="host-tree-header">
      <h3>主机列表</h3>
      <el-input
        v-model="keyword"
        placeholder="搜索主机"
        :prefix-icon="Search"
        clearable
        size="small"
        style="margin-top: 12px"
        @input="handleSearch"
      />
    </div>
    
    <div class="host-tree-toolbar">
      <el-button size="small" @click="handleSelectAll">全选</el-button>
      <el-button size="small" @click="handleInvertSelection">反选</el-button>
      <span class="selected-count">已选: {{ selectedHosts.length }}</span>
    </div>

    <div class="host-tree-content">
      <el-tree
        ref="treeRef"
        :data="treeData"
        :props="treeProps"
        show-checkbox
        node-key="id"
        :default-expand-all="true"
        :filter-node-method="filterNode"
        @check="handleCheck"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <el-icon v-if="data.isGroup" :size="16" style="margin-right: 4px">
              <Folder />
            </el-icon>
            <el-icon
              v-else
              :size="14"
              :color="getStatusColor(data.host?.status)"
              style="margin-right: 6px"
            >
              <Monitor />
            </el-icon>
            <span>{{ node.label }}</span>
          </span>
        </template>
      </el-tree>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { Search, Monitor, Folder } from '@element-plus/icons-vue'
import type { ElTree } from 'element-plus'
import { useHostStore } from '@/stores/host'
import type { Host } from '@/api/host'

interface TreeNode {
  id: string
  label: string
  children?: TreeNode[]
  isGroup?: boolean
  host?: Host
}

const emit = defineEmits<{
  (e: 'selection-change', hosts: string[]): void
}>()

const hostStore = useHostStore()
const treeRef = ref<InstanceType<typeof ElTree>>()
const keyword = ref('')
const selectedHosts = ref<string[]>([])

const treeProps = {
  children: 'children',
  label: 'label'
}

// 构建树形数据
const treeData = computed(() => {
  const hosts = hostStore.allHosts
  const groupMap = new Map<string, Host[]>()
  
  // 按分组组织主机
  hosts.forEach(host => {
    const group = host.group || '未分组'
    if (!groupMap.has(group)) {
      groupMap.set(group, [])
    }
    groupMap.get(group)!.push(host)
  })
  
  // 转换为树形结构
  const nodes: TreeNode[] = []
  groupMap.forEach((hosts, groupName) => {
    const groupNode: TreeNode = {
      id: `group-${groupName}`,
      label: groupName,
      isGroup: true,
      children: hosts.map(host => ({
        id: host.id || host.name,
        label: host.name,
        isGroup: false,
        host
      }))
    }
    nodes.push(groupNode)
  })
  
  return nodes
})

// 过滤节点
const filterNode = (value: string, data: TreeNode) => {
  if (!value) return true
  if (data.isGroup) {
    // 分组节点：如果子节点有匹配的，则显示
    return data.children?.some(child => 
      child.label.toLowerCase().includes(value.toLowerCase()) ||
      child.host?.host.toLowerCase().includes(value.toLowerCase())
    ) ?? false
  }
  return (
    data.label.toLowerCase().includes(value.toLowerCase()) ||
    data.host?.host.toLowerCase().includes(value.toLowerCase())
  )
}

// 搜索
const handleSearch = () => {
  treeRef.value?.filter(keyword.value)
}

// 获取状态颜色
const getStatusColor = (status?: string) => {
  switch (status) {
    case 'online':
      return '#67c23a'
    case 'offline':
      return '#f56c6c'
    default:
      return '#909399'
  }
}

// 全选
const handleSelectAll = () => {
  const allHostIds = hostStore.allHosts
    .filter(h => {
      // 根据搜索关键词过滤
      if (!keyword.value) return true
      const kw = keyword.value.toLowerCase()
      return h.name.toLowerCase().includes(kw) || h.host.toLowerCase().includes(kw)
    })
    .map(h => h.id || h.name)
  
  treeRef.value?.setCheckedKeys(allHostIds, false)
  updateSelectedHosts()
}

// 反选
const handleInvertSelection = () => {
  const allHostIds = hostStore.allHosts
    .filter(h => {
      if (!keyword.value) return true
      const kw = keyword.value.toLowerCase()
      return h.name.toLowerCase().includes(kw) || h.host.toLowerCase().includes(kw)
    })
    .map(h => h.id || h.name)
  
  const currentChecked = treeRef.value?.getCheckedKeys(false) as string[] || []
  const newChecked = allHostIds.filter(id => !currentChecked.includes(id))
  treeRef.value?.setCheckedKeys(newChecked, false)
  updateSelectedHosts()
}

// 处理复选框变化
const handleCheck = () => {
  updateSelectedHosts()
}

// 更新选中的主机列表
const updateSelectedHosts = () => {
  const checkedKeys = treeRef.value?.getCheckedKeys(false) as string[] || []
  // 过滤掉分组节点，只保留主机节点
  selectedHosts.value = checkedKeys.filter(key => !key.startsWith('group-'))
  emit('selection-change', selectedHosts.value)
}

// 监听搜索关键词变化
watch(keyword, () => {
  handleSearch()
})

// 加载所有主机
onMounted(async () => {
  await hostStore.loadAllHosts()
})
</script>

<style scoped>
.host-tree {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e4e7ed;
}

.host-tree-header {
  margin-bottom: 12px;
}

.host-tree-header h3 {
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.host-tree-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e4e7ed;
}

.selected-count {
  margin-left: auto;
  font-size: 12px;
  color: #909399;
}

.host-tree-content {
  flex: 1;
  overflow: auto;
}

.tree-node {
  display: flex;
  align-items: center;
  flex: 1;
}

:deep(.el-tree-node__content) {
  height: 32px;
}

:deep(.el-checkbox) {
  margin-right: 8px;
}
</style>
