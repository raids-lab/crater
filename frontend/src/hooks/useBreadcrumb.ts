import { BreadCrumbItem, globalBreadCrumb } from "@/utils/store";
import { getBreadcrumbByPath } from "@/utils/title";
import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";
import { useSetAtom } from "jotai";
import { useTranslation } from "react-i18next";

const useBreadcrumb = () => {
  const setBreadcrumb = useSetAtom(globalBreadCrumb);
  const location = useLocation();
  const [detail, setDetail] = useState<BreadCrumbItem[]>([]);
  const { t } = useTranslation();

  useEffect(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    const titles = getBreadcrumbByPath(pathParts);
    if (titles) {
      let url = "";
      const ans: BreadCrumbItem[] = [];
      for (let i = 0; i < titles.length; i++) {
        url += `/${titles[i].path}`;
        ans.push({
          title: t(titles[i].title), // 会在渲染时使用翻译
          path: url,
          isEmpty: titles[i].isEmpty,
        });
      }
      if (detail) {
        ans.push(...detail);
      }
      setBreadcrumb(ans.slice(1));
    }
  }, [location, setBreadcrumb, detail, t]);

  return setDetail;
};

export default useBreadcrumb;
