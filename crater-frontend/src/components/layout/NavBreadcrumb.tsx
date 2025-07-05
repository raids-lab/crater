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

import { Fragment } from 'react'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'
import { useAtomValue } from 'jotai'
import { globalBreadCrumb } from '@/utils/store'
import { Link } from 'react-router-dom'
import { cn } from '@/lib/utils'
import useBreadcrumb from '@/hooks/useBreadcrumb'
import { useTranslation } from 'react-i18next'

export const NavBreadcrumb = ({ className }: { className: string }) => {
  useBreadcrumb()
  const breadcrumb = useAtomValue(globalBreadCrumb)
  const { t } = useTranslation()

  return (
    <Breadcrumb className={className}>
      <BreadcrumbList>
        {breadcrumb.map((item, index) => {
          return (
            <Fragment key={`bread-${index}`}>
              {index !== 0 && <BreadcrumbSeparator key={`bread-separator-${index}`} />}
              {item.isEmpty && (
                <BreadcrumbPage
                  className={cn({
                    'text-muted-foreground': breadcrumb.length > 1,
                  })}
                >
                  {t(item.title)}
                </BreadcrumbPage>
              )}
              {!item.isEmpty && (
                <BreadcrumbItem key={`bread-item-${index}`}>
                  {item.path && index !== breadcrumb.length - 1 ? (
                    <BreadcrumbLink asChild>
                      <Link to={item.path}>{t(item.title)}</Link>
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage>{t(item.title)}</BreadcrumbPage>
                  )}
                </BreadcrumbItem>
              )}
            </Fragment>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
