<template>
  <div class="host-tree">
    <div class="host-tree-header">
      <div class="header-controls">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索主机名称或地址"
          clearable
          size="small"
          style="flex: 1; margin-right: 10px"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-select
          v-model="statusFilter"
          placeholder="状态"
          clearable
          size="small"
          style="width: 100px; margin-right: 10px"
        >
          <el-option label="全部" value="" />
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
        </el-select>
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button label="group">按分组</el-radio-button>
          <el-radio-button label="tag">按标签</el-radio-button>
        </el-radio-group>
      </div>
    </div>
    
    <div class="host-tree-toolbar">
      <el-button size="small" @click="handleSelectAll">全选</el-button>
      <el-button size="small" @click="handleSelectNone">清空</el-button>
      <el-button size="small" @click="handleInvertSelection">反选</el-button>
      <span class="selected-count">已选择: {{ selectedHosts.length }}</span>
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
        style="overflow: auto"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <el-icon 
              v-if="data.isGroup || data.isTagGroup" 
              :size="16" 
              style="margin-right: 4px; color: #409eff"
            >
              <Folder v-if="data.isGroup" />
              <CollectionTag v-else />
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
            <el-tag
              v-if="!data.isGroup && !data.isTagGroup && data.host?.tags && data.host.tags.length > 0"
              v-for="tag in data.host.tags"
              :key="tag"
              size="small"
              type="info"
              style="margin-left: 8px"
            >
              {{ tag }}
            </el-tag>
          </span>
        </template>
      </el-tree>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { Search, Monitor, Folder, CollectionTag } from '@element-plus/icons-vue'
import type { ElTree } from 'element-plus'
import { useHostStore } from '@/stores/host'
import { useConsoleStore } from '@/stores/console'
import type { Host } from '@/api/host'

interface TreeNode {
  id: string
  label: string
  children?: TreeNode[]
  isGroup?: boolean
  isTagGroup?: boolean
  host?: Host
  groupName?: string
  tagName?: string
}

const hostStore = useHostStore()
const consoleStore = useConsoleStore()
const treeRef = ref<InstanceType<typeof ElTree>>()
const searchKeyword = ref('')
const statusFilter = ref('')
const viewMode = ref<'group' | 'tag'>('group') // 视图模式：按分组或按标签

const treeProps = {
  children: 'children',
  label: 'label'
}

// 构建树形数据
const treeData = computed(() => {
  const hosts = hostStore.allHosts
  
  // 先进行状态筛选
  let filteredHosts = hosts
  if (statusFilter.value) {
    filteredHosts = hosts.filter(h => h.status === statusFilter.value)
  }
  
  // 再进行关键词搜索
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    filteredHosts = filteredHosts.filter(h =>
      h.name.toLowerCase().includes(keyword) ||
      h.host.toLowerCase().includes(keyword) ||
      (h.group || '').toLowerCase().includes(keyword) ||
      (h.tags || []).some(tag => tag.toLowerCase().includes(keyword))
    )
  }
  
  const nodes: TreeNode[] = []
  
  if (viewMode.value === 'group') {
    // 按分组组织
    const groupMap = new Map<string, Host[]>()
    filteredHosts.forEach(host => {
      const group = host.group || '未分组'
      if (!groupMap.has(group)) {
        groupMap.set(group, [])
      }
      groupMap.get(group)!.push(host)
    })
    
    groupMap.forEach((hosts, groupName) => {
      const groupNode: TreeNode = {
        id: `group-${groupName}`,
        label: `${groupName} (${hosts.length})`,
        isGroup: true,
        groupName,
        children: hosts.map(host => ({
          id: host.id || host.name,
          label: `${host.name} (${host.host}:${host.port})`,
          isGroup: false,
          host
        }))
      }
      nodes.push(groupNode)
    })
  } else {
    // 按标签组织
    const tagMap = new Map<string, Host[]>()
    const untagged: Host[] = []
    
    filteredHosts.forEach(host => {
      if (host.tags && host.tags.length > 0) {
        host.tags.forEach(tag => {
          if (!tagMap.has(tag)) {
            tagMap.set(tag, [])
          }
          tagMap.get(tag)!.push(host)
        })
      } else {
        untagged.push(host)
      }
    })
    
    // 添加标签分组
    tagMap.forEach((hosts, tagName) => {
      const tagNode: TreeNode = {
        id: `tag-${tagName}`,
        label: `${tagName} (${hosts.length})`,
        isGroup: true,
        isTagGroup: true,
        tagName,
        children: hosts.map(host => ({
          id: host.id || host.name,
          label: `${host.name} (${host.host}:${host.port})`,
          isGroup: false,
          host
        }))
      }
      nodes.push(tagNode)
    })
    
    // 添加未标签分组
    if (untagged.length > 0) {
      nodes.push({
        id: 'tag-未标签',
        label: `未标签 (${untagged.length})`,
        isGroup: true,
        isTagGroup: true,
        tagName: '未标签',
        children: untagged.map(host => ({
          id: host.id || host.name,
          label: `${host.name} (${host.host}:${host.port})`,
          isGroup: false,
          host
        }))
      })
    }
  }
  
  return nodes
})

// 过滤节点（用于搜索）
const filterNode = (value: string, data: TreeNode) => {
  if (!value) return true
  if (data.isGroup || data.isTagGroup) {
    // 分组节点：如果标签或子节点有匹配的，则显示
    return data.label.toLowerCase().includes(value.toLowerCase()) ||
      (data.children?.some(child => 
        child.label.toLowerCase().includes(value.toLowerCase()) ||
        child.host?.host.toLowerCase().includes(value.toLowerCase())
      ) ?? false)
  }
  return (
    data.label.toLowerCase().includes(value.toLowerCase()) ||
    data.host?.host.toLowerCase().includes(value.toLowerCase())
  )
}

// 搜索
const handleSearch = () => {
  treeRef.value?.filter(searchKeyword.value)
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

const selectedHosts = computed(() => consoleStore.selectedHosts)

// 全选
const handleSelectAll = () => {
  const allHostIds = hostStore.allHosts
    .filter(h => {
      // 根据搜索关键词和状态过滤
      if (statusFilter.value && h.status !== statusFilter.value) return false
      if (!searchKeyword.value) return true
      const kw = searchKeyword.value.toLowerCase()
      return h.name.toLowerCase().includes(kw) || 
             h.host.toLowerCase().includes(kw) ||
             (h.group || '').toLowerCase().includes(kw) ||
             (h.tags || []).some(tag => tag.toLowerCase().includes(kw))
    })
    .map(h => h.id || h.name)
  
  treeRef.value?.setCheckedKeys(allHostIds, false)
  updateSelectedHosts()
}

// 清空选择
const handleSelectNone = () => {
  treeRef.value?.setCheckedKeys([], false)
  updateSelectedHosts()
}

// 反选
const handleInvertSelection = () => {
  const allHostIds = hostStore.allHosts
    .filter(h => {
      if (statusFilter.value && h.status !== statusFilter.value) return false
      if (!searchKeyword.value) return true
      const kw = searchKeyword.value.toLowerCase()
      return h.name.toLowerCase().includes(kw) || 
             h.host.toLowerCase().includes(kw) ||
             (h.group || '').toLowerCase().includes(kw) ||
             (h.tags || []).some(tag => tag.toLowerCase().includes(kw))
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
  const hostIds = checkedKeys.filter(key => !key.startsWith('group-') && !key.startsWith('tag-'))
  consoleStore.selectHosts(hostIds)
}

// 监听store中的选中变化，同步到树
watch(() => consoleStore.selectedHosts, (newHosts) => {
  nextTick(() => {
    treeRef.value?.setCheckedKeys(newHosts, false)
  })
}, { deep: true })

// 监听搜索关键词变化
watch(searchKeyword, () => {
  handleSearch()
})

// 监听视图模式变化
watch(viewMode, () => {
  nextTick(() => {
    // 重新设置选中状态
    const currentSelected = consoleStore.selectedHosts
    treeRef.value?.setCheckedKeys(currentSelected, false)
  })
})

// 加载所有主机
onMounted(async () => {
  await hostStore.loadAllHosts()
  hostStore.startStatusAutoRefresh()
  
  // 初始化时同步已选中的主机到树
  if (consoleStore.selectedHosts.length > 0 && treeRef.value) {
    nextTick(() => {
      treeRef.value?.setCheckedKeys(consoleStore.selectedHosts, false)
    })
  }
})

// 组件卸载时停止自动刷新
onUnmounted(() => {
  hostStore.stopStatusAutoRefresh()
})
</script>

<style scoped>
.host-tree {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
  padding: 16px;
}

.host-tree-header {
  margin-bottom: 12px;
}

.header-controls {
  display: flex;
  align-items: center;
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
  height: calc(100vh - 280px);
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
