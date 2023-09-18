import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useNavigate } from "react-router-dom";
import { currentUserState } from "@/utils/store";
import { useSetRecoilState } from "recoil";

export default function Login() {
  const navigate = useNavigate();
  const setUserState = useSetRecoilState(currentUserState);

  return (
    <>
      <div className="container relative hidden h-screen flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="relative hidden h-full flex-col bg-muted p-10 text-white dark:border-r lg:flex">
          <div className="absolute inset-0 bg-gradient-to-tr from-sky-900 to-sky-600" />
          <div className="relative z-20 flex items-center text-lg font-medium">
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
          </div>
          <div className="relative z-20 mt-auto">
            <blockquote className="space-y-2">
              <footer className="text-sm">Copyright @ACT</footer>
            </blockquote>
          </div>
        </div>
        <div className="lg:p-8">
          <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                用户登录
              </h1>
              {/* <p className="text-sm text-muted-foreground">
                admin: 123456 user: 000000
              </p> */}
              <Input type="text" placeholder="用户名" id="user" />
              <Input type="password" placeholder="密码" id="pass" />
              <Button
                className="bg-sky-700 hover:bg-sky-800"
                onClick={() => {
                  setUserState((old) => {
                    return {
                      ...old,
                      role: "admin",
                    };
                  });
                  navigate("/dashboard");
                }}
              >
                登录
              </Button>
            </div>
            <p className="px-2 text-center text-sm text-muted-foreground">
              测试用账号：admin 123456
            </p>
          </div>
        </div>
      </div>
    </>
  );
}
