import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("image", "make");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">镜像制作</h1>
      <p>为用户提供一系列基础镜像，提供低代码的镜像打包方式。</p>
      <p>用户可以将关联的 ACT Gitlab 仓库中指定分支的代码，打包到镜像中 。</p>
    </div>
  );
};
