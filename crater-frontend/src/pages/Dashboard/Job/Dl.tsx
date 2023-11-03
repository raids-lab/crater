import type { FC } from "react";
import { useIndex } from "../hooks/useIndex";

export const Component: FC = () => {
  useIndex("job", "dl");
  return <>Dl</>;
};
