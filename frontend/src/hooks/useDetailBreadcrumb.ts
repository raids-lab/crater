import { BreadCrumbItem, globalBreadCrumb } from "@/utils/store";
import { getBreadcrumbByPath } from "@/utils/title";
import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { useSetRecoilState } from "recoil";

const useBreadcrumb = () => {
  const setBreadcrumb = useSetRecoilState(globalBreadCrumb);
  const location = useLocation();
  const [detail, setDetail] = useState<BreadCrumbItem[]>([]);

  useEffect(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    const titles = getBreadcrumbByPath(pathParts);
    if (titles) {
      let url = "";
      const ans = [];
      for (let i = 0; i < titles.length; i++) {
        url += `/${titles[i].path}`;
        ans.push({
          title: titles[i].title,
          path: url,
        });
      }
      if (detail) {
        ans.push(...detail);
      }
      setBreadcrumb(ans.slice(1));
    }
  }, [location, setBreadcrumb, detail]);

  return setDetail;
};

export default useBreadcrumb;
