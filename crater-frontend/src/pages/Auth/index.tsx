import CraterIcon from "@/components/icon/CraterIcon";
import CraterText from "@/components/icon/CraterText";
import { LoginForm } from "./LoginForm";
import { useState } from "react";
import { SignupForm } from "./SignupForm";
import { ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTheme } from "@/utils/theme";

export function Dashboard() {
  const [showSignup, setShowSignup] = useState(false);
  const { theme, setTheme } = useTheme();

  return (
    <div className="h-screen w-full lg:grid lg:grid-cols-2">
      {/* 左侧部分 */}
      <div className="hidden bg-primary dark:bg-slate-800/70 lg:block">
        <div className="relative h-full w-full">
          {/* 背景SVG图像 */}
          <svg
            className="absolute inset-0 h-full w-full object-cover"
            xmlns="http://www.w3.org/2000/svg"
            preserveAspectRatio="xMidYMid slice"
            viewBox="0 0 1463 823"
          >
            <g fill="none" fillRule="evenodd">
              <path
                className="text-zinc-950 opacity-40"
                fill="currentColor"
                d="M-39.04 645.846L1561.58-44l-22.288 1752.758z"
              />
            </g>
          </svg>
          {/* 顶部Logo */}
          <div
            className="absolute left-10 top-10 z-20 flex items-center text-lg font-medium"
            title="Switch signup and login"
          >
            <button
              className="flex h-14 w-full flex-row items-center justify-center text-white"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            >
              <CraterIcon className="mr-1.5 h-8 w-8" />
              <CraterText className="h-4" />
            </button>
          </div>
          {/* 底部版权信息 */}
          <div className="absolute bottom-10 left-10 z-20">
            <blockquote className="space-y-2">
              <footer className="text-sm text-white/80">
                Copyright @ ACT RAIDS Lab
              </footer>
            </blockquote>
          </div>
          {/* 中间文字内容 */}
          <div className="relative flex h-full items-center justify-center">
            <div className="z-10 px-6 py-8 text-left text-white lg:px-16 lg:py-12">
              <h1 className="mb-4 text-5xl font-semibold leading-tight">
                <span className="dark:text-primary">欢迎体验</span>
                <br />
                异构云资源混合调度
                <br />
                与智能运维平台
              </h1>
              <Button
                variant="ghost"
                className="bg-white text-black hover:bg-slate-200 hover:text-black dark:bg-primary dark:text-primary-foreground hover:dark:bg-primary/85 dark:hover:text-primary-foreground"
                onClick={() =>
                  window.open(`https://${import.meta.env.VITE_HOST}/website/`)
                }
              >
                <ExternalLink />
                平台文档
              </Button>
            </div>
          </div>
        </div>
      </div>
      {/* 右侧表单部分 */}
      <div className="flex items-center justify-center py-12">
        {showSignup ? (
          <div className="mx-auto w-[350px] space-y-6">
            <div className="space-y-2 text-center">
              <h1 className="text-3xl font-bold">用户注册</h1>
              <p className="text-sm text-muted-foreground">
                仅面向特定用户提供此功能
              </p>
            </div>
            <SignupForm />
            <div className="text-center text-sm text-muted-foreground">
              已有账号？
              <button
                onClick={() => setShowSignup(false)}
                className="underline"
              >
                立即登录
              </button>
            </div>
          </div>
        ) : (
          <div className="mx-auto w-[350px] space-y-6">
            <div className="space-y-2 text-center">
              <h1 className="text-3xl font-bold">用户登录</h1>
              <p className="text-sm text-muted-foreground">
                已接入 ACT 实验室统一身份认证
              </p>
            </div>
            <LoginForm />
            <div className="text-center text-sm text-muted-foreground">
              还没有账号？
              <button onClick={() => setShowSignup(true)} className="underline">
                立即注册
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default Dashboard;
