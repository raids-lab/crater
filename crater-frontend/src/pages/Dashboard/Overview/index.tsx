import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("overview", "");
  return (
    <div className="px-6 py-6">
      <h1 className="text-3xl leading-loose">Overview</h1>
      <p>集中显示用户最近的各种项目、资源使用情况等</p>
    </div>
  );
};
