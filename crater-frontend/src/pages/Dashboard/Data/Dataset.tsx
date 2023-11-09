import type { FC } from "react";

export const Component: FC = () => {
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">数据集管理</h1>
      <p>用户可上传数据集的压缩包。</p>
      <p>可以在创建任务时绑定数据集。</p>
      <p>后续系统可以关联训练过程对数据集的优化。</p>
    </div>
  );
};
