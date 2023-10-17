import {
  ACCESS_TOKEN_KEY,
  ILogin,
  ILoginResponse,
  ISignup,
  ISignupResponse,
  REFRESH_TOKEN_KEY,
} from "./types";
import useAxios from "./useAxios";

const useAuth = () => {
  const { instance } = useAxios();

  const signupUserFn = async (user: ISignup) => {
    const response = await instance.post<ISignupResponse>("signup", user);
    const { accessToken, refreshToken } = response.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    return response.data;
  };

  const loginUserFn = async (user: ILogin) => {
    const response = await instance.post<ILoginResponse>("login", user);
    const { accessToken, refreshToken } = response.data;
    localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
    localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
    return response.data;
  };

  return { signupUserFn, loginUserFn };
};

export default useAuth;
