import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("jupyter", "");
  return (
    <div className="px-6 py-6">
      <h1 className="text-3xl leading-loose">Jupyter 管理</h1>
      <p>显示已申请的 Jupyter Notebook 列表、新建 Jupyter Notebook 等功能</p>
    </div>
  );
};
