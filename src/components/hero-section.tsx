"use client";

import { ArrowRight, BookOpenIcon } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

export function HeroSection() {
  return (
    <section className="py-20 px-4">
      <div className="container mx-auto my-20 max-w-6xl">
        <div className="flex flex-col items-center text-center">
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-cyan-500">
            Crater · 云原生智算平台
          </h1>
          <p className="text-xl md:text-2xl leading-relaxed text-balance max-w-3xl mb-10 text-gray-700 dark:text-gray-300">
            <span>基于 Kubernetes 的机器学习一站式解决方案</span>
            <br />
            <span>整合开源生态，为 AI 训练与服务提供简单高效的体验</span>
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <a
              href="https://gpu.act.buaa.edu.cn/portal"
              className="flex items-center justify-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg transition-colors"
            >
              开始使用
              <ArrowRight size={18} />
            </a>
            <Link
              href="/docs"
              className="flex items-center justify-center gap-2 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 px-6 py-3 rounded-lg transition-colors"
            >
              阅读文档
              <BookOpenIcon size={18} />
            </Link>
          </div>
        </div>

        <div className="mt-16 relative">
          <div className="absolute rounded-2xl inset-0 bg-gradient-to-t via-60% from-white dark:from-gray-950 z-10 top-64 bottom-0"></div>
          <div className="bg-gray-100 dark:bg-gray-800 rounded-2xl p-4 md:p-4 shadow-lg">
            <div className="aspect-[16/10] rounded-lg flex items-center justify-center">
              <Image
                width={2940}
                height={1840}
                src="./hero-dark.webp"
                alt="Crater Platform"
                className="rounded-lg w-full hidden dark:block"
              />
              <Image
                width={2940}
                height={1840}
                src="./hero-light.webp"
                alt="Crater Platform"
                className="rounded-lg w-full block dark:hidden"
              />
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
