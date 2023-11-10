import { useNavigate } from "react-router-dom";
import { SignupForm } from "./Form";

export default function Signup() {
  const navigate = useNavigate();
  return (
    <div className="container relative grid h-screen items-center justify-center md:grid-cols-2 md:px-0 lg:max-w-none">
      <div className="relative hidden h-full flex-col bg-muted p-10 text-white dark:border-r lg:flex">
        <div className="absolute inset-0 bg-gradient-to-tr from-sky-900 to-sky-600 dark:from-secondary dark:to-secondary" />
        <button
          className="relative z-20 flex items-center text-lg font-medium"
          onClick={() => {
            navigate("/login");
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
      <div className="lg:p-8">
        <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
          <div className="flex flex-col space-y-2 text-center">
            <h1 className="text-2xl font-semibold tracking-tight">用户注册</h1>
            <SignupForm />
          </div>
        </div>
      </div>
    </div>
  );
}
