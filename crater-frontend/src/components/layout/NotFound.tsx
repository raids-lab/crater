import { Button } from "@/components/ui/button";
import { useNavigate } from "react-router-dom";

export default function NotFound() {
  const navigate = useNavigate();
  return (
    <div className="h-svh">
      <div className="m-auto flex h-full w-full flex-col items-center justify-center gap-2">
        <h1 className="text-[5rem] font-bold leading-tight">404</h1>
        <span className="font-medium">页面未找到</span>
        <p className="text-center text-muted-foreground">
          你正在寻找的页面 <br />
          不存在或者可能已被移除
        </p>
        <div className="mt-6 flex gap-4 pb-24">
          <Button variant="outline" onClick={() => history.go(-1)}>
            返回上级
          </Button>
          <Button onClick={() => navigate("/")}>返回首页</Button>
        </div>
      </div>
    </div>
  );
}
