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

import DynamicLink from "fumadocs-core/dynamic-link";
import { ArrowRight, BookOpenIcon } from "lucide-react";
import Image from "next/image";
import { useParams } from "next/navigation";

// Define interface for text content
interface TextContent {
  title: string;
  description: {
    line1: string;
    line2: string;
  };
  buttons: {
    getStarted: string;
    readDocs: string;
  };
  altText: string;
}

// Define text content for each language
const textContent: Record<string, TextContent> = {
  cn: {
    title: "Crater · 云原生智算平台",
    description: {
      line1: "基于 Kubernetes 的机器学习一站式解决方案",
      line2: "整合开源生态，为 AI 训练与服务提供简单高效的体验",
    },
    buttons: {
      getStarted: "开始使用",
      readDocs: "阅读文档",
    },
    altText: "Crater Platform",
  },
  en: {
    title: "Crater - Cloud-Native AI Platform",
    description: {
      line1: "A one-stop solution for machine learning based on Kubernetes.",
      line2: "Efficient experience in AI training and serving.",
    },
    buttons: {
      getStarted: "Get Started",
      readDocs: "Read the Docs",
    },
    altText: "Crater Platform",
  },
};

export function HeroSection() {
  const { lang = "cn" } = useParams();

  // Select the appropriate content based on lang (default to 'cn' if not found)
  const content = textContent[lang as string] || textContent.cn;

  return (
    <section className="py-20 px-4">
      <div className="container mx-auto my-20 max-w-6xl">
        <div className="flex flex-col items-center text-center">
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold mb-6 bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-cyan-500">
            {content.title}
          </h1>
          <p className="text-xl md:text-2xl leading-relaxed text-balance max-w-3xl mb-10 text-gray-700 dark:text-gray-300">
            <span>{content.description.line1}</span>
            <br />
            <span>{content.description.line2}</span>
          </p>
          <div className="flex flex-col sm:flex-row gap-4">
            <a
              href={`/crater/${lang}/docs/admin/deployment/overview|||https://gpu.act.buaa.edu.cn/portal`}
              className="flex items-center justify-center gap-2 bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg transition-colors"
            >
              {content.buttons.getStarted}
              <ArrowRight size={18} />
            </a>
            <DynamicLink
              href="/[lang]/docs"
              className="flex items-center justify-center gap-2 bg-gray-200 dark:bg-gray-800 hover:bg-gray-300 dark:hover:bg-gray-700 px-6 py-3 rounded-lg transition-colors"
            >
              {content.buttons.readDocs}
              <BookOpenIcon size={18} />
            </DynamicLink>
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
                alt={content.altText}
                className="rounded-lg w-full hidden dark:block"
              />
              <Image
                width={2940}
                height={1840}
                src="./hero-light.webp"
                alt={content.altText}
                className="rounded-lg w-full block dark:hidden"
              />
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
