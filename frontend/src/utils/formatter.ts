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

// crater-harbor.crater.example.io/docker.io/jupyter/base-notebook:ubuntu-22.04 -> base-notebook:ubuntu-22.04
export const shortenImageName = (imageName: string): string => {
  return imageName.split('/').slice(-2).join('/').replace('docker.io/', '')
}

export const shortestImageName = (imageName: string): string => {
  return imageName.split('/').slice(-1).join('/')
}

export function formatBytes(bytes: number, decimals: number = 2): string {
  if (bytes === 0) return '0 B'
  if (bytes < 0) return '-' + formatBytes(-bytes, decimals)

  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const value = bytes / Math.pow(k, i)

  return `${value.toFixed(decimals)} ${sizes[i]}`
}
