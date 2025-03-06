import { RouteObject } from "react-router-dom";

export type RouteSubItem = {
  route: RouteObject;
};

export type RouteItem = {
  path: string;
  children: RouteSubItem[];
  route?: RouteObject;
};
