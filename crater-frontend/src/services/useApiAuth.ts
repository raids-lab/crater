import {
  ACCESS_TOKEN_KEY,
  ILogin,
  ILoginResponse,
  ISignup,
  ISignupResponse,
  REFRESH_TOKEN_KEY,
} from "./types";
import useAxios from "./useAxios";

const useApiAuth = () => {
  const { instance } = useAxios();

  const apiSignupUser = async (user: ISignup) => {
    const response = await instance.post<ISignupResponse>("signup", user);
    const { accessToken, refreshToken } = response.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    return response.data;
  };

  const apiLoginUser = async (user: ILogin): Promise<ILoginResponse> => {
    const response = await instance.post<ILoginResponse>("login", user);
    const { accessToken, refreshToken } = response.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    return response.data;
  };

  return { apiSignupUser, apiLoginUser };
};

export default useApiAuth;
