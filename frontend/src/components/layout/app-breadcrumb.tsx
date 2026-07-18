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
import { Link, isMatch, useMatches, useRouter } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import { Fragment, ReactNode, useMemo } from 'react'

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb'

import { atomBreadcrumb } from '@/utils/store'

export interface NavBreadcrumbItem {
  href: string
  label: ReactNode
  back?: boolean
}

export const NavBreadcrumb = ({
  className,
  listClassName,
  items: providedItems,
}: {
  className: string
  listClassName?: string
  items?: NavBreadcrumbItem[]
}) => {
  const customItems = useAtomValue(atomBreadcrumb)
  const matches = useMatches()
  const router = useRouter()

  const items = useMemo(() => {
    if (providedItems) {
      return providedItems
    }
    if (customItems.length > 0) {
      return customItems
    }
    const matchesWithCrumbs = matches.filter((match) => isMatch(match, 'loaderData.crumb'))
    const matchedWithBacks = matches.filter((match) => isMatch(match, 'loaderData.back'))
    return matchesWithCrumbs.map(({ pathname, loaderData }) => {
      return {
        href: pathname,
        label: loaderData?.crumb,
        back: matchedWithBacks.some((match) => match.pathname === pathname),
      }
    })
  }, [matches, customItems, providedItems])

  return (
    <Breadcrumb className={className}>
      <BreadcrumbList className={listClassName}>
        {items.map((item, index) => {
          return (
            <Fragment key={`bread-${index}`}>
              {index !== 0 && <BreadcrumbSeparator key={`bread-separator-${index}`} />}
              <BreadcrumbItem key={`bread-item-${index}`}>
                {index === items.length - 1 ? (
                  <BreadcrumbPage>{item.label}</BreadcrumbPage>
                ) : item.back ? (
                  <BreadcrumbLink onClick={() => router.history.back()} className="cursor-pointer">
                    {item.label}
                  </BreadcrumbLink>
                ) : (
                  <BreadcrumbLink asChild>
                    <Link to={item.href}>{item.label}</Link>
                  </BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </Fragment>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}
