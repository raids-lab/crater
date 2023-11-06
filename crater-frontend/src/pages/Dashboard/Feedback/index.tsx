import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("feedback", "");
  return (
    <div className="space-y-1 px-6 py-6 text-xl">
      <h1 className="text-3xl font-bold leading-loose">问题反馈</h1>
      <p>表单，或提供反馈邮箱。</p>
    </div>
  );
};
