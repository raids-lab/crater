/**
 * Copyright 2025 Crater
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { source } from "@/lib/source";
import { NextResponse } from 'next/server';
import { type OramaDocument } from 'fumadocs-core/search/orama-cloud';

// ▼ 新增下面这一行代码
// 这会告诉 Next.js，在执行 `next build` 时，
// 应该运行一次 GET 函数，并将其结果保存为一个静态的 JSON 文件。
export const dynamic = 'force-static';

export function GET() {
  const results: OramaDocument[] = [];

  // 从 source 获取所有语言和对应的页面
  source.getLanguages().forEach(({ language, pages }) => {
    for (const page of pages) {
      results.push({
        id: page.url,
        title: page.data.title,
        url: page.url,
        structured: page.data.structuredData ?? '',
        description: page.data.description,
        extra_data: {
          locale: language,
        }
      });
    }
  });

  return NextResponse.json(results);
}