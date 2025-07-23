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
import { CheckCircle, Share2, Lock } from "lucide-react";

export function WhyChooseSection() {
  return (
    <section id="features" className="py-20 px-4 bg-gray-50 dark:bg-gray-900">
      <div className="container mx-auto max-w-6xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-16">
          为何选择{" "}
          <span className="text-blue-600 dark:text-blue-500">Crater</span>
        </h2>

        <div className="grid md:grid-cols-3 gap-8">
          <FeatureCard
            icon={<CheckCircle className="h-8 w-8 text-blue-600" />}
            title="开箱即用的深度学习平台"
            description="无需用户掌握容器或 Kubernetes 知识，提供直观易用的界面，降低使用门槛"
          />

          <FeatureCard
            icon={<Lock className="h-8 w-8 text-blue-600" />}
            title="开源增强，避免厂商锁定"
            description="深度集成 Volcano/Fluid/Envd 等开源项目，兼容 K8s 生态，确保技术自主可控"
          />

          <FeatureCard
            icon={<Share2 className="h-8 w-8 text-blue-600" />}
            title="智能算力共享，优化成本"
            description="通过干扰感知的智能共享调度策略，在用户无感知的情况下，GPU 资源利用率提升 12%"
          />
        </div>
      </div>
    </section>
  );
}

function FeatureCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow-sm hover:shadow-md transition-shadow">
      <div className="mb-4">{icon}</div>
      <h3 className="text-xl font-semibold mb-3">{title}</h3>
      <p className="text-gray-600 dark:text-gray-300">{description}</p>
    </div>
  );
}
