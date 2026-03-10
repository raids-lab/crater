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

import { createMDX } from "fumadocs-mdx/next";
import createNextIntlPlugin from 'next-intl/plugin';
import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));

function getChartVersion() {
  const fromEnv = process.env.NEXT_PUBLIC_CRATER_CHART_VERSION;
  if (fromEnv && fromEnv !== '<chart-version>') return fromEnv;
  const chartPath = resolve(__dirname, '../charts/crater/Chart.yaml');
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

const withMDX = createMDX();
const withNextIntl = createNextIntlPlugin();

/** @type {import('next').NextConfig} */
const config = {
  reactStrictMode: true,
  basePath: "/crater",
  output: "export",
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  env: {
    NEXT_PUBLIC_CRATER_CHART_VERSION: chartVersion,
  },
};

export default withNextIntl(withMDX(config));
