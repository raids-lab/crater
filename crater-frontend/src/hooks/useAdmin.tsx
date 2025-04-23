import { useLocation } from "react-router-dom";

/**
 * useIsAdmin is a custom hook that checks if the current path is an admin view.
 * @returns true if the current path is an admin view, false otherwise
 */
const useIsAdmin = () => {
  const location = useLocation();

  const pathParts = location.pathname.split("/").filter(Boolean);
  const isAdminView = pathParts[0] === "admin";

  return isAdminView;
};

export default useIsAdmin;
