import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("data", "personal");
  return (
    <div className="px-6 py-6">
      <h1 className="text-3xl leading-loose">个人空间管理</h1>
      <p>其它类型的文件，使用 Jupyter 界面</p>
    </div>
  );
};
