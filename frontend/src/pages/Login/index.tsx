import CraterIcon from "@/components/icon/CraterIcon";
import CraterText from "@/components/icon/CraterText";
import { LoginForm } from "./Form";
import { toast } from "sonner";

export function Dashboard() {
  return (
    <div className="h-screen w-full lg:grid lg:grid-cols-2">
      <div className="hidden bg-muted lg:block">
        <div className="h-full w-full bg-gradient-to-tr from-slate-800 to-primary dark:from-black dark:to-secondary">
          <div
            className="absolute left-10 top-10 z-20 flex items-center text-lg font-medium"
            title="Switch signup and login"
          >
            <div className="flex h-14 w-full flex-row items-center justify-center text-white">
              <CraterIcon className="mr-1.5 h-8 w-8" />
              <CraterText className="h-4" />
            </div>
          </div>
          <div className="absolute bottom-10 left-10 z-20">
            <blockquote className="space-y-2">
              <footer className="text-sm text-white/80">Copyright @ACT</footer>
            </blockquote>
          </div>
          <div className="relative hidden h-full text-white md:block">
            <div className="flex h-full w-full flex-col items-start justify-center px-6 py-8 lg:px-16 lg:py-12">
              <h1 className="mb-4 text-left text-6xl font-bold leading-tight dark:text-primary">
                欢迎体验
                <br />
                异构云资源混合调度
                <br />
                与智能运维平台
              </h1>
              <p className="text-left text-xl text-slate-400 dark:text-muted-foreground">
                Crater 是基于 Kubernetes 的异构云资源混合调度与智能运维平台。
              </p>
            </div>
          </div>
        </div>
      </div>
      <div className="flex items-center justify-center py-12">
        <div className="mx-auto grid w-[350px] gap-6">
          <div className="grid gap-2 text-center">
            <h1 className="text-3xl font-bold">用户登录</h1>
            <p className="text-sm text-muted-foreground">
              已接入 ACT 实验室统一身份认证
            </p>
          </div>
          <LoginForm />
          <div className="text-center text-sm text-muted-foreground">
            还没有账号？
            <button
              onClick={() => toast.info("请联系 ACT 实验室账号管理员")}
              className="underline"
            >
              立即注册
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Dashboard;
