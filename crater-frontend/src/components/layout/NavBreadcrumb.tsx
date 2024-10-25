import { Fragment } from "react";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { useAtomValue } from "jotai";
import { globalBreadCrumb } from "@/utils/store";
import { Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import useBreadcrumb from "@/hooks/useBreadcrumb";

export const NavBreadcrumb = ({ className }: { className: string }) => {
  useBreadcrumb();
  const breadcrumb = useAtomValue(globalBreadCrumb);

  return (
    <Breadcrumb className={className}>
      <BreadcrumbList>
        {breadcrumb.map((item, index) => {
          return (
            <Fragment key={`bread-${index}`}>
              {index !== 0 && (
                <BreadcrumbSeparator key={`bread-separator-${index}`} />
              )}
              {item.isEmpty && (
                <BreadcrumbPage
                  className={cn({
                    "text-muted-foreground": breadcrumb.length > 1,
                  })}
                >
                  {item.title}
                </BreadcrumbPage>
              )}
              {!item.isEmpty && (
                <BreadcrumbItem key={`bread-item-${index}`}>
                  {item.path && index !== breadcrumb.length - 1 ? (
                    <BreadcrumbLink asChild>
                      <Link to={item.path}>{item.title}</Link>
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage>{item.title}</BreadcrumbPage>
                  )}
                </BreadcrumbItem>
              )}
            </Fragment>
          );
        })}
      </BreadcrumbList>
    </Breadcrumb>
  );
};
