<template>
  <div class="chunk-viewer">
    <!-- Header with Statistics -->
    <div class="viewer-header">
      <div class="stats-cards">
        <div class="stat-card">
          <div class="stat-icon">
            <t-icon name="layers" />
          </div>
          <div class="stat-content">
            <div class="stat-label">{{ $t('chunkViewer.totalChunks') || '总分块数' }}</div>
            <div class="stat-value">{{ total }}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">
            <t-icon name="file-text" />
          </div>
          <div class="stat-content">
            <div class="stat-label">{{ $t('chunkViewer.avgLength') || '平均长度' }}</div>
            <div class="stat-value">{{ avgLength }} {{ $t('chunkViewer.chars') || '字符' }}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">
            <t-icon name="image" />
          </div>
          <div class="stat-content">
            <div class="stat-label">{{ $t('chunkViewer.withImages') || '含图片' }}</div>
            <div class="stat-value">{{ chunksWithImages }}</div>
          </div>
        </div>
      </div>
      
      <div class="viewer-actions">
        <t-input
          v-model="searchKeyword"
          :placeholder="$t('chunkViewer.searchPlaceholder') || '搜索分块内容...'"
          clearable
          @change="handleSearch"
        >
          <template #prefix-icon>
            <t-icon name="search" />
          </template>
        </t-input>
      </div>
    </div>

    <!-- Chunk List -->
    <div v-loading="loading" class="chunk-list">
      <t-empty v-if="!loading && chunks.length === 0" :description="$t('chunkViewer.noChunks') || '暂无分块数据'" />
      
      <div v-for="(chunk, index) in chunks" :key="chunk.id" class="chunk-item">
        <div class="chunk-header">
          <div class="chunk-meta">
            <span class="chunk-index">#{{ chunk.chunk_index + 1 }}</span>
            <span class="chunk-id">ID: {{ chunk.id.substring(0, 8) }}...</span>
            <span class="chunk-length">{{ chunk.content?.length || 0 }} {{ $t('chunkViewer.chars') || '字符' }}</span>
            <span v-if="chunk.chunk_type !== 'text'" class="chunk-type">{{ chunk.chunk_type }}</span>
          </div>
          <div class="chunk-actions">
            <t-button
              theme="default"
              variant="text"
              size="small"
              @click="toggleChunk(index)"
            >
              <t-icon :name="expandedChunks[index] ? 'chevron-up' : 'chevron-down'" />
              {{ expandedChunks[index] ? ($t('chunkViewer.collapse') || '收起') : ($t('chunkViewer.expand') || '展开') }}
            </t-button>
            <t-button
              theme="default"
              variant="text"
              size="small"
              @click="copyChunk(chunk)"
            >
              <t-icon name="copy" />
              {{ $t('chunkViewer.copy') || '复制' }}
            </t-button>
          </div>
        </div>

        <div v-show="expandedChunks[index]" class="chunk-body">
          <div class="chunk-content">
            <div class="content-label">{{ $t('chunkViewer.content') || '内容' }}:</div>
            <div class="content-text">{{ chunk.content }}</div>
          </div>

          <div v-if="chunk.image_info" class="chunk-images">
            <div class="content-label">{{ $t('chunkViewer.images') || '图片' }}:</div>
            <div class="image-list">
              <div v-for="(img, imgIndex) in parseImageInfo(chunk.image_info)" :key="imgIndex" class="image-item">
                <img v-if="img.url" :src="img.url" :alt="`Image ${imgIndex + 1}`" @click="previewImage(img.url)" />
                <div class="image-meta">
                  <p v-if="img.caption"><strong>{{ $t('chunkViewer.caption') || '描述' }}:</strong> {{ img.caption }}</p>
                  <p v-if="img.ocr_text"><strong>OCR:</strong> {{ img.ocr_text }}</p>
                </div>
              </div>
            </div>
          </div>

          <div class="chunk-details">
            <div class="detail-item">
              <span class="detail-label">{{ $t('chunkViewer.position') || '位置' }}:</span>
              <span class="detail-value">[{{ chunk.start_at }} - {{ chunk.end_at }}]</span>
            </div>
            <div v-if="chunk.pre_chunk_id" class="detail-item">
              <span class="detail-label">{{ $t('chunkViewer.prevChunk') || '上一块' }}:</span>
              <span class="detail-value chunk-link" @click="jumpToChunk(chunk.pre_chunk_id)">
                {{ chunk.pre_chunk_id.substring(0, 8) }}...
              </span>
            </div>
            <div v-if="chunk.next_chunk_id" class="detail-item">
              <span class="detail-label">{{ $t('chunkViewer.nextChunk') || '下一块' }}:</span>
              <span class="detail-value chunk-link" @click="jumpToChunk(chunk.next_chunk_id)">
                {{ chunk.next_chunk_id.substring(0, 8) }}...
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ $t('chunkViewer.status') || '状态' }}:</span>
              <t-tag :theme="chunk.is_enabled ? 'success' : 'default'" size="small">
                {{ chunk.is_enabled ? ($t('chunkViewer.enabled') || '已启用') : ($t('chunkViewer.disabled') || '已禁用') }}
              </t-tag>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ $t('chunkViewer.createdAt') || '创建时间' }}:</span>
              <span class="detail-value">{{ formatDate(chunk.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > pageSize" class="pagination-wrapper">
      <t-pagination
        v-model="currentPage"
        :total="total"
        :page-size="pageSize"
        :show-jumper="true"
        @change="handlePageChange"
      />
    </div>

    <!-- Image Preview Dialog -->
    <t-dialog
      v-model:visible="imagePreviewVisible"
      :header="$t('chunkViewer.imagePreview') || '图片预览'"
      width="80%"
      :footer="false"
    >
      <div class="image-preview-container">
        <img :src="previewImageUrl" alt="Preview" style="max-width: 100%; max-height: 70vh;" />
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
import { MessagePlugin } from 'tdesign-vue-next';
import { getKnowledgeChunks } from '@/api/knowledge-base';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

interface Props {
  knowledgeId: string;
}

const props = defineProps<Props>();

interface Chunk {
  id: string;
  content: string;
  chunk_index: number;
  chunk_type: string;
  start_at: number;
  end_at: number;
  pre_chunk_id: string;
  next_chunk_id: string;
  is_enabled: boolean;
  image_info: string;
  created_at: string;
}

const chunks = ref<Chunk[]>([]);
const loading = ref(false);
const total = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);
const searchKeyword = ref('');
const expandedChunks = ref<Record<number, boolean>>({});
const imagePreviewVisible = ref(false);
const previewImageUrl = ref('');

// Statistics
const avgLength = computed(() => {
  if (chunks.value.length === 0) return 0;
  const totalLength = chunks.value.reduce((sum, chunk) => sum + (chunk.content?.length || 0), 0);
  return Math.round(totalLength / chunks.value.length);
});

const chunksWithImages = computed(() => {
  return chunks.value.filter(chunk => chunk.image_info && chunk.image_info !== '').length;
});

// Load chunks
const loadChunks = async () => {
  if (!props.knowledgeId) return;
  
  loading.value = true;
  try {
    const response = await getKnowledgeChunks(props.knowledgeId, {
      page: currentPage.value,
      page_size: pageSize.value,
    });
    
    if (response.success) {
      chunks.value = response.data || [];
      total.value = response.total || 0;
      
      // Expand first chunk by default
      if (chunks.value.length > 0 && currentPage.value === 1) {
        expandedChunks.value[0] = true;
      }
    }
  } catch (error: any) {
    MessagePlugin.error(error.message || t('chunkViewer.loadError') || '加载分块失败');
  } finally {
    loading.value = false;
  }
};

// Toggle chunk expansion
const toggleChunk = (index: number) => {
  expandedChunks.value[index] = !expandedChunks.value[index];
};

// Copy chunk content
const copyChunk = async (chunk: Chunk) => {
  try {
    await navigator.clipboard.writeText(chunk.content);
    MessagePlugin.success(t('chunkViewer.copySuccess') || '复制成功');
  } catch (error) {
    MessagePlugin.error(t('chunkViewer.copyError') || '复制失败');
  }
};

// Parse image info
const parseImageInfo = (imageInfo: string) => {
  if (!imageInfo) return [];
  try {
    return JSON.parse(imageInfo);
  } catch {
    return [];
  }
};

// Preview image
const previewImage = (url: string) => {
  previewImageUrl.value = url;
  imagePreviewVisible.value = true;
};

// Jump to chunk (placeholder for future implementation)
const jumpToChunk = (chunkId: string) => {
  MessagePlugin.info(`Jump to chunk: ${chunkId}`);
};

// Handle page change
const handlePageChange = (page: number) => {
  currentPage.value = page;
  expandedChunks.value = {};
  loadChunks();
};

// Handle search
const handleSearch = () => {
  // TODO: Implement search functionality
  MessagePlugin.info(t('chunkViewer.searchNotImplemented') || '搜索功能即将推出');
};

// Format date
const formatDate = (dateString: string) => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return date.toLocaleString();
};

// Watch knowledge ID changes
watch(() => props.knowledgeId, () => {
  currentPage.value = 1;
  expandedChunks.value = {};
  loadChunks();
}, { immediate: true });

onMounted(() => {
  loadChunks();
});
</script>

<style scoped lang="less">
.chunk-viewer {
  padding: 20px;
  background: #f5f5f5;
  min-height: 100%;
}

.viewer-header {
  background: white;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.stats-cards {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.stat-card {
  flex: 1;
  min-width: 200px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 8px;
  color: white;

  &:nth-child(2) {
    background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  }

  &:nth-child(3) {
    background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  }
}

.stat-icon {
  font-size: 32px;
  opacity: 0.9;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 12px;
  opacity: 0.9;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
}

.viewer-actions {
  display: flex;
  gap: 12px;
}

.chunk-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chunk-item {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  }
}

.chunk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #fafafa;
  border-bottom: 1px solid #e5e5e5;
}

.chunk-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.chunk-index {
  font-weight: bold;
  color: #667eea;
  font-size: 16px;
}

.chunk-id,
.chunk-length {
  color: #666;
  font-size: 13px;
}

.chunk-type {
  padding: 2px 8px;
  background: #e0e7ff;
  color: #4f46e5;
  border-radius: 4px;
  font-size: 12px;
}

.chunk-actions {
  display: flex;
  gap: 8px;
}

.chunk-body {
  padding: 20px;
}

.chunk-content {
  margin-bottom: 20px;
}

.content-label {
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.content-text {
  padding: 12px;
  background: #f9f9f9;
  border-radius: 6px;
  white-space: pre-wrap;
  line-height: 1.6;
  color: #333;
  max-height: 300px;
  overflow-y: auto;
}

.chunk-images {
  margin-bottom: 20px;
}

.image-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.image-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 6px;

  img {
    width: 120px;
    height: 120px;
    object-fit: cover;
    border-radius: 4px;
    cursor: pointer;
    transition: transform 0.2s;

    &:hover {
      transform: scale(1.05);
    }
  }
}

.image-meta {
  flex: 1;
  font-size: 13px;
  color: #666;

  p {
    margin: 4px 0;
  }
}

.chunk-details {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 12px;
  padding: 12px;
  background: #f9f9f9;
  border-radius: 6px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-label {
  font-weight: 600;
  color: #666;
  font-size: 13px;
}

.detail-value {
  color: #333;
  font-size: 13px;
}

.chunk-link {
  color: #667eea;
  cursor: pointer;
  text-decoration: underline;

  &:hover {
    color: #764ba2;
  }
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 24px;
  padding: 20px;
  background: white;
  border-radius: 8px;
}

.image-preview-container {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
}
</style>
