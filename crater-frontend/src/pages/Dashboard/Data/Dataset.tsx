import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("data", "dataset");
  return (
    <div className="px-6 py-6">
      <h1 className="text-3xl leading-loose">数据集管理</h1>
      <p>可以在创建任务时绑定依赖的数据集</p>
    </div>
  );
};
