<template>
  <div class="admin-container">
    <div class="header">
      <h2>审计日志</h2>
    </div>

    <t-form class="filter-form" layout="inline">
      <t-form-item v-if="authStore.canAccessAllTenants" label="租户ID" name="tenant_id">
        <t-input v-model="filter.tenant_id" placeholder="请输入租户ID" clearable />
      </t-form-item>
      <t-form-item>
        <t-button @click="fetchLogs">查询</t-button>
      </t-form-item>
    </t-form>

    <t-table
      :data="logs"
      :columns="columns"
      row-key="id"
      :loading="loading"
      :pagination="pagination"
      @page-change="onPageChange"
    >
      <template #status="{ row }">
        <t-tag :theme="row.status === 'success' ? 'success' : 'danger'">
          {{ row.status === 'success' ? '成功' : '失败' }}
        </t-tag>
      </template>
      <template #created_at="{ row }">
        {{ formatDate(row.created_at) }}
      </template>
    </t-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { get } from '@/utils/request'
import dayjs from 'dayjs'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const logs = ref([])
const loading = ref(false)
const filter = reactive({
  tenant_id: ''
})

const pagination = ref({
  current: 1,
  pageSize: 10,
  total: 0
})

const columns = computed(() => {
  const cols = [
    { colKey: 'id', title: 'ID', width: 80 },
    { colKey: 'username', title: '用户名', width: 120 },
    { colKey: 'action', title: '操作', width: 100 },
    { colKey: 'resource', title: '资源', width: 120 },
    { colKey: 'status', title: '状态', cell: 'status', width: 100 },
    { colKey: 'ip', title: 'IP地址', width: 140 },
    { colKey: 'created_at', title: '操作时间', cell: 'created_at', width: 180 },
    { colKey: 'details', title: '详情' }
  ]

  if (authStore.canAccessAllTenants) {
    cols.splice(2, 0, { colKey: 'tenant_id', title: '租户ID', width: 100 })
  }

  return cols
})

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await get('/api/v1/audit-logs', {
      page: pagination.value.current,
      page_size: pagination.value.pageSize,
      tenant_id: filter.tenant_id
    })
    logs.value = res.data.items
    pagination.value.total = res.data.total
  } catch (err) {
    MessagePlugin.error('获取审计日志失败')
  } finally {
    loading.value = false
  }
}

const onPageChange = (pageInfo) => {
  pagination.value.current = pageInfo.current
  pagination.value.pageSize = pageInfo.pageSize
  fetchLogs()
}

const formatDate = (date) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(fetchLogs)
</script>

<style scoped>
.admin-container {
  padding: 24px;
  background: #fff;
  height: 100%;
}
.header {
  margin-bottom: 24px;
}
.filter-form {
  margin-bottom: 24px;
}
</style>
