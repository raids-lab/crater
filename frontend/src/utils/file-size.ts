/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// 文件大小限制常量
export const FILE_SIZE_LIMITS = {
  // 1GB = 1024 * 1024 * 1024 bytes
  MAX_FILE_SIZE: 1024 * 1024 * 1024,
  // 可读的文件大小限制文本
  MAX_FILE_SIZE_TEXT: '1GB',
} as const

/**
 * 检查文件大小是否超过限制
 * @param fileSize 文件大小（字节）
 * @returns 是否超过限制
 */
export const isFileSizeExceeded = (fileSize: number): boolean => {
  return fileSize > FILE_SIZE_LIMITS.MAX_FILE_SIZE
}

/**
 * 格式化文件大小为可读格式
 * @param bytes 文件大小（字节）
 * @returns 格式化后的文件大小字符串
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'

  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}
