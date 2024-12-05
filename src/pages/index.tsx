import clsx from "clsx";
import Link from "@docusaurus/Link";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import Heading from "@theme/Heading";

import DemoImg from "@site/static/img/undraw_read_notes_gqmq.svg";

import styles from "./index.module.css";
import { Zap, BarChart, Box } from "lucide-react";
import TestimonialCard from "./TestimonialCard";

const testimonials = [
  {
    text: "GPUCluster彻底改变了我们的研究工作流程。我们的模型训练速度提高了5倍，使我们能够更快地迭代并推动AI的边界。",
    name: "陈博士",
    title: "AI研究主管，科技创新公司",
    imageUrl: "https://github.com/mondaycha.png",
  },
  {
    text: "作为一家初创公司，GPUCluster对我们来说是一个游戏规则的改变者。该平台的易用性和强大功能使我们能够在AI领域与大公司竞争。",
    name: "李明",
    title: "首席技术官，AI初创公司",
    imageUrl: "https://github.com/mondaycha.png",
  },
  {
    text: "GPUCluster彻底改变了我们的研究工作流程。我们的模型训练速度提高了5倍，使我们能够更快地迭代并推动AI的边界。",
    name: "陈博士",
    title: "AI研究主管，科技创新公司",
    imageUrl: "https://github.com/mondaycha.png",
  },
  {
    text: "作为一家初创公司，GPUCluster对我们来说是一个游戏规则的改变者。该平台的易用性和强大功能使我们能够在AI领域与大公司竞争。",
    name: "李明",
    title: "首席技术官，AI初创公司",
    imageUrl: "https://github.com/mondaycha.png",
  },
];

function LandingPage() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <div className="flex flex-col min-h-screen">
      <main>
        {/* Hero Section */}
        <section className="text-gray-600 dark:text-gray-300 body-font bg-white dark:bg-gray-900 transition-colors duration-200">
          <div className="container mx-auto flex px-5 py-24 md:flex-row flex-col items-center h-[calc(100vh_-_60px)]">
            <div className="lg:flex-grow md:w-1/2 lg:pr-24 md:pr-16 flex flex-col md:items-start md:text-left mb-16 md:mb-0 items-center text-center">
              <h1 className="title-font leading-tight text-4xl mb-4 font-semibold text-gray-900 dark:text-white">
                使用异构集群管理平台
                <br className="hidden lg:inline-block" />
                加速您的科研工作
              </h1>
              <p className="mb-8 leading-relaxed">
                在 GPU 集群上运行您的深度学习训练和推理任务。
                <br className="hidden lg:inline-block" />
                轻松扩展您的 AI 项目，加快研究和开发速度。
              </p>
              <div className="flex justify-center">
                <Link
                  to={`${siteConfig.url}/portal`}
                  className="flex-shrink-0 text-white hover:no-underline hover:text-white bg-sky-700 border-0 py-2 px-8 focus:outline-none hover:bg-sky-600 rounded text-lg mt-10 sm:mt-0"
                >
                  立即体验
                </Link>
                <Link
                  className="ml-4 inline-flex text-gray-700 bg-gray-100 py-2 px-8 border-0 focus:outline-none hover:no-underline hover:text-black hover:bg-gray-200 rounded text-lg"
                  to="/docs/intro"
                >
                  使用教程
                </Link>
              </div>
            </div>
            <div className="lg:max-w-lg lg:w-full md:w-1/2 w-5/6">
              <DemoImg className="object-cover object-center w-full" />
            </div>
          </div>
        </section>

        {/* Feature Section */}
        <section className="text-gray-600 dark:text-gray-300 body-font bg-gray-100 dark:bg-gray-800 transition-colors duration-200">
          <div className="container px-5 py-24 mx-auto">
            <div className="text-center mb-20">
              <h2 className="sm:text-3xl text-2xl font-medium title-font text-gray-900 dark:text-white mb-4">
                为AI研究人员和开发者提供强大功能
              </h2>
              <p className="text-base leading-relaxed xl:w-2/4 lg:w-3/4 mx-auto text-gray-500 dark:text-gray-400">
                我们的平台提供全面的工具和功能，以简化您的深度学习工作流程。
              </p>
              <div className="flex mt-6 justify-center">
                <div className="w-16 h-1 rounded-full bg-sky-600 inline-flex"></div>
              </div>
            </div>
            <div className="flex flex-wrap sm:-m-4 -mx-4 -mb-10 -mt-4 md:space-y-0 space-y-6">
              <div className="p-4 md:w-1/3 flex flex-col text-center items-center">
                <div className="w-20 h-20 inline-flex items-center justify-center rounded-full bg-sky-100 dark:bg-sky-800 text-sky-500 dark:text-sky-300 mb-5 flex-shrink-0">
                  <Box className="w-10 h-10" />
                </div>
                <div className="flex-grow">
                  <h2 className="text-gray-900 dark:text-white text-lg title-font font-medium mb-3">
                    容器化支持
                  </h2>
                  <p className="leading-relaxed text-base">
                    使用容器技术轻松部署和管理
                    <br />
                    深度学习任务，提高效率
                  </p>
                </div>
              </div>
              <div className="p-4 md:w-1/3 flex flex-col text-center items-center">
                <div className="w-20 h-20 inline-flex items-center justify-center rounded-full bg-sky-100 dark:bg-sky-800 text-sky-500 dark:text-sky-300 mb-5 flex-shrink-0">
                  <Zap className="w-10 h-10" />
                </div>
                <div className="flex-grow">
                  <h2 className="text-gray-900 dark:text-white text-lg title-font font-medium mb-3">
                    高性能加速卡
                  </h2>
                  <p className="leading-relaxed text-base">
                    访问为深度学习任务优化的
                    <br />
                    异构加速卡，确保最佳性能
                  </p>
                </div>
              </div>
              <div className="p-4 md:w-1/3 flex flex-col text-center items-center">
                <div className="w-20 h-20 inline-flex items-center justify-center rounded-full bg-sky-100 dark:bg-sky-800 text-sky-500 dark:text-sky-300 mb-5 flex-shrink-0">
                  <BarChart className="w-10 h-10" />
                </div>
                <div className="flex-grow">
                  <h2 className="text-gray-900 dark:text-white text-lg title-font font-medium mb-3">
                    实时监控
                  </h2>
                  <p className="leading-relaxed text-base">
                    通过全面的仪表板和性能指标
                    <br />
                    实时监控您的任务
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Testimonial Section */}
        <section className="text-gray-600 dark:text-gray-300 body-font bg-white dark:bg-gray-900 transition-colors duration-200">
          <div className="container px-5 py-24 mx-auto">
            <div className="text-center mb-12">
              <h2 className="text-3xl font-medium title-font text-gray-900 dark:text-white mb-6 text-center">
                用户评价
              </h2>
              <div className="flex justify-center">
                <div className="w-16 h-1 rounded-full bg-sky-600 inline-flex"></div>
              </div>
            </div>
            <div className="flex flex-wrap -m-4">
              {testimonials.map((testimonial, index) => (
                <TestimonialCard
                  key={index}
                  text={testimonial.text}
                  name={testimonial.name}
                  title={testimonial.title}
                  imageUrl={testimonial.imageUrl}
                />
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className={clsx("hero hero--primary", styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/intro"
          >
            Docusaurus Tutorial - 5min ⏱️
          </Link>
        </div>
      </div>
    </header>
  );
}

export default function Home(): JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout
      title={`Hello from ${siteConfig.title}`}
      description="Description will go into a meta tag in <head />"
    >
      <LandingPage />
    </Layout>
  );
}
