import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("image", "list");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">镜像列表</h1>
      <p>后端打算用 Harbor 管理，前端仅展示该用户创建的镜像组。</p>
    </div>
  );
};
