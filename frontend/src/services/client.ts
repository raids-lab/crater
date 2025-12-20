import { getDefaultStore } from 'jotai'
import ky, { HTTPError, Options } from 'ky'
import { toast } from 'sonner'

import { ACCESS_TOKEN_KEY, REFRESH_TOKEN_KEY } from '@/utils/store'
import { configAPIBaseAtom } from '@/utils/store/config'
import { showErrorToast } from '@/utils/toast'

import {
  ERROR_BACKEND,
  ERROR_BUSINESSLOFGIC_ERROR,
  ERROR_INVALID_REQUEST,
  ERROR_NOT_SPECIFIED,
  ERROR_TOKEN_EXPIRED,
  ERROR_TOKEN_INVALID,
  ERROR_USER_EMAIL_NOT_VERIFIED,
  ERROR_USER_NOT_ALLOWED,
  ErrorCode,
} from './error_code'
import type { IErrorResponse, IRefresh, IRefreshResponse, IResponse } from './types'

const store = getDefaultStore()
const apiBase = store.get(configAPIBaseAtom)
const baseURL = `${apiBase ?? ''}/api`

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

        // 🔥 【关键修改】将解析后的数据挂载到 error 对象上
        // 这样上层组件通过 error.data 就能拿到后端返回的 { code, msg }
        Object.assign(error, { data: errorResponse })

        // 根据错误码进行不同处理
        switch (errorResponse.code) {
          case ERROR_TOKEN_INVALID:
            break
          case ERROR_BUSINESSLOFGIC_ERROR:
            break
          case ERROR_INVALID_REQUEST:
            showErrorToast(`请求参数有误, ${errorResponse.msg}`)
            break
          case ERROR_USER_NOT_ALLOWED:
            showErrorToast('用户激活成功，但无关联账户，请联系平台管理员')
            break
          case ERROR_USER_EMAIL_NOT_VERIFIED:
            showErrorToast('接收通知需要验证邮箱，请前往个人主页验证')
            break
          case ERROR_BACKEND:
            // 确保后端返回了 msg
            showErrorToast(errorResponse.msg || '后端服务异常')
            break
          case ERROR_NOT_SPECIFIED:
            showErrorToast(error)
            break
          default:
            toast.error(errorMessage, {
              description: errorResponse.msg || errorMessage || '请求失败，请稍后重试',
            })
            break
        }

        throw error
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
    try {
      const errorResponse = await error.response.json<IErrorResponse>()
      return [errorResponse.code ?? ERROR_NOT_SPECIFIED, errorResponse.msg ?? error.message]
    } catch {
      return [ERROR_NOT_SPECIFIED, error.message]
    }
  }
  return [ERROR_NOT_SPECIFIED, (error as Error).message]
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
