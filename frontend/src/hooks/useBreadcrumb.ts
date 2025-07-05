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

import { BreadCrumbItem, globalBreadCrumb } from '@/utils/store'
import { getBreadcrumbByPath } from '@/utils/title'
import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useSetAtom } from 'jotai'
import { useTranslation } from 'react-i18next'

const useBreadcrumb = () => {
  const setBreadcrumb = useSetAtom(globalBreadCrumb)
  const location = useLocation()
  const [detail, setDetail] = useState<BreadCrumbItem[]>([])
  const { t } = useTranslation()

  useEffect(() => {
    const pathParts = location.pathname.split('/').filter(Boolean)
    const titles = getBreadcrumbByPath(pathParts)
    if (titles) {
      let url = ''
      const ans: BreadCrumbItem[] = []
      for (let i = 0; i < titles.length; i++) {
        url += `/${titles[i].path}`
        ans.push({
          title: t(titles[i].title), // 会在渲染时使用翻译
          path: url,
          isEmpty: titles[i].isEmpty,
        })
      }
      if (detail) {
        ans.push(...detail)
      }
      setBreadcrumb(ans.slice(1))
    }
  }, [location, setBreadcrumb, detail, t])

  return setDetail
}

export default useBreadcrumb
