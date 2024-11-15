import { useLocation } from "react-router-dom";

export const useIsAdmin = () => {
  const location = useLocation();

  const pathParts = location.pathname.split("/").filter(Boolean);
  const isAdminView = pathParts[0] === "admin";

  return isAdminView;
};
