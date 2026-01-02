<template>
  <div class="dashboard-container">
    <div class="header">
      <h2 class="title">{{ $t('dashboard.title') }}</h2>
      <div class="system-info" v-if="dashboardData.system_info">
        <t-tag variant="light" theme="primary">v{{ dashboardData.system_info.version }}</t-tag>
        <t-tag variant="light" theme="default" class="ml-2">Uptime: {{ formatUptime(dashboardData.system_info.uptime) }}</t-tag>
      </div>
    </div>

    <!-- Stats Cards -->
    <t-row :gutter="[16, 16]" class="stats-row">
      <t-col :xs="12" :sm="6" :md="3" v-for="(stat, key) in statsList" :key="key">
        <t-card class="stat-card" :bordered="false">
          <div class="stat-content">
            <div class="stat-label">{{ $t(`dashboard.stats.${key}`) }}</div>
            <div class="stat-value">{{ stat.value }}</div>
          </div>
          <div class="stat-icon" :style="{ color: stat.color }">
            <component :is="stat.icon" size="24" />
          </div>
        </t-card>
      </t-col>
    </t-row>

    <t-row :gutter="[16, 16]" class="mt-4">
      <!-- Storage Usage -->
      <t-col :xs="12" :md="6">
        <t-card :title="$t('dashboard.storage.title')" :bordered="false">
          <div class="storage-usage">
            <div class="usage-info">
              <span>{{ formatBytes(storageUsed) }} / {{ formatBytes(storageQuota) }}</span>
              <span>{{ storagePercent }}%</span>
            </div>
            <t-progress :percentage="storagePercent" :color="storageColor" />
            <div class="usage-desc mt-2">
              {{ $t('dashboard.storage.desc') }}
            </div>
          </div>
        </t-card>
      </t-col>

      <!-- Recent Activities -->
      <t-col :xs="12" :md="6">
        <t-card :title="$t('dashboard.recent_activities')" :bordered="false">
          <t-list :split="true" v-if="dashboardData.recent_audit_logs && dashboardData.recent_audit_logs.length">
            <t-list-item v-for="log in dashboardData.recent_audit_logs" :key="log.id">
              <t-list-item-meta
                :title="log.action"
                :description="`${log.username} @ ${formatTime(log.created_at)}`"
              >
                <template #avatar>
                  <t-avatar shape="circle" :icon="getActionIcon(log.action)" />
                </template>
              </t-list-item-meta>
              <template #action>
                <t-tag :theme="log.status === 'success' ? 'success' : 'danger'" variant="light-outline">
                  {{ log.status }}
                </t-tag>
              </template>
            </t-list-item>
          </t-list>
          <div v-else class="empty-state">
            {{ $t('common.no_data') }}
          </div>
        </t-card>
      </t-col>
    </t-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { 
  UsergroupIcon, 
  ServerIcon, 
  BookIcon, 
  FileIcon,
  LoginIcon,
  EditIcon,
  AddIcon,
  DeleteIcon,
  SettingIcon
} from 'tdesign-icons-vue-next';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import axios from 'axios';

dayjs.extend(relativeTime);

const authStore = useAuthStore();
const dashboardData = ref<any>({});
const loading = ref(false);

const fetchDashboardData = async () => {
  loading.value = true;
  try {
    const response = await axios.get('/api/v1/dashboard');
    dashboardData.value = response.data;
  } catch (error) {
    console.error('Failed to fetch dashboard data:', error);
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchDashboardData();
});

const statsList = computed(() => {
  const stats = dashboardData.value.stats || {};
  const list: any = {};
  
  if (authStore.canAccessAllTenants) {
    list.tenants = {
      value: stats.total_tenants || 0,
      icon: ServerIcon,
      color: '#0052D9'
    };
  }

  list.users = {
    value: stats.total_users || 0,
    icon: UsergroupIcon,
    color: '#2BA471'
  };
  
  list.knowledge_bases = {
    value: stats.total_knowledge_bases || 0,
    icon: BookIcon,
    color: '#E37318'
  };
  
  list.documents = {
    value: stats.total_documents || 0,
    icon: FileIcon,
    color: '#00A870'
  };

  return list;
});

const storageUsed = computed(() => {
  const stats = dashboardData.value.stats || {};
  return stats.total_storage_used || stats.storage_used || 0;
});

const storageQuota = computed(() => {
  const stats = dashboardData.value.stats || {};
  return stats.total_storage_quota || stats.storage_quota || 10737418240;
});

const storagePercent = computed(() => {
  if (storageQuota.value === 0) return 0;
  return Math.round((storageUsed.value / storageQuota.value) * 100);
});

const storageColor = computed(() => {
  if (storagePercent.value > 90) return '#E34D59';
  if (storagePercent.value > 70) return '#ED7B2F';
  return '#00A870';
});

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const formatTime = (time: string) => {
  return dayjs(time).fromNow();
};

const formatUptime = (uptime: string) => {
  // Simple cleanup of Go's duration string
  return uptime.split('.')[0] + 's';
};

const getActionIcon = (action: string) => {
  const a = action.toLowerCase();
  if (a.includes('login')) return LoginIcon;
  if (a.includes('create') || a.includes('add')) return AddIcon;
  if (a.includes('update') || a.includes('edit')) return EditIcon;
  if (a.includes('delete') || a.includes('remove')) return DeleteIcon;
  return SettingIcon;
};
</script>

<style scoped>
.dashboard-container {
  padding: 24px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.title {
  margin: 0;
  font-size: 24px;
  font-weight: 500;
}

.stat-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 14px;
  color: var(--td-text-color-secondary);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.stat-icon {
  background: var(--td-bg-color-container-hover);
  padding: 12px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.storage-usage {
  padding: 8px 0;
}

.usage-info {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
  font-weight: 500;
}

.usage-desc {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.empty-state {
  padding: 40px;
  text-align: center;
  color: var(--td-text-color-placeholder);
}

.ml-2 {
  margin-left: 8px;
}

.mt-4 {
  margin-top: 16px;
}

.mt-2 {
  margin-top: 8px;
}
</style>
