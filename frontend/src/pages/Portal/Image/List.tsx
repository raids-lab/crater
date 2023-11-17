import type { FC } from "react";

export const Component: FC = () => {
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <p>打算用 Harbor 管理，前端仅展示该用户创建的镜像组。</p>
      <p>提供公开和私有两个选项卡，公开镜像为管理员上传。</p>
    </div>
  );
};
