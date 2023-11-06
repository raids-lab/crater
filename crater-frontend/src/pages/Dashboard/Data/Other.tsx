import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("data", "other");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">其他类型文件管理</h1>
      <p>跳转到一个 Jupyter 界面。</p>
    </div>
  );
};
