import { useMemo, useEffect, Fragment } from "react";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { useRecoilState } from "recoil";
import { globalBreadCrumb } from "@/utils/store";
import { Link, useLocation } from "react-router-dom";
import { getBreadcrumbByPath } from "@/utils/title";
import { cn } from "@/lib/utils";

export const NavBreadcrumb = ({ className }: { className: string }) => {
  const location = useLocation();
  const pathParts = useMemo(() => {
    const pathParts = location.pathname.split("/").filter(Boolean);
    return pathParts;
  }, [location]);
  const [breadcrumb, setBreadcrumb] = useRecoilState(globalBreadCrumb);
  useEffect(() => {
    const titles = getBreadcrumbByPath(pathParts);
    if (titles) {
      let url = "";
      const ans = [];
      for (let i = 0; i < titles.length; i++) {
        url += `/${titles[i].path}`;
        ans.push({
          title: titles[i].title,
          path: i !== titles.length - 1 ? url : undefined,
        });
      }
      if (ans.length > 2) {
        ans[ans.length - 2].path = undefined;
      }
      setBreadcrumb(ans.slice(1));
    }
  }, [pathParts, setBreadcrumb]);

  return (
    <Breadcrumb className={className}>
      <BreadcrumbList>
        {breadcrumb.map((item, index) => {
          return (
            <Fragment key={`bread-${index}`}>
              {index !== 0 && (
                <BreadcrumbSeparator key={`bread-separator-${index}`} />
              )}
              {index === 0 && (
                <BreadcrumbPage
                  className={cn({
                    "text-muted-foreground": breadcrumb.length > 1,
                  })}
                >
                  {item.title}
                </BreadcrumbPage>
              )}
              {index !== 0 && (
                <BreadcrumbItem key={`bread-item-${index}`}>
                  {item.path && (
                    <BreadcrumbLink asChild>
                      <Link to={item.path}>{item.title}</Link>
                    </BreadcrumbLink>
                  )}
                  {!item.path && <BreadcrumbPage>{item.title}</BreadcrumbPage>}
                </BreadcrumbItem>
              )}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
};
