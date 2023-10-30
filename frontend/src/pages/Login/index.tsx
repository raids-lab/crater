import { Link, useNavigate } from "react-router-dom";
import { ProfileForm } from "./Form";

export default function Login() {
  const navigate = useNavigate();
  return (
    <>
      <div className="container relative grid h-screen items-center justify-center md:grid-cols-2 md:px-0 lg:max-w-none">
        <div className="relative hidden h-full flex-col bg-muted p-10 text-white dark:border-r md:flex">
          <div className="absolute inset-0 bg-gradient-to-tr from-sky-900 to-sky-600" />
          <button
            className="relative z-20 flex items-center text-lg font-medium"
            title="Switch signup and login"
            onClick={() => {
              navigate("/signup");
            }}
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="mr-2 h-6 w-6"
            >
              <rect width="7" height="7" x="3" y="3" rx="1" />
              <rect width="7" height="7" x="14" y="3" rx="1" />
              <rect width="7" height="7" x="14" y="14" rx="1" />
              <rect width="7" height="7" x="3" y="14" rx="1" />
            </svg>
            GPU Portal
          </button>
          <div className="relative z-20 mt-auto">
            <blockquote className="space-y-2">
              <footer className="text-sm">Copyright @ACT</footer>
            </blockquote>
          </div>
        </div>
        <div className="m-auto flex w-full flex-col items-stretch space-y-6 sm:w-[350px] lg:p-8">
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-2xl font-semibold tracking-tight">用户登录</h1>
            <ProfileForm />
          </div>
          <p className="px-2 text-center text-sm text-muted-foreground">
            测试用账号：test 123456
            <span className="ml-2 underline">
              <Link to="/signup">注册</Link>
            </span>
          </p>
        </div>
      </div>
    </>
  );
}
