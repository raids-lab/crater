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
import { adminPath } from '@/pages/Admin/path'
import { craterPath } from '@/pages/Portal/path'

export type PathInfo = {
  path: string // router path
  titleKey: string // i18n key for title
  titleNavKey?: string // i18n key for titleNav
  isEmpty?: boolean // if true, will disable the return in breadcrumb
  children?: PathInfo[] // children path info
}

const pathDict: PathInfo[] = [craterPath, adminPath]

/**
 *
 * @param path example: ['portal', 'job', 'ai']
 */
export const getBreadcrumbByPath = (
  path: string[]
): { path: string; title: string; isEmpty?: boolean }[] | null => {
  const result = []
  let currentPath = pathDict
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i])
    if (item) {
      result.push({
        title: item.titleNavKey ?? item.titleKey,
        path: item.path,
        isEmpty: item.isEmpty,
      })
      currentPath = item.children || []
    } else {
      break
    }
  }
  return result
}

export const getTitleByPath = (path: string[]): string => {
  let currentPath = pathDict
  for (let i = 0; i < path.length; i++) {
    const item = currentPath.find((item) => item.path === path[i])
    if (item) {
      if (i === path.length - 1) {
        return item.titleKey
      }
      currentPath = item.children || []
    } else {
      break
    }
  }
  return ''
}
