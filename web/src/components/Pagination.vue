<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  currentPage: number
  pageSize: number
  total: number
  pageSizes?: number[]
  layout?: string
  background?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  pageSizes: () => [10, 20, 50, 100],
  layout: 'total, sizes, prev, pager, next, jumper',
  background: true
})

const emit = defineEmits<{
  (e: 'update:currentPage', page: number): void
  (e: 'update:pageSize', size: number): void
  (e: 'change', page: number, size: number): void
}>()

const totalPages = computed(() => {
  return Math.ceil(props.total / props.pageSize)
})

const handleCurrentChange = (page: number) => {
  emit('update:currentPage', page)
  emit('change', page, props.pageSize)
}

const handleSizeChange = (size: number) => {
  emit('update:pageSize', size)
  // 当每页数量改变时，重置到第一页
  emit('update:currentPage', 1)
  emit('change', 1, size)
}
</script>

<template>
  <div class="pagination-wrapper">
    <el-pagination
      :current-page="currentPage"
      :page-size="pageSize"
      :page-sizes="pageSizes"
      :total="total"
      :layout="layout"
      :background="background"
      @current-change="handleCurrentChange"
      @size-change="handleSizeChange"
    />
    <div class="pagination-info">
      共 {{ totalPages }} 页
    </div>
  </div>
</template>

<style scoped>
.pagination-wrapper {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 16px 0;
}

.pagination-info {
  color: var(--muted-text);
  font-size: 14px;
}

/* Element Plus 分页组件主题覆盖 */
:deep(.el-pagination) {
  --el-pagination-bg-color: var(--secondary-bg);
  --el-pagination-text-color: var(--muted-text);
  --el-pagination-button-color: var(--text-color);
  --el-pagination-button-bg-color: var(--secondary-bg);
  --el-pagination-button-disabled-color: var(--subtle-text);
  --el-pagination-button-disabled-bg-color: var(--secondary-bg);
  --el-pagination-hover-color: var(--highlight-color);
}

:deep(.el-pagination.is-background .el-pager li) {
  background-color: var(--secondary-bg);
  border: 1px solid var(--border-color);
}

:deep(.el-pagination.is-background .el-pager li:not(.is-disabled).is-active) {
  background-color: var(--primary-color);
  border-color: var(--primary-color);
}

:deep(.el-pagination.is-background .el-pager li:not(.is-disabled):hover) {
  color: var(--highlight-color);
}

:deep(.el-pagination .el-select .el-input) {
  width: 100px;
}

:deep(.el-pagination .el-input__wrapper) {
  background-color: var(--secondary-bg) !important;
  border-color: var(--border-color) !important;
}
</style>
