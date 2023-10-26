import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("dl_training", "scheduler");
  return <>Scheduler</>;
};
