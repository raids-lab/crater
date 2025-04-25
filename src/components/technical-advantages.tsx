"use client";

import { Zap, Shield, Layers, RefreshCw } from "lucide-react";

export function TechnicalAdvantages() {
  return (
    <section id="advantages" className="py-20 px-4 bg-gray-50 dark:bg-gray-900">
      <div className="container mx-auto max-w-6xl">
        <h2 className="text-3xl md:text-4xl font-bold text-center mb-16">
          技术优势
        </h2>

        <div className="grid md:grid-cols-2 gap-8">
          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                <Zap className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <h3 className="text-xl font-semibold">高性能计算架构</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              基于 Kubernetes
              构建的高性能计算架构，支持大规模分布式训练和推理，充分发挥 GPU
              集群的计算潜力。
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  优化的 CUDA 加速计算
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  高效的内存管理机制
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-blue-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  智能化资源调度算法
                </span>
              </li>
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg">
                <Shield className="h-6 w-6 text-cyan-600 dark:text-cyan-400" />
              </div>
              <h3 className="text-xl font-semibold">企业级安全保障</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              提供全面的安全机制，保护您的数据和模型资产，满足企业级安全合规要求。
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  细粒度访问控制
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  数据传输加密
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-cyan-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  审计日志与合规报告
                </span>
              </li>
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                <Layers className="h-6 w-6 text-purple-600 dark:text-purple-400" />
              </div>
              <h3 className="text-xl font-semibold">开源生态集成</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              深度整合主流开源组件，提供统一的用户体验，避免技术碎片化。
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  Volcano 作业调度引擎
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  Fluid 数据加速系统
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-purple-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  Envd 环境管理工具
                </span>
              </li>
            </ul>
          </div>

          <div className="bg-white dark:bg-gray-800 p-8 rounded-xl shadow-sm">
            <div className="flex items-center gap-4 mb-6">
              <div className="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
                <RefreshCw className="h-6 w-6 text-green-600 dark:text-green-400" />
              </div>
              <h3 className="text-xl font-semibold">灵活的扩展能力</h3>
            </div>
            <p className="text-gray-700 dark:text-gray-300 mb-4">
              模块化设计，支持灵活扩展，适应不同规模和场景的需求。
            </p>
            <ul className="space-y-2">
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  插件化架构
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  自定义工作流支持
                </span>
              </li>
              <li className="flex items-start gap-2">
                <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-600 mt-2"></span>
                <span className="text-gray-600 dark:text-gray-400">
                  API 集成能力
                </span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </section>
  );
}
