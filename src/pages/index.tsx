import Link from "@docusaurus/Link";
import useDocusaurusContext from "@docusaurus/useDocusaurusContext";
import Layout from "@theme/Layout";
import HeroSvg from "@site/static/img/undraw_read_notes_gqmq.svg";
import DocusaurusImageUrl from "@site/static/img/docusaurus.png";
import wechatQrCode from "@site/static/img/qrcode.png";
import CraterLight from "@site/static/img/crater-light.webp";
import CraterDark from "@site/static/img/crater-dark.webp";
import { Zap, BarChart, Box } from "lucide-react";
import TestimonialCard from "./TestimonialCard";

const testimonials = [
  {
    text: "Crater 彻底改变了我们的研究工作流程。我们的模型训练速度提高了 5 倍，使我们能够更快地迭代并推动 AI 的边界。",
    name: "周博士",
    title: "AI 研究主管，科技创新公司",
    imageUrl: DocusaurusImageUrl,
  },
  {
    text: "Crater 是我们团队的必备工具。我们的研究人员和开发人员都喜欢使用它，因为它提供了一个简单而强大的方式来管理我们的深度学习任务。",
    name: "周同学",
    title: "AI 研究员，大学",
    imageUrl: DocusaurusImageUrl,
  },
  {
    text: "从 Slurm 到 Kubernetes，Crater 为我们提供了一个统一的平台，使我们能够在不同的集群上运行我们的深度学习任务。",
    name: "周工程师",
    title: "AI 工程师，科技公司",
    imageUrl: DocusaurusImageUrl,
  },
  {
    text: "作为一家初创公司，Crater 对我们来说是一个游戏规则的改变者。该平台的易用性和强大功能使我们能够在 AI 领域与大公司竞争。",
    name: "周明",
    title: "首席技术官，AI 初创公司",
    imageUrl: DocusaurusImageUrl,
  },
];

const questions = [
  {
    question: "如何使用 Crater 运行深度学习任务？",
    answer:
      "您可以在我们的文档中找到有关如何使用 Crater 运行深度学习任务的详细说明。",
  },
  {
    question: "Crater 支持哪些深度学习框架？",
    answer: "Crater 支持 TensorFlow、PyTorch、MXNet 等流行的深度学习框架。",
  },
  {
    question: "我可以在 Crater 上运行多任务吗？",
    answer: "是的，您可以在 Crater 上同时运行多个深度学习任务。",
  },
  {
    question: "Crater 如何保证我的任务的安全性？",
    answer: "Crater 使用容器技术隔离您的任务，确保您的任务在安全的环境中运行。",
  },
  {
    question: "Crater 如何保证我的任务的性能？",
    answer: "Crater 提供了高性能的 GPU 集群，确保您的任务获得最佳的性能。",
  },
  {
    question: "Crater 如何收费？",
    answer:
      "Crater 目前处于内测阶段，我们提供免费试用。请联系我们获取更多信息。",
  },
  {
    question: "我如何联系 Crater 的技术支持团队？",
    answer:
      "您可以通过微信群或邮件联系我们的技术支持团队。我们会尽快回复您的问题。",
  },
  {
    question: "Crater 支持哪些集群管理系统？",
    answer: "Crater 支持 Slurm、Kubernetes 等流行的集群管理系统。",
  },
  {
    question: "我如何加入 Crater 内测？",
    answer:
      "您可以通过我们的网站申请加入 Crater 内测。我们会尽快审核您的申请。",
  },
  {
    question: "Crater 如何保证我的数据的隐私？",
    answer: "Crater 严格遵守数据隐私法规，确保您的数据不会被泄露。",
  },
];

function LandingPage() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <div className="flex flex-col min-h-screen">
      <main>
        {/* Hero Section */}
        <section className="mx-auto px-4 pt-32 pb-32 max-w-6xl space-y-32">
          <div className="text-center mx-auto space-y-12">
            {/* Gradient Heading */}
            <h1 className="text-5xl md:text-7xl font-bold">
              使用异构集群管理平台
              <br className="hidden lg:inline-block" />
              加速您的科研工作
            </h1>
            {/* CTA Buttons */}
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
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
          {/* Screenshot */}
          <div className="relative w-full aspect-video">
            <img
              src={CraterLight}
              alt="AI IDE Interface"
              className="rounded-lg shadow-2xl dark:hidden"
            />
            <img
              src={CraterDark}
              alt="AI IDE Interface"
              className="rounded-lg shadow-2xl hidden dark:block"
            />
          </div>
        </section>

        {/* Feature Section */}
        <section className="text-gray-600 dark:text-gray-300 body-font bg-gray-100 dark:bg-gray-800 transition-colors duration-200">
          <div className="container px-5 py-24 mx-auto">
            <div className="text-center mb-20">
              <h2 className="sm:text-3xl text-2xl font-medium title-font text-gray-900 dark:text-white mb-3">
                为 AI 研究人员和开发者提供强大功能
              </h2>
              <p className="text-base leading-relaxed xl:w-2/4 lg:w-3/4 mx-auto text-gray-500 dark:text-gray-400">
                我们的平台提供全面的工具和功能，以简化您的深度学习工作流程
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
              <h2 className="sm:text-3xl text-2xl font-medium title-font text-gray-900 dark:text-white mb-3">
                用户评价
              </h2>
              <p className="text-base leading-relaxed xl:w-2/4 lg:w-3/4 mx-auto text-gray-500 dark:text-gray-400">
                来自不是我们的用户的虚假评价
              </p>
              <div className="flex mt-6 justify-center">
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

        {/* Feedback Section */}
        <section className="text-gray-600 dark:text-gray-300 body-font bg-gray-100 dark:bg-gray-800 transition-colors duration-200">
          <div className="container px-5 py-24 mx-auto">
            <div className="text-center mb-12">
              <h2 className="text-3xl font-medium title-font text-gray-900 dark:text-white mb-6 text-center">
                意见反馈
              </h2>
              <div className="flex justify-center">
                <div className="w-16 h-1 rounded-full bg-sky-600 inline-flex"></div>
              </div>
            </div>
            <div className="flex flex-wrap -m-4">
              <div className="p-4 w-full flex justify-center">
                <div className="h-full bg-gray-100 dark:bg-gray-800 p-8 rounded flex flex-col items-center">
                  <div className="flex-1 text-center">
                    <p className="leading-relaxed mb-6">
                      我们非常重视您的意见和建议。如果您有任何问题或建议，请随时联系我们。
                    </p>
                  </div>
                  <div className="inline-flex items-center mt-4 flex-col">
                    <img
                      alt="QR Code"
                      src={wechatQrCode}
                      className="w-72 flex-shrink-0 object-cover object-center"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* 常见问题 Section */}
        <section className="text-gray-600 dark:text-gray-300 body-font bg-white dark:bg-gray-900 transition-colors duration-200">
          <div className="container px-5 py-24 mx-auto">
            <div className="text-center mb-12">
              <h2 className="text-3xl font-medium title-font text-gray-900 dark:text-white mb-6 text-center">
                常见问题
              </h2>
              <p className="text-base leading-relaxed xl:w-2/4 lg:w-3/4 mx-auto text-gray-500 dark:text-gray-400">
                目前全是用 AI 生成的问题和答案
              </p>
              <div className="flex justify-center">
                <div className="w-16 h-1 rounded-full bg-sky-600 inline-flex"></div>
              </div>
            </div>
            <div className="flex flex-wrap -m-4">
              {/* 10个常见问题，卡片形式 */}
              {questions.map((question, index) => (
                <div key={index} className="p-4 md:w-1/2 w-full">
                  <div className="h-full bg-gray-100 dark:bg-gray-800 p-8 rounded flex flex-col">
                    <div className="flex items-start gap-3">
                      <div className="w-8 h-8 inline-flex items-center justify-center rounded-full bg-sky-200 dark:bg-sky-800 text-sky-600 dark:text-sky-300 flex-shrink-0 font-semibold">
                        Q
                      </div>
                      <h3 className="text-lg font-medium text-gray-900 dark:text-white">
                        {question.question}
                      </h3>
                    </div>
                    <div className="flex items-start gap-3">
                      <div className="w-8 h-8 inline-flex items-center justify-center rounded-full bg-emerald-200 dark:bg-emerald-800 text-emerald-600 dark:text-emerald-300 flex-shrink-0 font-semibold">
                        A
                      </div>
                      <p className="leading-relaxed text-gray-500 dark:text-gray-400">
                        {question.answer}
                      </p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
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
