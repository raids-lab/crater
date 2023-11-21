import type { FC } from "react";

export const GroupUser: FC = () => {
  return (
    <div className="space-y-1 px-6 py-6 text-base">
      <p>1. 创建账户</p>
      <p>2. 编辑账户信息</p>
      <p>3. 删除账户</p>
      <p>4. 指定账户管理员</p>
      <p>5. 指定账户成员（添加、删除）</p>
      <p>
        6.
        指定账户资源配额（参考slurm，可以提交的Job数量、可以同时运行的Job数量、可以申请的节点数量、可以同时使用的节点数量，可以申请的CPU核心数量，可以同时使用的CPU核心数量，可以申请的GPU数量，可以同时使用的GPU数量，包括不同型号的GPU、是否可共享GPU等）
      </p>
    </div>
  );
};
