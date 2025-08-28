import { request } from '@/utils/request';

// API 账号接口
const Api = {
  GetList: '/api/v1/accounts/list',
  Create: '/api/v1/accounts/create',
  Update: '/api/v1/accounts/update',
  Delete: '/api/v1/accounts/delete',
  GetDetail: '/api/v1/accounts/detail',
  UpdateActiveStatus: '/api/v1/accounts/update-active-status',
  UpdateCurrentStatus: '/api/v1/accounts/update-current-status',
  TestAccount: '/api/v1/accounts/test',
  // Claude OAuth 相关
  GetOAuthURL: '/api/v1/oauth/generate-auth-url',
  ExchangeCode: '/api/v1/oauth/exchange-code',
};

// 账号信息
export interface Account {
  id: number;
  name: string;
  platform_type: string; // claude/claude_console/openai/gemini
  request_url: string;
  secret_key: string; // 现在会返回密钥
  access_token: string; // 现在会返回访问令牌
  refresh_token: string; // 现在会返回刷新令牌
  expires_at: number;
  is_max: boolean;
  group_id: number;
  priority: number;
  weight: number;
  today_usage_count: number;
  today_input_tokens: number;
  today_output_tokens: number;
  today_cache_read_input_tokens: number;
  today_cache_creation_input_tokens: number;
  today_total_cost: number;
  enable_proxy: boolean;
  proxy_uri: string;
  model_mapping: string;
  last_used_time: string;
  rate_limit_end_time: string;
  current_status: number; // 1:正常,2:接口异常,3:账号异常/限流
  active_status: number; // 1:激活,2:禁用
  user_id: number;
  created_at: string;
  updated_at: string;
  user: {
    id: number;
    username: string;
  };
  group: {
    id: number;
    name: string;
  } | null;
  // 最近一周统计数据
  weekly_cost: number; // 最近一周使用费用
  weekly_count: number; // 最近一周使用次数
}

// 创建账号参数
export interface AccountCreateParams {
  name: string;
  platform_type: string;
  request_url?: string;
  secret_key?: string;
  group_id?: number;
  priority?: number;
  weight?: number;
  enable_proxy?: boolean;
  proxy_uri?: string;
  model_mapping?: string;
  active_status?: number;
  is_max?: boolean;
  access_token?: string;
  refresh_token?: string;
  expires_at?: number;
  today_usage_count?: number;
}

// 更新账号参数
export interface AccountUpdateParams {
  name: string;
  platform_type: string;
  request_url?: string;
  secret_key?: string;
  group_id?: number;
  priority?: number;
  weight?: number;
  enable_proxy?: boolean;
  proxy_uri?: string;
  model_mapping?: string;
  active_status?: number;
  is_max?: boolean;
  access_token?: string;
  refresh_token?: string;
  expires_at?: number;
  today_usage_count?: number;
}

// 账号列表参数
export interface AccountListParams {
  page?: number;
  limit?: number;
  user_id?: number;
}

// 账号列表响应
export interface AccountListResponse {
  accounts: Account[];
  total: number;
  page: number;
  limit: number;
}

// 更新激活状态参数
export interface UpdateAccountActiveStatusParams {
  active_status: number;
}

// 更新当前状态参数
export interface UpdateAccountCurrentStatusParams {
  current_status: number;
}

// 测试账号响应
export interface TestAccountResponse {
  success: boolean;
  message: string;
  status_code: number;
  platform_type: string;
}

// Claude OAuth 相关接口类型

// OAuth授权URL响应
export interface OAuthURLResponse {
  auth_url: string;
  state: string;
  code_challenge: string;
  code_verifier: string;
}

// OAuth授权码验证参数
export interface ExchangeCodeParams {
  authorization_code: string;
  callback_url: string;
  proxy_uri: string;
  code_verifier: string;
  state: string;
}

// OAuth授权码验证响应
export interface ExchangeCodeResponse {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  user_info: {
    id: string;
    email: string;
    name: string;
  };
}

/**
 * 获取账号列表
 */
export function getAccountList(params?: AccountListParams) {
  return request.get<AccountListResponse>({
    url: Api.GetList,
    params,
  });
}

/**
 * 获取账号详情
 */
export function getAccountDetail(id: number) {
  return request.get<Account>({
    url: `${Api.GetDetail}/${id}`,
  });
}

/**
 * 创建账号
 */
export function createAccount(data: AccountCreateParams) {
  return request.post<Account>({
    url: Api.Create,
    data,
  });
}

/**
 * 更新账号
 */
export function updateAccount(id: number, data: AccountUpdateParams) {
  return request.put<Account>({
    url: `${Api.Update}/${id}`,
    data,
  });
}

/**
 * 删除账号
 */
export function deleteAccount(id: number) {
  return request.delete({
    url: `${Api.Delete}/${id}`,
  });
}

/**
 * 批量删除账号
 */
export function batchDeleteAccounts(ids: number[]) {
  return request.delete({
    url: Api.Delete,
    data: { ids },
  });
}

/**
 * 更新账号激活状态
 */
export function updateAccountActiveStatus(id: number, active_status: number) {
  return request.put({
    url: `${Api.UpdateActiveStatus}/${id}`,
    data: { active_status },
  });
}

/**
 * 批量更新账号激活状态
 */
export function batchUpdateAccountActiveStatus(ids: number[], active_status: number) {
  return request.put({
    url: Api.UpdateActiveStatus,
    data: { ids, active_status },
  });
}

/**
 * 更新账号当前状态
 */
export function updateAccountCurrentStatus(id: number, current_status: number) {
  return request.put({
    url: `${Api.UpdateCurrentStatus}/${id}`,
    data: { current_status },
  });
}

/**
 * 批量更新账号当前状态
 */
export function batchUpdateAccountCurrentStatus(ids: number[], current_status: number) {
  return request.put({
    url: Api.UpdateCurrentStatus,
    data: { ids, current_status },
  });
}

// 测试账号
export function testAccount(id: number) {
  return request.post<TestAccountResponse>({
    url: `${Api.TestAccount}/${id}`,
  });
}

// === Claude OAuth 相关接口 ===

/**
 * 获取Claude OAuth授权URL
 */
export function getOAuthURL() {
  return request.get<OAuthURLResponse>({
    url: Api.GetOAuthURL,
  });
}

/**
 * 验证OAuth授权码并获取token
 */
export function exchangeCode(data: ExchangeCodeParams) {
  return request.post<ExchangeCodeResponse>({
    url: Api.ExchangeCode,
    data,
  });
}
