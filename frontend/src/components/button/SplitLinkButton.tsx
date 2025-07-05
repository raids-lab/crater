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

import { PlusCircleIcon } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import SplitButton, { SplitButtonItem } from './SplitButton'

interface URL {
  url: string
  name: string
  disabled?: boolean
}

const SplitLinkButton = ({ urls, title }: { urls: URL[]; title: string }) => {
  const navigate = useNavigate()

  const items: SplitButtonItem[] = urls.map((url) => ({
    key: url.url,
    title: url.name,
    action: () => navigate(`/${url.url}`),
    disabled: url.disabled,
  }))

  return (
    <SplitButton
      icon={<PlusCircleIcon className="size-4" />}
      renderTitle={(title) => `新建${title}`}
      itemTitle="作业类型"
      items={items}
      cacheKey={title}
    />
  )
}

export default SplitLinkButton
