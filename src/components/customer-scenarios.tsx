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

import type React from "react";
import { School, Building2, Server } from "lucide-react";

export function CustomerScenarios() {
  return (
    <section className="py-20 px-4">
      <div className="container mx-auto max-w-6xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-4">
          适用场景
        </h2>
        <p className="text-center text-gray-600 dark:text-gray-400 mb-16 max-w-3xl mx-auto">
          Crater 为不同类型的组织提供定制化的解决方案，满足各种 AI 计算需求
        </p>

        <div className="grid md:grid-cols-3 gap-8">
          <ScenarioCard
            icon={<School />}
            title="高校科研"
            description="替代传统的 Slurm 集群，管理私有的高性能 GPU 节点，提供更友好的用户体验和更高的资源利用率"
            features={["多用户资源隔离", "科研项目管理", "灵活的权限控制"]}
          />

          <ScenarioCard
            icon={<Building2 />}
            title="企业 AI 团队"
            description="为企业 AI 团队提供统一的开发和生产环境，加速模型从研发到部署的全流程"
            features={["DevOps 集成", "模型版本管理", "CI/CD 流水线"]}
          />

          <ScenarioCard
            icon={<Server />}
            title="云服务提供商"
            description="构建公有云或私有云 AI 平台服务，为客户提供弹性、安全、高效的机器学习基础设施"
            features={["多租户架构", "计量计费系统", "服务级别保障"]}
          />
        </div>
      </div>
    </section>
  );
}

function ScenarioCard({
  icon,
  title,
  description,
  features,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  features: string[];
}) {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm hover:shadow-md transition-shadow p-6">
      <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg inline-block mb-4">
        <div className="text-blue-600 dark:text-blue-400">{icon}</div>
      </div>
      <h3 className="text-xl font-semibold mb-3">{title}</h3>
      <p className="text-gray-600 dark:text-gray-400 mb-4">{description}</p>
      <ul className="space-y-2">
        {features.map((feature, index) => (
          <li key={index} className="flex items-start gap-2">
            <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
            <span className="text-gray-700 dark:text-gray-300">{feature}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
