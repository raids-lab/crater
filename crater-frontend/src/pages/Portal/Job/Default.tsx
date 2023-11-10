import type { FC } from "react";

export const Component: FC = () => {
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">通用任务管理</h1>
      <p>即 Kubernetes 自带的 Job。</p>
    </div>
  );
};
