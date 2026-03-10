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

import { defineDocs, defineConfig } from 'fumadocs-mdx/config';
import { readFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';

function getChartVersion() {
  const fromEnv = process.env.NEXT_PUBLIC_CRATER_CHART_VERSION;
  if (fromEnv && fromEnv !== '<chart-version>') return fromEnv;
  // source.config.ts runs in the website directory
  const chartPath = resolve(process.cwd(), '../charts/crater/Chart.yaml');
  try {
    if (!existsSync(chartPath)) return '<chart-version>';
    const content = readFileSync(chartPath, 'utf8');
    const match = content.match(/^version:\s*["']?([a-zA-Z0-9\.\-\+]+)/m);
    return match ? match[1].trim() : '<chart-version>';
  } catch {
    return '<chart-version>';
  }
}

const chartVersion = getChartVersion();

// Options: https://fumadocs.vercel.app/docs/mdx/collections#define-docs
export const docs = defineDocs({
  dir: 'content/docs',
});

export default defineConfig({
  mdxOptions: {
    remarkPlugins: [
      () => (tree) => {
        const traverse = (node: any) => {
          if (node.value && (node.type === 'code' || node.type === 'inlineCode')) {
            if (node.value.includes('<chart-version>')) {
              node.value = node.value.replaceAll('<chart-version>', chartVersion);
            }
          }
          if (node.children) {
            node.children.forEach(traverse);
          }
        };
        traverse(tree);
      },
    ],
  },
});
