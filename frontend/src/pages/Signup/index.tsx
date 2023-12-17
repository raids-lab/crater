import { Link } from "react-router-dom";
import { SignupForm } from "./Form";
import { CardHeader, CardContent, Card } from "@/components/ui/card";
import Logo2 from "@/assets/logo2.svg";

export default function Signup() {
  return (
    <div className="relative grid h-screen w-screen bg-gradient-to-tr from-sidebar to-primary md:grid-cols-2 md:px-0">
      <div
        className="absolute left-10 top-10 z-20 flex items-center text-lg font-medium"
        title="Switch signup and login"
      >
        <img className="h-10 w-auto select-none" src={Logo2} alt="logo" />
      </div>
      <div className="absolute bottom-10 left-10 z-20">
        <blockquote className="space-y-2">
          <footer className="text-sm text-white">Copyright @ACT</footer>
        </blockquote>
      </div>
      <div className="relative hidden h-full text-white dark:from-secondary dark:to-secondary md:block">
        <div className="flex h-full w-full flex-col items-start justify-center px-6 py-8 lg:px-16 lg:py-12">
          <h1 className="mb-4 text-left text-6xl font-bold leading-tight">
            欢迎体验
            <br />
            GPU 集群管理平台
          </h1>
          <p className="text-left text-xl text-slate-400 dark:text-muted-foreground">
            Crater 是一个基于 Kubernetes 的 GPU 集群管理平台，<br></br>
            提供了一站式的 GPU 集群管理解决方案。
          </p>
        </div>
      </div>
      <div className="flex items-center justify-center">
        <Card className="w-full max-w-md">
          <CardHeader>
            <h2 className="text-center text-2xl font-bold">用户注册</h2>
          </CardHeader>
          <CardContent className="space-y-4">
            <SignupForm />
            <p className="mt-4 px-2 text-center text-sm text-muted-foreground">
              已有账号？立即
              <span className="ml-1 underline">
                <Link to="/login">登录</Link>
              </span>
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
