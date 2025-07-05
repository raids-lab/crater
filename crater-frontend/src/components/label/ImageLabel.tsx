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

import { shortenImageName } from '@/utils/formatter'
import { Badge } from '@/components/ui/badge'
const ImageLabel = ({
  description,
  url,
  tags = [],
}: {
  description: string
  url: string
  tags: string[]
}) => {
  return (
    <div className="flex flex-col gap-0.5 text-left">
      <span className="truncate text-sm font-normal">{description}</span>
      <span className="text-muted-foreground truncate font-mono text-xs">
        {shortenImageName(url)}
      </span>
      <span className="text-muted-foreground truncate font-mono text-xs">
        <div className="flex max-w-[500px] flex-wrap gap-1">
          {tags != null &&
            tags.length !== 0 &&
            tags.map((tag) => (
              <Badge variant="secondary" key={tag} className="rounded-full">
                {tag}
              </Badge>
            ))}
        </div>
      </span>
    </div>
  )
}

export default ImageLabel
