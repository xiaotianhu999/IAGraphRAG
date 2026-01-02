<template>
  <div class="general-settings">
    <div class="section-header">
      <h2>{{ $t('general.title') }}</h2>
      <p class="section-description">{{ $t('general.description') }}</p>
    </div>

    <div class="settings-group">
      <!-- 语言选择 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('language.language') }}</label>
          <p class="desc">{{ $t('language.languageDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localLanguage"
            :placeholder="$t('language.selectLanguage')"
            @change="handleLanguageChange"
            style="width: 280px;"
          >
            <t-option value="zh-CN" :label="$t('language.zhCN')">{{ $t('language.zhCN') }}</t-option>
            <t-option value="en-US" :label="$t('language.enUS')">{{ $t('language.enUS') }}</t-option>

          </t-select>
        </div>
      </div>

      <!-- 主题选择 -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('general.theme') }}</label>
          <p class="desc">{{ $t('general.themeDescription') }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localTheme"
            @change="handleThemeChange"
            style="width: 280px;"
          >
            <t-option value="light" :label="$t('general.light')">{{ $t('general.light') }}</t-option>
            <t-option value="dark" :label="$t('general.dark')">{{ $t('general.dark') }}</t-option>
          </t-select>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

const { t, locale } = useI18n()

// 本地状态
const localLanguage = ref('zh-CN')
const localTheme = ref('light')

// 初始化加载
onMounted(() => {
  // 从 localStorage 加载语言设置
  const savedLocale = localStorage.getItem('locale')
  if (savedLocale) {
    localLanguage.value = savedLocale
    locale.value = savedLocale
  } else {
    localLanguage.value = locale.value
  }

  // 从 localStorage 加载主题设置
  const savedSettings = localStorage.getItem('WeKnora_general_settings')
  if (savedSettings) {
    try {
      const settings = JSON.parse(savedSettings)
      if (settings.theme) {
        localTheme.value = settings.theme
      }
    } catch (e) {
      console.error('Failed to parse general settings', e)
    }
  }
})

// 处理语言变化
const handleLanguageChange = () => {
  locale.value = localLanguage.value
  localStorage.setItem('locale', localLanguage.value)
  
  // 同时更新 general_settings 中的语言
  const savedSettings = localStorage.getItem('WeKnora_general_settings')
  let settings = {}
  if (savedSettings) {
    try {
      settings = JSON.parse(savedSettings)
    } catch (e) {}
  }
  settings = { ...settings, language: localLanguage.value }
  localStorage.setItem('WeKnora_general_settings', JSON.stringify(settings))
  
  MessagePlugin.success(t('language.languageSaved'))
}

// 处理主题变化
const handleThemeChange = () => {
  // 应用主题到 DOM
  document.documentElement.setAttribute('theme-mode', localTheme.value)
  
  const savedSettings = localStorage.getItem('WeKnora_general_settings')
  let settings = {}
  if (savedSettings) {
    try {
      settings = JSON.parse(savedSettings)
    } catch (e) {}
  }
  settings = { ...settings, theme: localTheme.value }
  localStorage.setItem('WeKnora_general_settings', JSON.stringify(settings))
  MessagePlugin.success(t('common.success'))
}
</script>

<style lang="less" scoped>
.general-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-border);

  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  flex: 1;
  max-width: 65%;
  padding-right: 24px;

  label {
    font-size: 15px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: 13px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}
</style>

