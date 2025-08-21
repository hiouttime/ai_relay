/**
 * 环境变量工具函数
 * 优先从系统环境变量读取，如果没有则从 import.meta.env 读取
 */

/**
 * 获取环境变量值
 * @param key 环境变量键名
 * @param defaultValue 默认值
 * @returns 环境变量值
 */
export function getEnv(key: string, defaultValue?: string): string {
  // 优先从系统环境变量读取（仅在 Node.js 环境下可用）
  if (typeof process !== 'undefined' && process.env && process.env[key]) {
    return process.env[key] as string;
  }
  
  // 从 Vite 环境变量读取
  if (import.meta.env && import.meta.env[key]) {
    return import.meta.env[key] as string;
  }
  
  // 返回默认值
  return defaultValue || '';
}

/**
 * 获取 API 相关配置
 */
export const apiConfig = {
  // API 基础 URL
  get apiUrl(): string {
    return getEnv('VITE_API_URL', '');
  },
  
  // API URL 前缀
  get apiUrlPrefix(): string {
    return getEnv('VITE_API_URL_PREFIX', '');
  },
  
  // 是否使用代理
  get isRequestProxy(): boolean {
    return getEnv('VITE_IS_REQUEST_PROXY', 'false') === 'true';
  },
  
  // 应用基础路径
  get baseUrl(): string {
    return getEnv('VITE_BASE_URL', '/');
  },
  
  // 当前环境
  get mode(): string {
    return getEnv('MODE', 'development');
  }
};