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

"use client";

import { FileText, Github, Mail } from "lucide-react";

export function GetStarted() {
  return (
    <section className="py-20 px-4 bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-950">
      <div className="container mx-auto max-w-6xl">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold mb-4">
            立即开始使用 Crater
          </h2>
          <p className="text-gray-600 dark:text-gray-400 max-w-2xl mx-auto">
            通过以下资源快速了解和部署 Crater，开启您的云原生 AI 计算之旅
          </p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          <a
            href="https://docs.crater.ai/get-started"
            className="flex flex-col items-center bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow text-center"
          >
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg mb-4">
              <FileText className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-2">快速入门指南</h3>
            <p className="text-gray-600 dark:text-gray-400">
              详细的文档和教程，帮助您快速上手 Crater 平台
            </p>
          </a>

          <a
            href="https://github.com/yourorg/crater"
            className="flex flex-col items-center bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow text-center"
          >
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg mb-4">
              <Github className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-2">GitHub 仓库</h3>
            <p className="text-gray-600 dark:text-gray-400">
              访问我们的开源代码，参与社区贡献和讨论
            </p>
          </a>

          <a
            href="mailto:sales@crater.ai"
            className="flex flex-col items-center bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow text-center"
          >
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg mb-4">
              <Mail className="h-6 w-6 text-blue-600 dark:text-blue-400" />
            </div>
            <h3 className="text-xl font-semibold mb-2">联系我们</h3>
            <p className="text-gray-600 dark:text-gray-400">
              获取技术支持或了解企业级部署方案
            </p>
          </a>
        </div>
      </div>
    </section>
  );
}
