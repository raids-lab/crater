import { useAuth } from "@/hooks/useAuth";
import { FC, PropsWithChildren } from "react";
import { Navigate } from "react-router-dom";
import { Role } from "@/services/api/auth";

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth(Role.User);
  // const isAuthenticated = true;
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

export default AuthedRouter;
