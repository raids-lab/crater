"use client";

import type React from "react";
import {
  Database,
  Code,
  Cpu,
  BarChart3,
  GitBranch,
  Monitor,
} from "lucide-react";
import { cn } from "@/lib/utils";

export function CoreCapabilities() {
  return (
    <section
      id="capabilities"
      className="py-24 px-4 bg-gradient-to-b from-transparent to-gray-50 dark:to-gray-900/30"
    >
      <div className="container mx-auto max-w-6xl">
        <div className="text-center mb-20">
          <h2 className="text-4xl md:text-4xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-gray-900 to-gray-600 dark:from-white dark:to-gray-400">
            核心能力
          </h2>
          <p className="text-lg text-gray-600 dark:text-gray-400 max-w-3xl mx-auto">
            Crater
            提供全面的机器学习平台能力，从数据管理到模型训练，一站式解决您的 AI
            工作流需求
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          <CapabilityCard
            icon={<Database className="text-white" />}
            title="数据管理"
            features={[
              "Fluid 加速的分布式缓存系统",
              "细粒度数据共享机制",
              "智能数据预处理流水线",
            ]}
            color="from-blue-600 to-blue-400"
          />

          <CapabilityCard
            icon={<Code className="text-white" />}
            title="环境搭建"
            features={[
              "Envd 环境模板，无需掌握 Docker",
              "支持 JupyterLab/VSCode 远程开发",
              "环境共享与快速复用",
            ]}
            color="from-cyan-600 to-cyan-400"
          />

          <CapabilityCard
            icon={<Cpu className="text-white" />}
            title="模型训练"
            features={[
              "分布式训练框架支持",
              "实时 GPU 利用率监控",
              "训练任务自动调度",
            ]}
            color="from-purple-600 to-purple-400"
          />

          <CapabilityCard
            icon={<BarChart3 className="text-white" />}
            title="性能监控"
            features={[
              "实时损失曲线可视化",
              "资源使用统计报表",
              "训练进度追踪",
            ]}
            color="from-green-600 to-green-400"
          />

          <CapabilityCard
            icon={<GitBranch className="text-white" />}
            title="版本控制"
            features={["模型版本管理", "实验追踪与比较", "配置历史记录"]}
            color="from-orange-600 to-orange-400"
          />

          <CapabilityCard
            icon={<Monitor className="text-white" />}
            title="模型部署"
            features={["一键模型服务化", "自动扩缩容", "API 管理与监控"]}
            color="from-red-600 to-red-400"
          />
        </div>
      </div>
    </section>
  );
}

function CapabilityCard({
  icon,
  title,
  features,
  color,
}: {
  icon: React.ReactNode;
  title: string;
  features: string[];
  color: string;
}) {
  return (
    <div className="group bg-white dark:bg-gray-800/80 rounded-xl shadow-lg hover:shadow-xl duration-300 overflow-hidden border border-gray-100 dark:border-gray-700 backdrop-blur-sm">
      <div className={cn("h-2 bg-gradient-to-r", color)}></div>
      <div className="p-8">
        <div className="flex items-center gap-4 mb-6">
          <div
            className={cn(
              "p-3 rounded-xl bg-gradient-to-br shadow-sm",
              color,
              "duration-300"
            )}
          >
            {icon}
          </div>
          <h3 className="text-xl font-bold">{title}</h3>
        </div>
        <ul className="space-y-3">
          {features.map((feature, index) => (
            <li key={index} className="flex items-start gap-3">
              <span
                className={cn(
                  "inline-block w-2 h-2 rounded-full bg-gradient-to-r mt-2",
                  color
                )}
              ></span>
              <span className="text-gray-700 dark:text-gray-300">
                {feature}
              </span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
