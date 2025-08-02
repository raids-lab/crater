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
import { ExternalLink } from 'lucide-react'
import { useMemo } from 'react'

import ListedButton, { SplitButtonItem } from '../../button/listed-button'

interface PrefixLinkButtonProps {
  names: string[]
  prefixes: string[]
  title?: string
}

const PrefixLinkButton = ({ names = [], prefixes = [], title = ' ' }: PrefixLinkButtonProps) => {
  // 用 useMemo 优化过滤和 items 构建
  const filtered = useMemo(
    () =>
      Array.isArray(names) && Array.isArray(prefixes)
        ? names
            .map((name, idx) => ({ name, prefix: prefixes[idx] }))
            .filter((item) => item.name !== 'notebook')
        : [],
    [names, prefixes]
  )

  const items: SplitButtonItem[] = useMemo(
    () =>
      filtered.map((item) => ({
        key: item.prefix,
        title: item.name,
        action: () => window.open(item.prefix, '_blank'),
      })),
    [filtered]
  )

  if (!filtered || filtered.length === 0) {
    return null
  }

  return (
    <ListedButton
      icon={<ExternalLink className="size-4" />}
      renderTitle={(title) => `${title}`}
      itemTitle="入口类型"
      items={items}
      cacheKey={title}
    />
  )
}

export default PrefixLinkButton
