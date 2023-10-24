import { uiActivedState } from "@/utils/store";
import { useEffect } from "react";
import { useSetRecoilState } from "recoil";

export const useIndex = (item: string, subItem: string) => {
  const setActived = useSetRecoilState(uiActivedState);

  useEffect(() => {
    setActived({
      item: item,
      subItem: subItem,
    });
  }, [item, setActived, subItem]);
};
