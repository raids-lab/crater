// 1. Import the "HttpResponse" class from the library.
import { IAuthResponse, ILogin } from "@/services/api/auth";
import { http, HttpResponse } from "msw";

const baseURL = import.meta.env.VITE_API_BASE_URL;

export const handlers = [
  http.post<never, ILogin, IAuthResponse>(
    baseURL + `login`,
    async ({ request }) => {
      const { username, password } = await request.json();
      const mockUser = {
        username: "lyl",
        password: "000000",
        accessToken: "mockAccessToken123",
        refreshToken: "mockRefreshToken456",
        role: "admin", // 或者根据你的需求改为 'admin', 'user' 等等
      };
      // 检查用户名和密码是否与mock用户匹配
      if (username === mockUser.username && password === mockUser.password) {
        // 成功登录，返回accessToken、refreshToken和role
        return HttpResponse.json(
          {
            accessToken: mockUser.accessToken,
            refreshToken: mockUser.refreshToken,
            role: mockUser.role,
          },
          { status: 200 },
        );
      } else {
        // 用户名或密码错误
        return HttpResponse.json(
          {
            accessToken: mockUser.accessToken,
            refreshToken: mockUser.refreshToken,
            role: mockUser.role,
          },
          { status: 401 },
        );
      }
    },
  ),
];
