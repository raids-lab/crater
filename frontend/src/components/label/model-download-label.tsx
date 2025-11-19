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
interface ModelDownloadLabelProps {
  name: string
  source?: 'modelscope' | 'huggingface' | string
  revision?: string | null
  path?: string | null
}

const ModelDownloadLabel = ({ name, revision, path }: ModelDownloadLabelProps) => {
  return (
    <div className="flex max-w-full flex-col gap-0.5 text-left">
      <span className="truncate text-sm font-medium">{name}</span>
      {revision && <span className="text-muted-foreground truncate text-xs">版本: {revision}</span>}
      {path ? (
        <span className="text-muted-foreground truncate font-mono text-xs">{path}</span>
      ) : null}
    </div>
  )
}

export default ModelDownloadLabel
