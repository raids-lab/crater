import { globalFixedLayout } from "@/utils/store";
import { useSetAtom } from "jotai";
import { useEffect } from "react";

const useFixedLayout = () => {
  const setFixedLayout = useSetAtom(globalFixedLayout);

  useEffect(() => {
    setFixedLayout(true);
    return () => {
      setFixedLayout(false);
    };
  });
};

export default useFixedLayout;
