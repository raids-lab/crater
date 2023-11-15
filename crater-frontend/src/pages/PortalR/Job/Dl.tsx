import type { FC } from "react";

export const Component: FC = () => {
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">深度推荐训练任务</h1>
      <p>某种特定任务的列表、新建、调度结果展示</p>
      <p>点击某个任务后，可查看具体的进度、资源监控等信息。</p>
      <p>新建任务时，可以关联代码和数据集。</p>
    </div>
  );
};
