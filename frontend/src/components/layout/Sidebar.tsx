import React from "react";
import { RouteObject } from "react-router-dom";

export type SidebarSubItem = {
  route: RouteObject;
};

export type SidebarItem = {
  path: string;
  children: SidebarSubItem[];
  route?: RouteObject;
};

export type SidebarMenu = {
  path: string;
  icon: React.ElementType;
  route: RouteObject;
};
