import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("overview", "");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">Overview</h1>
      <p>用户登录到系统后，最先看到的页面。</p>
      <p>集中显示用户的近期项目、资源使用情况、任务进度等。</p>
    </div>
  );
};
