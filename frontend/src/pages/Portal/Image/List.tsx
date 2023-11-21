import type { FC } from "react";

export const Component: FC = () => {
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <p>1. 管理私有镜像（删改查）</p>
      <p>2. 共享镜像（选择某个组可见、全局可见）</p>
      <p>3. 查看公共镜像（他人共享）</p>
    </div>
  );
};
