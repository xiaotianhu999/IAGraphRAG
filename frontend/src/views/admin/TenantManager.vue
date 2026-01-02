<template>
  <div class="admin-container">
    <div class="header">
      <h2>租户管理</h2>
      <t-button @click="addTenant">新增租户</t-button>
    </div>

    <t-table
      :data="tenants"
      :columns="columns"
      row-key="id"
      :loading="loading"
      :pagination="pagination"
      @page-change="onPageChange"
    >
      <template #status="{ row }">
        <t-tag :theme="row.status === 'active' ? 'success' : 'danger'">
          {{ row.status === 'active' ? '启用' : '停用' }}
        </t-tag>
      </template>
      <template #operation="{ row }">
        <t-link theme="primary" @click="editTenant(row)">编辑</t-link>
        <t-divider layout="vertical" />
        <t-link :theme="row.status === 'active' ? 'danger' : 'success'" @click="toggleStatus(row)">
          {{ row.status === 'active' ? '停用' : '启用' }}
        </t-link>
        <t-divider layout="vertical" />
        <t-popconfirm content="确认删除该租户吗？" @confirm="deleteTenant(row)">
          <t-link theme="danger">删除</t-link>
        </t-popconfirm>
      </template>
    </t-table>

    <!-- 新增/编辑弹窗 -->
    <t-dialog
      v-model:visible="showEditDialog"
      :header="isEdit ? '编辑租户' : '新增租户'"
      @confirm="saveTenant"
    >
      <t-form :data="formData" label-align="right" label-width="100px">
        <t-form-item label="租户名称" name="name">
          <t-input v-model="formData.name" placeholder="请输入租户名称" />
        </t-form-item>
        <t-form-item label="业务归属" name="business">
          <t-input v-model="formData.business" placeholder="请输入业务归属" />
        </t-form-item>
        <t-form-item label="存储配额(GB)" name="storage_quota">
          <t-input-number v-model="storageQuotaGB" :min="1" />
        </t-form-item>
        <t-form-item label="功能菜单" name="menu_config">
          <t-checkbox-group v-model="formData.menu_config">
            <t-checkbox value="knowledge-bases">知识库管理</t-checkbox>
            <t-checkbox value="creatChat">AI对话</t-checkbox>
            <t-checkbox value="settings">系统设置</t-checkbox>
            <t-checkbox value="admin/users">用户管理</t-checkbox>
            <t-checkbox value="admin/audit-logs">审计日志</t-checkbox>
          </t-checkbox-group>
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { get, post, put, del } from '@/utils/request'

const tenants = ref([])
const loading = ref(false)
const showEditDialog = ref(false)
const isEdit = ref(false)
const formData = ref({
  id: 0,
  name: '',
  business: '',
  storage_quota: 10737418240,
  menu_config: ['knowledge-bases', 'creatChat', 'settings'],
  status: 'active'
})

const addTenant = () => {
  isEdit.value = false
  formData.value = {
    id: 0,
    name: '',
    business: '',
    storage_quota: 10737418240,
    menu_config: ['knowledge-bases', 'creatChat', 'settings'],
    status: 'active'
  }
  showEditDialog.value = true
}

const storageQuotaGB = computed({
  get: () => formData.value.storage_quota / (1024 * 1024 * 1024),
  set: (val) => { formData.value.storage_quota = val * 1024 * 1024 * 1024 }
})

const pagination = ref({
  current: 1,
  pageSize: 10,
  total: 0
})

const columns = [
  { colKey: 'id', title: 'ID', width: 100 },
  { colKey: 'name', title: '租户名称' },
  { colKey: 'business', title: '业务归属' },
  { colKey: 'status', title: '状态', cell: 'status' },
  { colKey: 'operation', title: '操作', cell: 'operation', width: 200 }
]

const fetchTenants = async () => {
  loading.value = true
  try {
    const res = await get('/api/v1/tenants/search', {
      page: pagination.value.current,
      page_size: pagination.value.pageSize
    })
    tenants.value = res.data.items
    pagination.value.total = res.data.total
  } catch (err) {
    MessagePlugin.error('获取租户列表失败')
  } finally {
    loading.value = false
  }
}

const onPageChange = (pageInfo) => {
  pagination.value.current = pageInfo.current
  pagination.value.pageSize = pageInfo.pageSize
  fetchTenants()
}

const editTenant = (row) => {
  isEdit.value = true
  formData.value = { 
    ...row,
    menu_config: row.menu_config || []
  }
  showEditDialog.value = true
}

const toggleStatus = async (row) => {
  const newStatus = row.status === 'active' ? 'disabled' : 'active'
  try {
    await put(`/api/v1/tenants/${row.id}`, { ...row, status: newStatus })
    MessagePlugin.success('操作成功')
    fetchTenants()
  } catch (err) {
    MessagePlugin.error('操作失败')
  }
}

const deleteTenant = async (row) => {
  try {
    await del(`/api/v1/tenants/${row.id}`)
    MessagePlugin.success('删除成功')
    fetchTenants()
  } catch (err) {
    MessagePlugin.error('删除失败')
  }
}

const saveTenant = async () => {
  try {
    if (isEdit.value) {
      await put(`/api/v1/tenants/${formData.value.id}`, formData.value)
    } else {
      await post('/api/v1/tenants', formData.value)
    }
    MessagePlugin.success('保存成功')
    showEditDialog.value = false
    fetchTenants()
  } catch (err) {
    MessagePlugin.error('保存失败')
  }
}

onMounted(fetchTenants)
</script>

<style scoped>
.admin-container {
  padding: 24px;
  background: #fff;
  height: 100%;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
</style>
