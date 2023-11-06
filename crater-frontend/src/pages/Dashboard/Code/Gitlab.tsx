import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("code", "gitlab");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">关联 Gitlab 仓库</h1>
      <p>让用户授权系统对 Gitlab 的访问，从而允许在镜像制作时直接使用代码。</p>
      <p>后续应该会放到个人设置中。</p>
    </div>
  );
};
