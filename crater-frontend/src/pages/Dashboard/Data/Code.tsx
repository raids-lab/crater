import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("data", "code");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">关联 Gitlab 仓库</h1>
      <p>让用户授权系统对 Gitlab 的访问，从而允许在镜像制作时直接使用代码。</p>
      <p>后续应该会放到个人设置中。</p>
      <h1 className="pt-4 text-3xl font-bold leading-loose">用户手动上传</h1>
      <p>用户也可以直接上传代码压缩包，并保存。</p>
    </div>
  );
};
