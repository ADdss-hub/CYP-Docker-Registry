<template>
  <div class="org-container">
    <div class="page-header">
      <div class="header-left">
        <h1>组织管理</h1>
        <p>管理团队组织和成员</p>
      </div>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon>
        创建组织
      </el-button>
    </div>

    <el-card class="org-list">
      <el-table :data="organizations" v-loading="loading" stripe>
        <el-table-column prop="name" label="组织名称" width="200">
          <template #default="{ row }">
            <div class="org-name">
              <el-icon><OfficeBuilding /></el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="display_name" label="显示名称" width="200" />
        <el-table-column prop="member_count" label="成员数" width="100" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" @click="viewMembers(row)">成员</el-button>
            <el-button size="small" @click="editOrg(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="deleteOrg(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create Organization Dialog -->
    <el-dialog v-model="showCreateDialog" title="创建组织" width="500px">
      <el-form :model="createForm" :rules="createRules" ref="createFormRef" label-width="100px">
        <el-form-item label="组织名称" prop="name">
          <el-input v-model="createForm.name" placeholder="唯一标识，如 my-team" />
        </el-form-item>
        <el-form-item label="显示名称" prop="display_name">
          <el-input v-model="createForm.display_name" placeholder="可选，如 我的团队" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createOrg">创建</el-button>
      </template>
    </el-dialog>

    <!-- Edit Organization Dialog -->
    <el-dialog v-model="showEditDialog" title="编辑组织" width="500px">
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="组织名称">
          <el-input v-model="editForm.name" disabled />
        </el-form-item>
        <el-form-item label="显示名称">
          <el-input v-model="editForm.display_name" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" :loading="updating" @click="updateOrg">保存</el-button>
      </template>
    </el-dialog>

    <!-- Members Dialog -->
    <el-dialog v-model="showMembersDialog" :title="`${currentOrg?.display_name || currentOrg?.name} - 成员管理`" width="600px">
      <div class="members-header">
        <el-button type="primary" size="small" @click="showAddMemberDialog = true">添加成员</el-button>
      </div>
      <el-table :data="members" v-loading="loadingMembers">
        <el-table-column prop="username" label="用户名" />
        <el-table-column prop="role" label="角色" width="120">
          <template #default="{ row }">
            <el-tag :type="row.role === 'owner' ? 'warning' : 'info'" size="small">
              {{ row.role === 'owner' ? '所有者' : '成员' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="加入时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button 
              v-if="row.role !== 'owner'" 
              size="small" 
              type="danger" 
              @click="removeMember(row)"
            >
              移除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- Add Member Dialog -->
    <el-dialog v-model="showAddMemberDialog" title="添加成员" width="400px">
      <el-form :model="addMemberForm" label-width="80px">
        <el-form-item label="用户ID">
          <el-input v-model.number="addMemberForm.user_id" type="number" placeholder="输入用户ID" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="addMemberForm.role">
            <el-option label="成员" value="member" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddMemberDialog = false">取消</el-button>
        <el-button type="primary" :loading="addingMember" @click="addMember">添加</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, OfficeBuilding } from '@element-plus/icons-vue'
import request from '@/utils/request'

interface Organization {
  id: number
  name: string
  display_name: string
  owner_id: number
  member_count?: number
  created_at: string
}

interface Member {
  id: number
  user_id: number
  username: string
  role: string
  created_at: string
}

const loading = ref(false)
const organizations = ref<Organization[]>([])

const showCreateDialog = ref(false)
const creating = ref(false)
const createFormRef = ref()
const createForm = reactive({
  name: '',
  display_name: ''
})
const createRules = {
  name: [
    { required: true, message: '请输入组织名称', trigger: 'blur' },
    { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和连字符', trigger: 'blur' }
  ]
}

const showEditDialog = ref(false)
const updating = ref(false)
const editForm = reactive({
  id: 0,
  name: '',
  display_name: ''
})

const showMembersDialog = ref(false)
const loadingMembers = ref(false)
const currentOrg = ref<Organization | null>(null)
const members = ref<Member[]>([])

const showAddMemberDialog = ref(false)
const addingMember = ref(false)
const addMemberForm = reactive({
  user_id: 0,
  role: 'member'
})

onMounted(() => {
  fetchOrganizations()
})

async function fetchOrganizations() {
  loading.value = true
  try {
    const response = await request.get('/api/v1/orgs')
    organizations.value = response.data.organizations || []
  } catch (error) {
    console.error('Failed to fetch organizations:', error)
  } finally {
    loading.value = false
  }
}

async function createOrg() {
  if (!createFormRef.value) return
  
  try {
    await createFormRef.value.validate()
  } catch {
    return
  }

  creating.value = true
  try {
    await request.post('/api/v1/orgs', createForm)
    ElMessage.success('组织创建成功')
    showCreateDialog.value = false
    createForm.name = ''
    createForm.display_name = ''
    fetchOrganizations()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '创建失败')
  } finally {
    creating.value = false
  }
}

function editOrg(org: Organization) {
  editForm.id = org.id
  editForm.name = org.name
  editForm.display_name = org.display_name
  showEditDialog.value = true
}

async function updateOrg() {
  updating.value = true
  try {
    await request.put(`/api/v1/orgs/${editForm.id}`, {
      display_name: editForm.display_name
    })
    ElMessage.success('组织更新成功')
    showEditDialog.value = false
    fetchOrganizations()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '更新失败')
  } finally {
    updating.value = false
  }
}

async function deleteOrg(org: Organization) {
  try {
    await ElMessageBox.confirm(
      `确定要删除组织 "${org.display_name || org.name}" 吗？此操作不可恢复。`,
      '删除确认',
      { type: 'warning' }
    )
  } catch {
    return
  }

  try {
    await request.delete(`/api/v1/orgs/${org.id}`)
    ElMessage.success('组织已删除')
    fetchOrganizations()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '删除失败')
  }
}

async function viewMembers(org: Organization) {
  currentOrg.value = org
  showMembersDialog.value = true
  loadingMembers.value = true
  
  try {
    const response = await request.get(`/api/v1/orgs/${org.id}/members`)
    members.value = response.data.members || []
  } catch (error) {
    console.error('Failed to fetch members:', error)
    members.value = []
  } finally {
    loadingMembers.value = false
  }
}

async function addMember() {
  if (!currentOrg.value || !addMemberForm.user_id) {
    ElMessage.warning('请输入用户ID')
    return
  }

  addingMember.value = true
  try {
    await request.post(`/api/v1/orgs/${currentOrg.value.id}/members`, addMemberForm)
    ElMessage.success('成员添加成功')
    showAddMemberDialog.value = false
    addMemberForm.user_id = 0
    addMemberForm.role = 'member'
    viewMembers(currentOrg.value)
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '添加失败')
  } finally {
    addingMember.value = false
  }
}

async function removeMember(member: Member) {
  if (!currentOrg.value) return

  try {
    await ElMessageBox.confirm(
      `确定要移除成员 "${member.username}" 吗？`,
      '移除确认',
      { type: 'warning' }
    )
  } catch {
    return
  }

  try {
    await request.delete(`/api/v1/orgs/${currentOrg.value.id}/members/${member.user_id}`)
    ElMessage.success('成员已移除')
    viewMembers(currentOrg.value)
  } catch (error: any) {
    ElMessage.error(error.response?.data?.error || '移除失败')
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  return new Date(dateStr).toLocaleString('zh-CN')
}
</script>

<style scoped>
.org-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-left h1 {
  color: var(--text-primary, #ffffff);
  font-size: 24px;
  margin: 0 0 8px 0;
}

.header-left p {
  color: var(--text-secondary, rgba(255, 255, 255, 0.6));
  margin: 0;
}

.org-list {
  background: var(--bg-secondary, #1a1f3a);
}

.org-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.org-name .el-icon {
  color: var(--primary, #00d4ff);
}

.members-header {
  margin-bottom: 16px;
}
</style>
