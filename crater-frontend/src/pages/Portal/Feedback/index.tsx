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

import { SheetIcon } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'

export default function FeedBack() {
  return (
    <div>
      {/* 第二节：跳转至反馈在线表格 https://365.kdocs.cn/l/cuQ2oVGbNVIt */}
      <section>
        <Card>
          <CardHeader className="flex items-center space-x-2">
            <SheetIcon size={24} className="text-primary" />
            <CardTitle>平台反馈在线表格</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col items-center">
            <a
              href="https://365.kdocs.cn/l/cuQ2oVGbNVIt"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary underline"
            >
              点击跳转至反馈表格
            </a>
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
