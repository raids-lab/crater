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

import { cn } from '@/lib/utils'
import { Badge } from '../ui/badge'
import { NodeType } from '@/services/api/cluster'

export const nodeTypes = [
  {
    value: 'hygon',
    label: 'Hygon',
  },
  {
    value: 'shenwei',
    label: 'Shenwei',
  },
  {
    value: 'yitian',
    label: 'Yitian',
  },
]

const getNodeTypeBadge = (
  phase: NodeType
): {
  label: string
  color: string
} => {
  switch (phase) {
    case NodeType.Hygon:
      return {
        label: '海光',
        color: 'text-highlight-purple bg-highlight-purple/10',
      }
    case NodeType.Shenwei:
      return {
        label: '申威',
        color: 'text-highlight-amber bg-highlight-amber/10',
      }
    case NodeType.Yitian:
      return {
        label: '倚天',
        color: 'text-highlight-cyan bg-highlight-cyan/10',
      }
    default:
      return {
        label: 'Intel',
        color: 'text-secondary-foreground bg-secondary',
      }
  }
}

const NodeTypeBadge = ({ nodeType }: { nodeType: NodeType }) => {
  const data = getNodeTypeBadge(nodeType)

  return (
    <Badge className={cn('border-none', data.color)} variant="outline">
      <div className="">{data.label}</div>
    </Badge>
  )
}

export default NodeTypeBadge
