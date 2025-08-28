<template>
  <div>
    <div class="info-banner">
      <div class="info-banner-icon">
        <t-icon name="info-circle-filled" />
      </div>
      <div class="info-banner-content">
        <div class="info-banner-title">账号调度逻辑</div>
        <div class="info-banner-text">
          优先级越高，权重越大，今日使用次数越少，则被调用的概率越大；同分组下相同优先级的账号优先调用今日使用量最少的账号
        </div>
      </div>
    </div>

    <t-card class="list-card-container" :bordered="false">
      <t-row justify="space-between">
        <div class="left-operation-container">
          <t-button @click="handleCreate"> 创建账号 </t-button>
        </div>
        <div class="search-input">
          <t-input v-model="searchValue" placeholder="搜索账号名称" clearable @enter="handleSearch">
            <template #suffix-icon>
              <search-icon size="16px" />
            </template>
          </t-input>
        </div>
      </t-row>

      <t-table
        :data="data"
        :columns="COLUMNS"
        :row-key="rowKey"
        :hover="true"
        :pagination="pagination"
        :selected-row-keys="selectedRowKeys"
        :loading="dataLoading"
        :header-affixed-top="headerAffixedTop"
        @page-change="handlePageChange"
        @select-change="handleSelectChange"
      >
        <template #platform_type="{ row }">
          <t-tag :theme="getPlatformTypeTheme(row.platform_type)" variant="light">
            {{ getPlatformTypeName(row.platform_type) }}
          </t-tag>
        </template>

        <template #group="{ row }">
          <t-tag v-if="row.group" theme="primary" variant="light">
            {{ row.group.name }}
          </t-tag>
          <t-tag v-else-if="row.group_id === 0" theme="success" variant="light"> 全局共享组 </t-tag>
          <span v-else class="text-placeholder">未分组</span>
        </template>

        <template #is_max="{ row }">
          <t-tag v-if="row.is_max" theme="warning" variant="light"> MAX </t-tag>
          <t-tag v-else theme="default" variant="light"> 普通 </t-tag>
        </template>

        <template #enable_proxy="{ row }">
          <t-tag v-if="row.enable_proxy" theme="success" variant="light"> 是 </t-tag>
          <t-tag v-else theme="default" variant="light"> 否 </t-tag>
        </template>

        <template #priority="{ row }">
          <t-tag theme="primary" variant="light"> {{ row.priority }} </t-tag>
        </template>

        <template #weight="{ row }">
          <t-tag theme="success" variant="light"> {{ row.weight }} </t-tag>
        </template>

        <template #today_usage_count="{ row }">
          <t-tag theme="default" variant="light"> {{ row.today_usage_count || 0 }} </t-tag>
        </template>

        <template #today_total_cost="{ row }">
          <span>${{ (row.today_total_cost || 0).toFixed(4) }}</span>
        </template>

        <template #weekly_count="{ row }">
          <t-tag theme="default" variant="light"> {{ row.weekly_count || 0 }} </t-tag>
        </template>

        <template #weekly_cost="{ row }">
          <span>${{ (row.weekly_cost || 0).toFixed(4) }}</span>
        </template>

        <template #current_status="{ row }">
          <t-tag v-if="row.current_status === 1" theme="success" variant="light"> 正常 </t-tag>
          <t-popconfirm
            v-else-if="row.current_status === 2"
            content="确认启用此账号吗?"
            @confirm="handlerUpdateAccountCurrentStatus(row)"
          >
            <t-tag theme="warning" variant="light" style="cursor: pointer"> 接口异常 </t-tag>
          </t-popconfirm>

          <t-tag v-else theme="danger" variant="light"> 限流中 </t-tag>
        </template>

        <template #active_status="{ row }">
          <t-tag v-if="row.active_status === 1" theme="success" variant="light"> 激活 </t-tag>
          <t-tag v-else theme="danger" variant="light"> 禁用 </t-tag>
        </template>

        <template #rate_limit_end_time="{ row }">
          <span v-if="row.rate_limit_end_time">{{ formatDateTime(row.rate_limit_end_time) }}</span>
          <span v-else class="text-placeholder">未限流</span>
        </template>

        <template #last_used_time="{ row }">
          <span v-if="row.last_used_time">{{ formatDateTime(row.last_used_time) }}</span>
          <span v-else class="text-placeholder">从未使用</span>
        </template>

        <template #op="{ row }">
          <t-space size="2px">
            <t-button variant="text" size="small" theme="warning" @click="handleTest(row)"> 测试 </t-button>
            <t-button variant="text" size="small" theme="primary" @click="handleEdit(row)"> 编辑 </t-button>
            <t-button
              variant="text"
              size="small"
              :theme="row.active_status === 1 ? 'warning' : 'success'"
              @click="handleToggleActiveStatus(row)"
            >
              {{ row.active_status === 1 ? '禁用' : '启用' }}
            </t-button>
            <t-button variant="text" size="small" theme="danger" @click="handleDelete([row])"> 删除 </t-button>
          </t-space>
        </template>
      </t-table>
    </t-card>

    <!-- 创建/编辑账号弹窗 -->
    <t-dialog
      v-model:visible="formVisible"
      :header="editingItem ? '编辑账号' : '创建账号'"
      width="800px"
      placement="center"
      @confirm="handleFormConfirm"
      @cancel="handleFormCancel"
    >
      <t-form ref="formRef" :model="formData" label-align="top" label-width="120px">
        <t-row :gutter="16">
          <t-col :span="6">
            <t-form-item label="账号名称" name="name">
              <t-input v-model="formData.name" placeholder="请输入账号名称" />
            </t-form-item>
          </t-col>
          <t-col :span="6">
            <t-form-item label="平台类型" name="platform_type">
              <t-select v-model="formData.platform_type" placeholder="选择平台类型">
                <t-option value="claude" label="Claude" />
                <t-option value="claude_console" label="Claude Console" />
                <t-option value="openai" label="OpenAI" />
                <t-option value="gemini" label="Gemini" disabled />
              </t-select>
            </t-form-item>
          </t-col>
        </t-row>

        <!-- 非Claude平台才显示请求地址和密钥 -->
        <t-row v-if="formData.platform_type !== 'claude'" :gutter="16">
          <t-col :span="6">
            <t-form-item label="请求地址" name="request_url">
              <t-input
                v-model="formData.request_url"
                :placeholder="formData.platform_type === 'openai' ? 'https://api.openai.com/v1' : '请输入API请求地址'"
              />
            </t-form-item>
          </t-col>
          <t-col :span="6">
            <t-form-item label="密钥" name="secret_key">
              <t-input v-model="formData.secret_key" type="password" placeholder="请输入API密钥" />
            </t-form-item>
          </t-col>
        </t-row>

        <t-row :gutter="16">
          <t-col :span="6">
            <t-form-item label="分组" name="group_id">
              <t-select
                v-model="formData.group_id"
                placeholder="选择分组（可选）"
                filterable
                :loading="groupsLoading"
                clearable
              >
                <t-option :value="0" label="全局共享组" />
                <t-option v-for="group in groups" :key="group.id" :value="group.id" :label="group.name" />
              </t-select>
            </t-form-item>
          </t-col>
          <t-col :span="3">
            <t-form-item label="优先级" name="priority">
              <t-input-number v-model="formData.priority" :min="1" :max="999" placeholder="优先级" />
            </t-form-item>
          </t-col>
          <t-col :span="3">
            <t-form-item label="权重" name="weight">
              <t-input-number v-model="formData.weight" :min="1" :max="999" placeholder="权重" />
            </t-form-item>
          </t-col>
        </t-row>

        <t-row :gutter="16">
          <t-col :span="6">
            <t-form-item label="代理配置" name="enable_proxy">
              <t-switch v-model="formData.enable_proxy" />
            </t-form-item>
          </t-col>
          <t-col :span="6">
            <t-form-item v-if="formData.enable_proxy" label="代理地址" name="proxy_uri">
              <t-input v-model="formData.proxy_uri" placeholder="http://proxy:8080" />
            </t-form-item>
          </t-col>
        </t-row>

        <!-- OpenAI 平台模型映射配置 -->
        <t-row v-if="formData.platform_type === 'openai'" :gutter="16">
          <t-col :span="12">
            <t-form-item label="模型映射" name="model_mapping">
              <t-textarea
                v-model="formData.model_mapping"
                placeholder="请输入模型映射配置，格式：claude-haiku-20250303:gpt-4o-mini,claude-sonnet:gpt-4o"
                :rows="3"
              />
              <template #tips>
                <div class="model-mapping-tips">
                  格式：源模型:目标模型，多个映射用逗号分隔<br />
                  示例：claude-haiku-20250303:gpt-4o-mini,claude-sonnet:gpt-4o <br />
                  注意：映射的模型上下文至少保证64k, 否则在一些任务场景会失败
                </div>
              </template>
            </t-form-item>
          </t-col>
        </t-row>

        <t-row :gutter="16">
          <t-col :span="4">
            <t-form-item label="是否MAX账号" name="is_max">
              <t-switch v-model="formData.is_max" />
            </t-form-item>
          </t-col>
          <t-col :span="4">
            <t-form-item label="激活状态" name="active_status">
              <t-radio-group v-model="formData.active_status">
                <t-radio :value="1">激活</t-radio>
                <t-radio :value="2">禁用</t-radio>
              </t-radio-group>
            </t-form-item>
          </t-col>
          <t-col :span="4">
            <t-form-item label="今日使用次数" name="today_usage_count">
              <t-input-number v-model="formData.today_usage_count" :min="0" placeholder="使用次数" />
            </t-form-item>
          </t-col>
        </t-row>

        <!-- Claude 平台令牌配置 -->
        <template v-if="formData.platform_type === 'claude'">
          <!-- Claude平台显示授权方式选择 -->
          <t-form-item label="授权方式" name="auth_method">
            <t-radio-group v-model="authMethod">
              <t-radio value="manual">手动输入令牌</t-radio>
              <t-radio value="oauth">OAuth授权</t-radio>
            </t-radio-group>
          </t-form-item>

          <!-- 手动输入令牌模式 -->
          <template v-if="authMethod === 'manual'">
            <t-form-item label="访问令牌" name="access_token">
              <t-textarea v-model="formData.access_token" placeholder="请输入Claude访问令牌" :rows="3" />
            </t-form-item>

            <t-form-item label="刷新令牌" name="refresh_token">
              <t-textarea v-model="formData.refresh_token" placeholder="请输入Claude刷新令牌" :rows="3" />
            </t-form-item>
          </template>

          <!-- OAuth授权模式 -->
          <template v-if="authMethod === 'oauth'">
            <t-form-item label="OAuth授权">
              <t-space direction="vertical" style="width: 100%">
                <t-button theme="primary" :loading="oauthLoading" @click="handleGetOAuthURL">
                  {{ oauthURL ? '重新获取授权链接' : '获取授权链接' }}
                </t-button>

                <div v-if="oauthURL" class="oauth-url-container">
                  <t-alert theme="info" message="请复制以下链接到浏览器中进行授权：">
                    <template #operation>
                      <t-button size="small" variant="text" @click="openOAuthURL(oauthURL)"> 新窗口打开链接 </t-button>
                    </template>
                  </t-alert>
                  <t-textarea :value="oauthURL" readonly :rows="3" style="margin-top: 8px" />
                </div>

                <t-input v-model="authCode" placeholder="请输入授权完成后获得的授权码" :disabled="!oauthURL" />

                <t-button
                  theme="success"
                  :disabled="!authCode || !oauthURL"
                  :loading="exchangeLoading"
                  @click="handleExchangeCode"
                >
                  验证授权码并获取令牌
                </t-button>
              </t-space>
            </t-form-item>

            <!-- OAuth获取的令牌显示（只读） -->
            <template v-if="formData.access_token || formData.refresh_token">
              <t-form-item :label="editingItem ? '访问令牌（当前已有）' : '访问令牌（已自动获取）'">
                <t-textarea v-model="formData.access_token" readonly :rows="2" />
              </t-form-item>

              <t-form-item :label="editingItem ? '刷新令牌（当前已有）' : '刷新令牌（已自动获取）'">
                <t-textarea v-model="formData.refresh_token" readonly :rows="2" />
              </t-form-item>
            </template>
          </template>
        </template>
      </t-form>
    </t-dialog>

    <!-- 删除确认弹窗 -->
    <t-dialog
      v-model:visible="deleteVisible"
      header="确认删除"
      @confirm="handleDeleteConfirm"
      @cancel="handleDeleteCancel"
    >
      <p>{{ deleteConfirmText }}</p>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { FormInstanceFunctions, PrimaryTableCol, TableRowData } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';

import type { Account, AccountCreateParams, AccountUpdateParams, OAuthURLResponse } from '@/api/account';
import {
  batchDeleteAccounts,
  batchUpdateAccountActiveStatus,
  createAccount,
  deleteAccount,
  exchangeCode,
  getAccountList,
  getOAuthURL,
  testAccount,
  updateAccount,
  updateAccountActiveStatus,
  updateAccountCurrentStatus,
} from '@/api/account';
import type { Group } from '@/api/group';
import { getAllGroups } from '@/api/group';
import { prefix } from '@/config/global';
import { useSettingStore } from '@/store';

defineOptions({
  name: 'AccountsList',
});

const store = useSettingStore();

// 表格列定义
const COLUMNS: PrimaryTableCol<TableRowData>[] = [
  {
    title: 'ID',
    align: 'left',
    width: 80,
    colKey: 'id',
  },
  {
    title: '账号名称',
    align: 'left',
    width: 160,
    colKey: 'name',
    ellipsis: true,
  },
  {
    title: '平台类型',
    colKey: 'platform_type',
    width: 160,
  },
  {
    title: '分组',
    colKey: 'group',
    width: 180,
  },
  {
    title: '当前状态',
    colKey: 'current_status',
    width: 120,
  },
  {
    title: '激活状态',
    colKey: 'active_status',
    width: 100,
  },
  {
    title: '代理',
    colKey: 'enable_proxy',
    width: 80,
  },
  {
    title: '优先级',
    colKey: 'priority',
    width: 80,
  },
  {
    title: '权重',
    colKey: 'weight',
    width: 80,
  },
  {
    title: '类型',
    colKey: 'is_max',
    width: 80,
  },
  {
    title: '今日使用',
    colKey: 'today_usage_count',
    width: 100,
  },
  {
    title: '今日费用',
    colKey: 'today_total_cost',
    width: 100,
  },
  {
    title: '本周使用',
    colKey: 'weekly_count',
    width: 100,
  },
  {
    title: '本周费用',
    colKey: 'weekly_cost',
    width: 100,
  },
  {
    title: '限流结束时间',
    colKey: 'rate_limit_end_time',
    width: 180,
  },
  {
    title: '最后使用时间',
    colKey: 'last_used_time',
    width: 180,
  },
  {
    title: '创建时间',
    colKey: 'created_at',
    width: 180,
    cell: (h, { row }) => formatDateTime(row.created_at),
  },
  {
    title: '操作',
    align: 'center',
    fixed: 'right',
    width: 200,
    colKey: 'op',
  },
];

// 数据相关
const data = ref<Account[]>([]);
const dataLoading = ref(false);
const selectedRowKeys = ref<(string | number)[]>([]);
const searchValue = ref('');

// 分页
const pagination = ref({
  current: 1,
  pageSize: 20,
  total: 0,
  showJumper: true,
  showSizeChanger: true,
});

// 分组数据
const groups = ref<Group[]>([]);
const groupsLoading = ref(false);

// 表单相关
const formVisible = ref(false);
const formRef = ref<FormInstanceFunctions>();
const editingItem = ref<Account | null>(null);
const formData = reactive<AccountCreateParams & AccountUpdateParams>({
  name: '',
  platform_type: 'claude',
  request_url: '',
  secret_key: '',
  group_id: 0,
  priority: 100,
  weight: 100,
  enable_proxy: false,
  proxy_uri: '',
  model_mapping: '',
  active_status: 1,
  is_max: false,
  access_token: '',
  refresh_token: '',
  expires_at: 0,
  today_usage_count: 0,
});

// 删除相关
const deleteVisible = ref(false);
const deleteItems = ref<Account[]>([]);

// OAuth相关
const authMethod = ref('manual'); // 'manual' | 'oauth'
const oauthURL = ref('');
const oauthState = ref('');
const authCode = ref('');
const oauthLoading = ref(false);
const exchangeLoading = ref(false);
const generateAuthInfo = ref<OAuthURLResponse | null>(null);

// 计算属性
const headerAffixedTop = computed(
  () =>
    ({
      offsetTop: store.isUseTabsRouter ? 48 : 0,
      container: `.${prefix}-layout`,
    }) as any,
);

const rowKey = 'id';

const deleteConfirmText = computed(() => {
  if (deleteItems.value.length === 1) {
    return `确认删除账号 "${deleteItems.value[0].name}" 吗？`;
  }
  return `确认删除选中的 ${deleteItems.value.length} 个账号吗？`;
});

// 工具方法
const getPlatformTypeTheme = (type: string): 'primary' | 'success' | 'warning' | 'danger' | 'default' => {
  const themeMap: Record<string, 'primary' | 'success' | 'warning' | 'danger' | 'default'> = {
    claude: 'primary',
    claude_console: 'success',
    openai: 'warning',
    gemini: 'danger',
  };
  return themeMap[type] || 'default';
};

const getPlatformTypeName = (type: string) => {
  const nameMap: Record<string, string> = {
    claude: 'Claude',
    claude_console: 'Claude Console',
    openai: 'OpenAI',
    gemini: 'Gemini',
  };
  return nameMap[type] || type;
};

const formatDateTime = (dateStr: string): string => {
  if (!dateStr) return '';
  return new Date(dateStr).toLocaleString('zh-CN');
};

// 数据获取
const fetchData = async () => {
  dataLoading.value = true;
  try {
    const params = {
      page: pagination.value.current,
      limit: pagination.value.pageSize,
    };
    const result = await getAccountList(params);
    data.value = result.accounts || [];
    pagination.value = {
      ...pagination.value,
      total: result.total || 0,
    };
  } catch (error) {
    console.error('获取账号列表失败:', error);
    MessagePlugin.error('获取账号列表失败');
  } finally {
    dataLoading.value = false;
  }
};

const fetchGroups = async () => {
  groupsLoading.value = true;
  try {
    const result = await getAllGroups();
    groups.value = result || [];
  } catch (error) {
    console.error('获取分组列表失败:', error);
  } finally {
    groupsLoading.value = false;
  }
};

const handleSearch = () => {
  pagination.value.current = 1;
  fetchData();
};

const handlePageChange = (pageInfo: any) => {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  fetchData();
};

const handleSelectChange = (value: (string | number)[]) => {
  selectedRowKeys.value = value;
};

// 测试账号
const handleTest = async (row: Account) => {
  try {
    const res = await testAccount(row.id);
    if (res.success) {
      MessagePlugin.success(res.message);
    } else {
      MessagePlugin.error(res.message);
    }
  } catch (error) {
    console.error('测试账号失败:', error);
    MessagePlugin.error('测试账号失败');
  }
};

// 操作相关方法
const handleCreate = () => {
  editingItem.value = null;
  Object.assign(formData, {
    name: '',
    platform_type: 'claude',
    request_url: '',
    secret_key: '',
    group_id: 0,
    priority: 100,
    weight: 100,
    enable_proxy: false,
    proxy_uri: '',
    model_mapping: '',
    active_status: 1,
    is_max: false,
    access_token: '',
    refresh_token: '',
    today_usage_count: 0,
  });

  // 重置OAuth相关状态
  authMethod.value = 'manual';
  oauthURL.value = '';
  oauthState.value = '';
  authCode.value = '';

  formVisible.value = true;
};

const handleEdit = (item: Account) => {
  editingItem.value = item;
  Object.assign(formData, {
    name: item.name,
    platform_type: item.platform_type,
    request_url: item.request_url || '',
    secret_key: item.secret_key || '', // 现在回填密钥
    group_id: item.group_id || 0,
    priority: item.priority,
    weight: item.weight,
    enable_proxy: item.enable_proxy,
    proxy_uri: item.proxy_uri || '',
    model_mapping: item.model_mapping || '',
    active_status: item.active_status,
    is_max: item.is_max,
    access_token: item.access_token || '', // 现在回填访问令牌
    refresh_token: item.refresh_token || '', // 现在回填刷新令牌
    expires_at: item.expires_at || 0,
    today_usage_count: item.today_usage_count,
  });

  // 根据是否有令牌数据智能设置授权方式
  if (item.access_token && item.refresh_token && item.platform_type === 'claude') {
    // 如果是Claude平台且有令牌，默认选择OAuth模式以显示现有令牌
    authMethod.value = 'oauth';
  } else {
    // 否则默认手动模式
    authMethod.value = 'manual';
  }

  // 重置OAuth相关状态
  oauthURL.value = '';
  oauthState.value = '';
  authCode.value = '';

  formVisible.value = true;
};

const handleFormConfirm = async () => {
  const valid = await formRef.value?.validate();
  if (!valid) return;

  try {
    if (editingItem.value) {
      // 编辑
      const updateData: AccountUpdateParams = {
        name: formData.name,
        platform_type: formData.platform_type,
        request_url: formData.request_url,
        secret_key: formData.secret_key,
        group_id: formData.group_id,
        priority: formData.priority,
        weight: formData.weight,
        enable_proxy: formData.enable_proxy,
        proxy_uri: formData.proxy_uri,
        model_mapping: formData.model_mapping,
        active_status: formData.active_status,
        is_max: formData.is_max,
        access_token: formData.access_token,
        refresh_token: formData.refresh_token,
        today_usage_count: formData.today_usage_count,
      };
      await updateAccount(editingItem.value.id, updateData);
      MessagePlugin.success('更新成功');
    } else {
      // 创建
      const createData: AccountCreateParams = {
        name: formData.name,
        platform_type: formData.platform_type,
        request_url: formData.request_url,
        secret_key: formData.secret_key,
        group_id: formData.group_id,
        priority: formData.priority,
        weight: formData.weight,
        enable_proxy: formData.enable_proxy,
        proxy_uri: formData.proxy_uri,
        model_mapping: formData.model_mapping,
        active_status: formData.active_status,
        is_max: formData.is_max,
        access_token: formData.access_token,
        refresh_token: formData.refresh_token,
        expires_at: formData.expires_at,
        today_usage_count: formData.today_usage_count,
      };
      await createAccount(createData);
      MessagePlugin.success('创建成功');
    }

    formVisible.value = false;
    await fetchData();
  } catch (error) {
    console.error('操作失败:', error);
    MessagePlugin.error('操作失败');
  }
};

const handleFormCancel = () => {
  formVisible.value = false;
};

const handleToggleActiveStatus = async (item: Account) => {
  try {
    const newStatus = item.active_status === 1 ? 2 : 1;
    await updateAccountActiveStatus(item.id, newStatus);
    MessagePlugin.success(newStatus === 1 ? '已启用' : '已禁用');
    await fetchData();
  } catch (error) {
    console.error('状态更新失败:', error);
    MessagePlugin.error('状态更新失败');
  }
};

const _handleBatchUpdateStatus = async (status: number) => {
  const selectedItems = data.value.filter((item) => selectedRowKeys.value.includes(item.id));
  if (selectedItems.length === 0) {
    MessagePlugin.warning('请先选择要操作的账号');
    return;
  }

  try {
    const ids = selectedItems.map((item) => item.id);
    await batchUpdateAccountActiveStatus(ids, status);
    MessagePlugin.success(status === 1 ? '批量启用成功' : '批量禁用成功');
    selectedRowKeys.value = [];
    await fetchData();
  } catch (error) {
    console.error('批量状态更新失败:', error);
    MessagePlugin.error('批量状态更新失败');
  }
};

// 消除账号接口异常状态
const handlerUpdateAccountCurrentStatus = async (row: Account) => {
  try {
    await updateAccountCurrentStatus(row.id, 1);
    MessagePlugin.success('操作成功');
    await fetchData();
  } catch (error) {
    console.error('操作失败:', error);
    MessagePlugin.error('操作失败');
  }
};

const handleDelete = (items: Account[]) => {
  deleteItems.value = items;
  deleteVisible.value = true;
};

const handleDeleteConfirm = async () => {
  try {
    if (deleteItems.value.length === 1) {
      await deleteAccount(deleteItems.value[0].id);
    } else {
      const ids = deleteItems.value.map((item) => item.id);
      await batchDeleteAccounts(ids);
    }
    MessagePlugin.success('删除成功');
    selectedRowKeys.value = [];
    await fetchData();
  } catch (error) {
    console.error('删除失败:', error);
    MessagePlugin.error('删除失败');
  } finally {
    deleteVisible.value = false;
  }
};

const handleDeleteCancel = () => {
  deleteVisible.value = false;
};

// OAuth相关方法
const handleGetOAuthURL = async () => {
  oauthLoading.value = true;
  try {
    const result = await getOAuthURL();
    oauthURL.value = result.auth_url;
    oauthState.value = result.state;
    authCode.value = ''; // 清空授权码
    // 保存本次返回数据, 用于后续验证授权码
    generateAuthInfo.value = result;
    MessagePlugin.success('授权链接获取成功');
  } catch (error) {
    console.error('获取授权链接失败:', error);
    MessagePlugin.error('获取授权链接失败');
  } finally {
    oauthLoading.value = false;
  }
};

const handleExchangeCode = async () => {
  if (!authCode.value || !oauthState.value) {
    MessagePlugin.warning('请输入授权码');
    return;
  }

  exchangeLoading.value = true;
  try {
    const result = await exchangeCode({
      authorization_code: authCode.value,
      callback_url: generateAuthInfo.value?.auth_url || '',
      proxy_uri: formData.proxy_uri,
      code_verifier: generateAuthInfo.value?.code_verifier || '',
      state: generateAuthInfo?.value.state,
    });

    // 自动回填令牌
    formData.access_token = result.access_token;
    formData.refresh_token = result.refresh_token;
    formData.expires_at = result.expires_at || 0;

    MessagePlugin.success('授权成功，令牌已自动填入');

    // 清空OAuth相关数据
    authCode.value = '';
    oauthURL.value = '';
    oauthState.value = '';
    generateAuthInfo.value = null;
  } catch (error) {
    console.error('授权码验证失败:', error);
    MessagePlugin.error('授权码验证失败');
  } finally {
    exchangeLoading.value = false;
  }
};

// 打开授权链接
const openOAuthURL = async (url: string) => {
  try {
    window.open(url, '_blank');
  } catch (error) {
    console.error('打开链接失败:', error);
    MessagePlugin.error('打开链接失败');
  }
};

// 生命周期
onMounted(async () => {
  await Promise.all([fetchData(), fetchGroups()]);
});
</script>
<style lang="less" scoped>
.list-card-container {
  padding: var(--td-comp-paddingTB-xxl) var(--td-comp-paddingLR-xxl);

  :deep(.t-card__body) {
    padding: 0;
  }
  margin-top: 10px;
}

.left-operation-container {
  display: flex;
  align-items: center;
  margin-bottom: var(--td-comp-margin-xxl);
  gap: var(--td-comp-margin-m);

  .selected-count {
    display: inline-block;
    margin-left: var(--td-comp-margin-l);
    color: var(--td-text-color-secondary);
  }
}

.search-input {
  width: 360px;
}

.text-placeholder {
  color: var(--td-text-color-placeholder);
}

.oauth-url-container {
  width: 100%;

  .t-textarea {
    font-family: monospace;
    font-size: 12px;
  }
}

.info-banner {
  background: linear-gradient(135deg, #e6f3ff 0%, #f0f8ff 100%);
  border: 1px solid #91caff;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 16px;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  position: relative;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    width: 4px;
    height: 100%;
    background: #1890ff;
  }

  .info-banner-icon {
    flex-shrink: 0;
    margin-top: 2px;

    .t-icon {
      font-size: 18px;
      color: #1890ff;
    }
  }
}

.model-mapping-tips {
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
  margin-top: 4px;
}

.info-banner-content {
  flex: 1;
  min-width: 0;
}

.info-banner-title {
  font-size: 14px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 4px;
  line-height: 1.4;
}

.info-banner-text {
  font-size: 13px;
  color: #595959;
  line-height: 1.6;
  margin: 0;
}

:deep(.t-form) {
  width: 720px !important;
}
</style>
