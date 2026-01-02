<template>
  <div class="init-wizard-container">
    <div class="wizard-card">
      <div class="wizard-header">
        <img src="@/assets/img/logo.png" alt="Logo" class="logo" />
        <h1 class="title">{{ $t('init.title') }}</h1>
        <p class="subtitle">{{ $t('init.subtitle') }}</p>
      </div>

      <t-steps :current="currentStep" class="mb-4">
        <t-step-item :title="$t('init.steps.admin')" />
        <t-step-item :title="$t('init.steps.tenant')" />
        <t-step-item :title="$t('init.steps.confirm')" />
      </t-steps>

      <div class="step-content">
        <!-- Step 1: Admin Account -->
        <div v-if="currentStep === 0">
          <t-form :data="formData" :rules="rules" @submit="nextStep">
            <t-form-item :label="$t('init.form.username')" name="admin_username">
              <t-input v-model="formData.admin_username" :placeholder="$t('init.placeholder.username')" />
            </t-form-item>
            <t-form-item :label="$t('init.form.email')" name="admin_email">
              <t-input v-model="formData.admin_email" :placeholder="$t('init.placeholder.email')" />
            </t-form-item>
            <t-form-item :label="$t('init.form.password')" name="admin_password">
              <t-input v-model="formData.admin_password" type="password" :placeholder="$t('init.placeholder.password')" />
            </t-form-item>
            <t-form-item>
              <t-button theme="primary" type="submit" block>{{ $t('common.next') }}</t-button>
            </t-form-item>
          </t-form>
        </div>

        <!-- Step 2: Tenant Info -->
        <div v-if="currentStep === 1">
          <t-form :data="formData" :rules="rules" @submit="nextStep">
            <t-form-item :label="$t('init.form.tenant_name')" name="tenant_name">
              <t-input v-model="formData.tenant_name" :placeholder="$t('init.placeholder.tenant_name')" />
            </t-form-item>
            <t-form-item class="mt-4">
              <div class="button-group">
                <t-button variant="outline" @click="prevStep">{{ $t('initialization.previous') }}</t-button>
                <t-button theme="primary" type="submit">{{ $t('initialization.next') }}</t-button>
              </div>
            </t-form-item>
          </t-form>
        </div>

        <!-- Step 3: Confirm -->
        <div v-if="currentStep === 2">
          <div class="confirm-info">
            <t-descriptions :title="$t('init.confirm.title')" bordered>
              <t-descriptions-item :label="$t('init.form.username')">{{ formData.admin_username }}</t-descriptions-item>
              <t-descriptions-item :label="$t('init.form.email')">{{ formData.admin_email }}</t-descriptions-item>
              <t-descriptions-item :label="$t('init.form.tenant_name')">{{ formData.tenant_name }}</t-descriptions-item>
            </t-descriptions>
          </div>
          <div class="button-group mt-4">
            <t-button variant="outline" @click="prevStep" :disabled="loading">{{ $t('initialization.previous') }}</t-button>
            <t-button theme="primary" @click="handleInitialize" :loading="loading">{{ $t('init.submit') }}</t-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue';
import { useRouter } from 'vue-router';
import { MessagePlugin } from 'tdesign-vue-next';
import axios from 'axios';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const router = useRouter();
const currentStep = ref(0);
const loading = ref(false);

const formData = reactive({
  admin_username: 'admin',
  admin_email: '',
  admin_password: '',
  tenant_name: 'Default Tenant'
});

const rules = {
  admin_username: [{ required: true, message: t('init.rules.username'), trigger: 'blur' }],
  admin_email: [
    { required: true, message: t('init.rules.email'), trigger: 'blur' },
    { email: true, message: t('init.rules.email_format'), trigger: 'blur' }
  ],
  admin_password: [
    { required: true, message: t('init.rules.password'), trigger: 'blur' },
    { min: 8, message: t('init.rules.password_len'), trigger: 'blur' }
  ],
  tenant_name: [{ required: true, message: t('init.rules.tenant_name'), trigger: 'blur' }]
};

const nextStep = (context: any) => {
  if (context.validateResult === true) {
    currentStep.value++;
  }
};

const prevStep = () => {
  currentStep.value--;
};

const handleInitialize = async () => {
  loading.value = true;
  try {
    await axios.post('/api/v1/system/initialize', formData);
    MessagePlugin.success(t('init.success'));
    router.push('/login');
  } catch (error: any) {
    MessagePlugin.error(error.response?.data?.error || t('init.failed'));
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.init-wizard-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--td-bg-color-page);
}

.wizard-card {
  width: 600px;
  padding: 40px;
  background: var(--td-bg-color-container);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-2);
}

.wizard-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  height: 48px;
  margin-bottom: 16px;
}

.title {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.subtitle {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.step-content {
  margin-top: 32px;
}

.button-group {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.button-group .t-button {
  flex: 1;
}

.mb-4 {
  margin-bottom: 16px;
}

.mt-4 {
  margin-top: 16px;
}
</style>
