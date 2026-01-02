<template>
  <div class="admin-container">
    <div class="header">
      <h2>用户管理</h2>
      <t-button @click="handleAdd">新增用户</t-button>
    </div>

    <t-table
      :data="users"
      :columns="columns"
      row-key="id"
      :loading="loading"
      :pagination="pagination"
      @page-change="onPageChange"
    >
      <template #is_active="{ row }">
        <t-tag :theme="row.is_active ? 'success' : 'danger'">
          {{ row.is_active ? '启用' : '停用' }}
        </t-tag>
      </template>
      <template #role="{ row }">
        <t-tag :theme="row.role === 'admin' ? 'primary' : 'default'">
          {{ row.role === 'admin' ? '租户管理员' : '普通用户' }}
        </t-tag>
      </template>
      <template #can_access_all_tenants="{ row }">
        <t-tag :theme="row.can_access_all_tenants ? 'warning' : 'default'">
          {{ row.can_access_all_tenants ? '超级管理员' : '普通用户' }}
        </t-tag>
      </template>
      <template #operation="{ row }">
        <t-link theme="primary" @click="handleEdit(row)">编辑</t-link>
        <t-divider layout="vertical" />
        <t-link :theme="row.is_active ? 'danger' : 'success'" @click="toggleStatus(row)">
          {{ row.is_active ? '停用' : '启用' }}
        </t-link>
        <t-divider layout="vertical" />
        <t-popconfirm content="确认删除该用户吗？" @confirm="deleteUser(row)">
          <t-link theme="danger">删除</t-link>
        </t-popconfirm>
      </template>
    </t-table>

    <!-- 新增/编辑弹窗 -->
    <t-dialog
      v-model:visible="dialogVisible"
      :header="isEdit ? '编辑用户' : '新增用户'"
      :confirm-btn="confirmBtn"
      @confirm="handleConfirm"
    >
      <t-form :data="formData" :rules="formRules" label-align="right" label-width="100px">
        <t-form-item label="用户名" name="username">
          <t-input v-model="formData.username" placeholder="请输入用户名" />
        </t-form-item>
        <t-form-item label="邮箱" name="email">
          <t-input v-model="formData.email" placeholder="请输入邮箱" />
        </t-form-item>
        <t-form-item label="密码" name="password">
          <t-input v-model="formData.password" type="password" :placeholder="isEdit ? '留空表示不修改' : '请输入密码'" />
        </t-form-item>
        <t-form-item label="角色" name="role">
          <t-select v-model="formData.role" placeholder="请选择角色">
            <t-option label="租户管理员" value="admin" />
            <t-option label="普通用户" value="user" />
          </t-select>
        </t-form-item>
        <t-form-item v-if="formData.role === 'user'" label="功能菜单权限" name="menu_config">
          <t-select 
            v-model="formData.menu_config" 
            multiple 
            placeholder="请选择允许访问的功能（管理员不受限制）"
            clearable
          >
            <t-option label="AI对话" value="creatChat" />
            <t-option label="知识库管理" value="knowledge-bases" />
            <t-option label="系统设置" value="settings" />
          </t-select>
          <div style="color: var(--td-text-color-placeholder); font-size: 12px; margin-top: 4px;">
            留空则使用租户默认配置，管理员拥有所有权限
          </div>
        </t-form-item>
        <t-form-item v-if="authStore.canAccessAllTenants" label="所属租户" name="tenant_id">
          <t-select v-model="formData.tenant_id" placeholder="请选择租户">
            <t-option v-for="item in tenants" :key="item.id" :label="item.name" :value="item.id" />
          </t-select>
        </t-form-item>
        <t-form-item v-if="authStore.canAccessAllTenants" label="超级管理员" name="can_access_all_tenants">
          <t-switch v-model="formData.can_access_all_tenants" />
        </t-form-item>
        <t-form-item label="状态" name="is_active">
          <t-switch v-model="formData.is_active" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { get, post, put, del } from '@/utils/request'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const users = ref([])
const tenants = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEdit = ref(false)
const currentUserId = ref('')

const pagination = ref({
  current: 1,
  pageSize: 10,
  total: 0
})

const formData = ref({
  username: '',
  email: '',
  password: '',
  tenant_id: null,
  role: 'user',
  menu_config: [],
  is_active: true,
  can_access_all_tenants: false
})

const formRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { email: true, message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  password: [{ required: !isEdit.value, message: '请输入密码', trigger: 'blur' }],
  tenant_id: [{ required: authStore.canAccessAllTenants, message: '请选择租户', trigger: 'change' }]
}

const columns = computed(() => {
  const cols = [
    { colKey: 'username', title: '用户名' },
    { colKey: 'email', title: '邮箱' },
    { colKey: 'role', title: '角色', cell: 'role' },
    { colKey: 'is_active', title: '状态', cell: 'is_active' },
    { colKey: 'operation', title: '操作', cell: 'operation', width: 200 }
  ]
  
  if (authStore.canAccessAllTenants) {
    cols.splice(2, 0, { colKey: 'tenant_id', title: '租户ID' })
    cols.splice(4, 0, { colKey: 'can_access_all_tenants', title: '超级权限', cell: 'can_access_all_tenants' })
  }
  
  return cols
})

const confirmBtn = computed(() => ({
  content: '确定',
  loading: loading.value
}))

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await get('/api/v1/users', {
      page: pagination.value.current,
      page_size: pagination.value.pageSize
    })
    users.value = res.data.items
    pagination.value.total = res.data.total
  } catch (err) {
    MessagePlugin.error('获取用户列表失败')
  } finally {
    loading.value = false
  }
}

const fetchTenants = async () => {
  try {
    const res = await get('/api/v1/tenants/all')
    console.log('Fetched tenants:', res)
    tenants.value = res.data.items || []
  } catch (err) {
    console.error('Failed to fetch tenants:', err)
    MessagePlugin.error('获取租户列表失败')
  }
}

const onPageChange = (pageInfo) => {
  pagination.value.current = pageInfo.current
  pagination.value.pageSize = pageInfo.pageSize
  fetchUsers()
}

const handleAdd = () => {
  isEdit.value = false
  currentUserId.value = ''
  formData.value = {
    username: '',
    email: '',
    password: '',
    tenant_id: authStore.canAccessAllTenants ? null : authStore.user?.tenant_id,
    role: 'user',
    menu_config: [],
    is_active: true,
    can_access_all_tenants: false
  }
  dialogVisible.value = true
}

const handleEdit = (row) => {
  isEdit.value = true
  currentUserId.value = row.id
  formData.value = {
    username: row.username,
    email: row.email,
    password: '', // 编辑时密码字段留空，不发送到后端
    tenant_id: row.tenant_id,
    role: row.role || 'user',
    menu_config: row.menu_config || [],
    is_active: row.is_active,
    can_access_all_tenants: row.can_access_all_tenants
  }
  dialogVisible.value = true
}

const handleConfirm = async () => {
  loading.value = true
  try {
    if (isEdit.value) {
      // 编辑用户时，如果密码为空则不发送password字段
      const updateData = { ...formData.value }
      if (!updateData.password || updateData.password.trim() === '') {
        delete updateData.password
      }
      await put(`/api/v1/users/${currentUserId.value}`, updateData)
      MessagePlugin.success('更新成功')
    } else {
      await post('/api/v1/users', formData.value)
      MessagePlugin.success('创建成功')
    }
    dialogVisible.value = false
    fetchUsers()
  } catch (err) {
    MessagePlugin.error(isEdit.value ? '更新失败' : '创建失败')
  } finally {
    loading.value = false
  }
}

const toggleStatus = async (row) => {
  try {
    await put(`/api/v1/users/${row.id}/status`, { is_active: !row.is_active })
    MessagePlugin.success('操作成功')
    fetchUsers()
  } catch (err) {
    MessagePlugin.error('操作失败')
  }
}

const deleteUser = async (row) => {
  try {
    await del(`/api/v1/users/${row.id}`)
    MessagePlugin.success('删除成功')
    fetchUsers()
  } catch (err) {
    MessagePlugin.error('删除失败')
  }
}

onMounted(() => {
  fetchUsers()
  if (authStore.canAccessAllTenants) {
    fetchTenants()
  }
})
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
