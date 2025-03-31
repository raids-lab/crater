import { urlWebsiteBaseAtom } from "@/utils/store/config";
import { useAtomValue } from "jotai";
import { type FC } from "react";

const Website: FC = () => {
  const website = useAtomValue(urlWebsiteBaseAtom);
  window.location.href = website;
  return <></>;
};

export default Website;
