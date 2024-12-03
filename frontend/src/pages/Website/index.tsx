import { type FC } from "react";

const Website: FC = () => {
  window.location.href = `https://${import.meta.env.VITE_HOST}/website`;
  return <></>;
};

export default Website;
