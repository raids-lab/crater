import { getDefaultStore } from 'jotai'
import ky, { HTTPError, Options } from 'ky'
import { toast } from 'sonner'

import { logger } from '@/utils/loglevel'
import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from '@/utils/store'
import { configAPIBaseAtom } from '@/utils/store/config'
import { showErrorToast } from '@/utils/toast'

import {
  CONFLICT_ERROR_GROUP,
  ERROR_INVALID_CREDENTIALS,
  ERROR_INVALID_REQUEST,
  ERROR_PARAMETER_ERROR,
  ERROR_PERMISSION_DENIED,
  ERROR_REGISTER_NOT_FOUND,
  ERROR_SERVICE_ERROR,
  ERROR_TOKEN_EXPIRED,
  ERROR_TOKEN_INVALID,
  ERROR_USER_EMAIL_NOT_VERIFIED,
  ErrorCode,
} from './error_code'
import {
  LEGACY_ERROR_INVALID_CREDENTIALS,
  LEGACY_ERROR_LDAP_USER_NOT_FOUND,
  LEGACY_ERROR_LEGACY_TOKEN_NOT_SUPPORTED,
  LEGACY_ERROR_NOT_SPECIFIED,
  LEGACY_ERROR_USER_NOT_ALLOWED,
} from './error_code_legacy'
import type { IErrorResponse, IRefresh, IRefreshResponse, IResponse } from './types'

const store = getDefaultStore()
const apiBase = store.get(configAPIBaseAtom)
const baseURL = `${apiBase ?? ''}/api`
const FALLBACK_ERROR_CODE = LEGACY_ERROR_NOT_SPECIFIED

type EnhancedHTTPError = HTTPError & {
  data?: IErrorResponse
  httpStatus?: number
  isHandledByBiz?: boolean
  fallbackLogTimer?: ReturnType<typeof setTimeout>
}

type ErrorPolicyAction = 'silent' | 'toast' | 'delegate'

type ErrorPolicy = {
  action: ErrorPolicyAction
  match: (code: number, error: EnhancedHTTPError) => boolean
  message?: string | ((errorResponse: IErrorResponse, error: EnhancedHTTPError) => string)
}

const getErrorGroup = (code?: number): number => {
  if (!code || Number.isNaN(code)) {
    return 0
  }
  return Math.floor(code / 100)
}

const ERROR_POLICY_TABLE: ErrorPolicy[] = [
  {
    action: 'delegate',
    match: (code) => getErrorGroup(code) === CONFLICT_ERROR_GROUP,
  },
  {
    action: 'silent',
    match: (code) => code === ERROR_TOKEN_INVALID,
  },
  {
    action: 'silent',
    match: (code) =>
      code === ERROR_INVALID_CREDENTIALS || code === LEGACY_ERROR_INVALID_CREDENTIALS,
  },
  {
    action: 'silent',
    match: (code) => code === LEGACY_ERROR_LDAP_USER_NOT_FOUND || code === ERROR_REGISTER_NOT_FOUND,
  },
  {
    action: 'toast',
    match: (code) => code === LEGACY_ERROR_LEGACY_TOKEN_NOT_SUPPORTED,
    message: '不再支持这种登陆方式，直接通过 LDAP 登录即可',
  },
  {
    action: 'toast',
    match: (code) => code === ERROR_INVALID_REQUEST || code === ERROR_PARAMETER_ERROR,
    message: (errorResponse) => `请求参数有误, ${errorResponse.msg}`,
  },
  {
    action: 'toast',
    match: (code) => code === ERROR_PERMISSION_DENIED,
    message: (errorResponse) => errorResponse.msg || '没有权限执行此操作',
  },
  {
    action: 'toast',
    match: (code) => code === LEGACY_ERROR_USER_NOT_ALLOWED,
    message: '用户激活成功，但无关联账户，请联系平台管理员',
  },
  {
    action: 'toast',
    match: (code) => code === ERROR_USER_EMAIL_NOT_VERIFIED,
    message: '接收通知需要验证邮箱，请前往个人主页验证',
  },
  {
    action: 'toast',
    match: (code) => code === ERROR_SERVICE_ERROR,
    message: (errorResponse) => errorResponse.msg || '后端服务异常',
  },
  {
    action: 'toast',
    match: (code) => code === FALLBACK_ERROR_CODE,
    message: (errorResponse) => errorResponse.msg || '请求失败，请稍后重试',
  },
]

const scheduleUnhandledConflictFallback = (error: EnhancedHTTPError) => {
  if (error.fallbackLogTimer) {
    return
  }

  error.fallbackLogTimer = setTimeout(() => {
    if (error.isHandledByBiz) {
      return
    }

    const businessCode = error.data?.code ?? FALLBACK_ERROR_CODE
    const httpStatus = error.httpStatus ?? error.response.status
    const message = error.data?.msg ?? error.message
    showErrorToast(error)
    logger.error(`[UnhandledConflict] http=${httpStatus} code=${businessCode} msg=${message}`)
  }, 0)
}

const resolveErrorPolicy = (code: number, error: EnhancedHTTPError): ErrorPolicy | undefined =>
  ERROR_POLICY_TABLE.find((policy) => policy.match(code, error))

const applyErrorPolicy = (
  policy: ErrorPolicy,
  errorResponse: IErrorResponse,
  error: EnhancedHTTPError
) => {
  switch (policy.action) {
    case 'delegate':
      scheduleUnhandledConflictFallback(error)
      return
    case 'silent':
      return
    case 'toast': {
      const message =
        typeof policy.message === 'function' ? policy.message(errorResponse, error) : policy.message
      if (message) {
        showErrorToast(message)
      }
      return
    }
  }
}

export const markApiErrorHandled = (error: unknown) => {
  if (!error || typeof error !== 'object') {
    return
  }

  const enhancedError = error as EnhancedHTTPError
  enhancedError.isHandledByBiz = true

  if (enhancedError.fallbackLogTimer) {
    clearTimeout(enhancedError.fallbackLogTimer)
    enhancedError.fallbackLogTimer = undefined
  }
}

// Token 刷新函数
const refreshTokenFn = async (): Promise<string> => {
  const data: IRefresh = {
    refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) || '',
  }

  // 使用基本的 ky 实例避免循环调用
  const basicClient = ky.create({ prefixUrl: baseURL })

  const response = await basicClient
    .post('auth/refresh', { json: data })
    .json<IResponse<IRefreshResponse>>()
  const { accessToken, refreshToken } = response.data
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  return accessToken
}

// 重试队列，避免并发刷新
let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (error: unknown) => void }> = []

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token!)
    }
  })

  failedQueue = []
}

// 创建 ky 实例
export const apiClient = ky.create({
  prefixUrl: baseURL,
  retry: 0,
  timeout: 10000,
  hooks: {
    beforeRequest: [
      (request) => {
        // 添加认证头
        const token = localStorage.getItem(ACCESS_TOKEN_KEY)
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`)
        }
        request.headers.set('Content-Type', 'application/json')
      },
    ],
    afterResponse: [
      async (request, options, response) => {
        // 如果响应成功，直接返回
        if (response.ok) {
          return response
        }

        // 处理错误响应
        if (response.status === 401) {
          let errorData: IErrorResponse
          try {
            errorData = await response.clone().json()
          } catch {
            throw new HTTPError(response, request, options)
          }

          // 处理 token 过期
          if (errorData.code === ERROR_TOKEN_EXPIRED) {
            if (isRefreshing) {
              // 如果正在刷新，将请求加入队列
              return new Promise((resolve, reject) => {
                failedQueue.push({ resolve, reject })
              }).then((token) => {
                request.headers.set('Authorization', `Bearer ${token}`)
                return ky(request)
              })
            }

            isRefreshing = true

            try {
              const newToken = await refreshTokenFn()
              processQueue(null, newToken)

              // 重新发起原始请求
              request.headers.set('Authorization', `Bearer ${newToken}`)
              return ky(request)
            } catch (error) {
              processQueue(error, null)
              // 跳转到登录页
              if (!window.location.href.endsWith('/auth')) {
                window.location.href = '/auth'
                throw error
              }
            } finally {
              isRefreshing = false
            }
          }
        }

        // 对于其他错误，让它继续抛出
        throw new HTTPError(response, request, options)
      },
    ],
  },
})

/**
 * 通用 API 请求方法，处理错误并返回统一的响应格式
 */
export async function apiRequest<T>(
  requestFn: () => Promise<T>,
  errorMessage?: string
): Promise<IResponse<T extends IResponse<infer U> ? U : T>> {
  try {
    const response = await requestFn()
    return response as IResponse<T extends IResponse<infer U> ? U : T>
  } catch (error) {
    // 如果是 HTTPError，尝试返回后端响应内容
    if (error instanceof HTTPError) {
      try {
        const errorResponse = await error.response.json<IErrorResponse>()
        const enhancedError = error as EnhancedHTTPError

        // Mount the parsed error response data to the error object
        // This allows upper-level components to access backend response { code, msg } via error.data
        // Also mount HTTP status code for error display
        Object.assign(enhancedError, {
          data: errorResponse,
          httpStatus: error.response.status,
        })

        const matchedPolicy = resolveErrorPolicy(errorResponse.code, enhancedError)
        if (matchedPolicy) {
          applyErrorPolicy(matchedPolicy, errorResponse, enhancedError)
          throw enhancedError
        }

        toast.error(errorMessage, {
          description: errorResponse.msg || errorMessage || '请求失败，请稍后重试',
        })

        throw enhancedError
      } catch (parseError) {
        // 如果解析 JSON 失败（比如后端返回了 HTML 或者空），抛出原始错误
        if (parseError instanceof HTTPError) {
          throw parseError
        }
        // 如果是上面的 JSON 解析动作本身报错，或者 switch 里的逻辑报错
        // 这里的处理可以保持原样，或者稍微优化
        if (error instanceof HTTPError) {
          toast.error(errorMessage, {
            description: error.message || '请求失败，请稍后重试',
          })
        }
        throw error
      }
    }

    throw error // 抛出原始错误以便上层处理
  }
}

/**
 * 获取错误码的辅助函数
 */
export const getErrorCode = async (error: unknown): Promise<[ErrorCode, string]> => {
  if (error instanceof HTTPError) {
    const errorWithData = error as EnhancedHTTPError
    if (errorWithData.data) {
      return [
        errorWithData.data.code ?? FALLBACK_ERROR_CODE,
        errorWithData.data.msg ?? error.message,
      ]
    }

    try {
      const errorResponse = await error.response.json<IErrorResponse>()
      return [errorResponse.code ?? FALLBACK_ERROR_CODE, errorResponse.msg ?? error.message]
    } catch {
      return [FALLBACK_ERROR_CODE, error.message]
    }
  }
  return [FALLBACK_ERROR_CODE, (error as Error).message]
}

/**
 * GET 请求的辅助函数
 */
export const apiV1Get = <T>(url: string, options?: Options) =>
  apiRequest(() => apiClient.get(`v1/${url}`, options).json<T>())

/**
 * POST 请求的辅助函数
 */
export const apiV1Post = <T>(url: string, json?: unknown) =>
  apiRequest(() => apiClient.post(`v1/${url}`, { json }).json<T>())

/**
 * PUT 请求的辅助函数
 */
export const apiV1Put = <T>(url: string, json?: unknown) =>
  apiRequest(() => apiClient.put(`v1/${url}`, { json }).json<T>())

/**
 * DELETE 请求的辅助函数
 */
export const apiV1Delete = <T>(url: string, json?: unknown) =>
  apiRequest(() => apiClient.delete(`v1/${url}`, { json }).json<T>())

export const apiGet = <T>(url: string, options?: Options) =>
  apiRequest(() => apiClient.get(url, options).json<T>())

export const apiPost = <T>(url: string, json?: unknown) =>
  apiRequest(() => apiClient.post(url, { json }).json<T>())

export const apiPut = <T>(url: string, json?: unknown) =>
  apiRequest(() => apiClient.put(url, { json }).json<T>())

export const apiDelete = <T>(url: string, json?: unknown) =>
  apiRequest(() => apiClient.delete(url, { json }).json<T>())

/**
 * 支持上传进度的 PUT 请求
 */
export const apiXMLPut = async <T>(
  url: string,
  body: ArrayBuffer | Blob,
  onProgress?: (progressEvent: { loaded: number; total: number }) => void
): Promise<T> => {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    // 获取完整的 URL
    const fullUrl = `${baseURL}/${url}`

    xhr.open('PUT', fullUrl)

    // 设置认证头
    const token = localStorage.getItem(ACCESS_TOKEN_KEY)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }

    // 监听上传进度
    if (onProgress) {
      xhr.upload.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          onProgress({
            loaded: e.loaded,
            total: e.total,
          })
        }
      })
    }

    // 处理响应
    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const response = xhr.responseText ? JSON.parse(xhr.responseText) : null
          resolve(response)
        } catch {
          resolve(null as T)
        }
      } else {
        // 处理错误响应
        try {
          const errorResponse = JSON.parse(xhr.responseText) as IErrorResponse
          reject(new Error(errorResponse.msg || '上传失败'))
        } catch {
          reject(new Error('上传失败，请稍后重试'))
        }
      }
    })

    // 处理网络错误
    xhr.addEventListener('error', () => {
      reject(new Error('网络错误，请检查网络连接'))
    })

    // 处理超时
    xhr.addEventListener('timeout', () => {
      reject(new Error('上传超时，请稍后重试'))
    })

    // 设置超时时间
    xhr.timeout = 30000 // 30秒超时

    // 发送请求
    xhr.send(body)
  })
}

/**
 * 支持下载进度的 GET 请求（使用 XMLHttpRequest）
 */
export const apiXMLDownload = async (
  url: string,
  filename: string,
  onProgress?: (progressEvent: { loaded: number; total: number }) => void
): Promise<void> => {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    // 获取完整的 URL
    const fullUrl = `${baseURL}/${url}`

    xhr.open('GET', fullUrl)
    xhr.responseType = 'blob'

    // 设置认证头
    const token = localStorage.getItem(ACCESS_TOKEN_KEY)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }

    // 监听下载进度
    if (onProgress) {
      xhr.addEventListener('progress', (e) => {
        if (e.lengthComputable) {
          onProgress({
            loaded: e.loaded,
            total: e.total,
          })
        }
      })
    }

    // 处理响应
    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        const blob = xhr.response as Blob

        const a = document.createElement('a')
        a.style.display = 'none'
        a.download = filename
        a.href = URL.createObjectURL(blob)
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        URL.revokeObjectURL(a.href)
        resolve()
      } else {
        // 处理错误响应
        reject(new Error(`下载失败: ${xhr.statusText}`))
      }
    })

    // 处理网络错误
    xhr.addEventListener('error', () => {
      reject(new Error('网络错误，请检查网络连接'))
    })

    // 处理超时
    xhr.addEventListener('timeout', () => {
      reject(new Error('下载超时，请稍后重试'))
    })

    // 设置超时时间
    xhr.timeout = 600000 // 600秒超时（下载可能需要更长时间）

    // 发送请求
    xhr.send()
  })
}
