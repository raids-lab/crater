import type { FC } from "react";

const Navibar: FC = () => {
  return (
    <>
      <div className="flex h-full flex-row items-center justify-start border-b bg-background px-6">
        <p className="font-bold">GPU 集群管理系统</p>
      </div>
    </>
  );
};

export default Navibar;
