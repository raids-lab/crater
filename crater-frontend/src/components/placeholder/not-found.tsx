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
import { Link, useRouter } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'

export default function NotFound() {
  const router = useRouter()
  return (
    <div className="h-svh">
      <div className="m-auto flex h-full w-full flex-col items-center justify-center gap-2">
        <h1 className="text-[5rem] leading-tight font-bold">404</h1>
        <span className="font-medium">页面未找到</span>
        <p className="text-muted-foreground text-center">
          你正在寻找的页面 <br />
          不存在或者可能已被移除
        </p>
        <div className="mt-6 flex gap-4 pb-24">
          <Button variant="outline" onClick={() => router.history.back()}>
            返回上一页
          </Button>
          <Button variant="outline" asChild>
            <Link to="/">返回首页</Link>
          </Button>
        </div>
      </div>
    </div>
  )
}
