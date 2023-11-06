import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("job", "default");
  return (
    <div className="px-6 py-6">
      <h1 className="text-3xl leading-loose">通用任务管理</h1>
      <p>即 Kubernetes 自带的 Job</p>
    </div>
  );
};
