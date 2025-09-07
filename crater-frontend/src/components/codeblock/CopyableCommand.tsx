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

import { CopyButton } from '../button/copy-button'
import TooltipLink from '../label/tooltip-link'

interface CopyableCommandProps {
  label?: string
  command: string
  isLink?: boolean
  isSensitive?: boolean
}

export function CopyableCommand({
  label,
  command,
  isLink = false,
  isSensitive = false,
}: CopyableCommandProps) {
  return (
    <div className="flex flex-col gap-1.5">
      {label && <span className="text-muted-foreground text-sm font-medium">{label}</span>}
      <div className="bg-muted group relative flex flex-row items-center justify-between rounded-md p-2 dark:bg-slate-800/60">
        <div className="mr-2 flex-1 overflow-hidden">
          {isLink ? (
            <div className="flex items-center">
              <code className="mr-2 flex-1 text-sm break-all">{command}</code>
              <TooltipLink
                to={command.startsWith('http') ? command : `http://${command}`}
                name={<ExternalLink className="h-4 w-4" />}
                tooltip="在新标签页中打开"
              />
            </div>
          ) : (
            <code className="text-sm break-all">{command}</code>
          )}
        </div>
        {!isSensitive && <CopyButton content={command} />}
      </div>
    </div>
  )
}
