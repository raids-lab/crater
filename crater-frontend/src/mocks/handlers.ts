// 1. Import the "HttpResponse" class from the library.
import { IAuthResponse, ILogin, Role } from "@/services/api/auth";
import { IResponse } from "@/services/types";
import { logger } from "@/utils/loglevel";
import { http, HttpResponse } from "msw";

const baseURL = import.meta.env.VITE_API_BASE_URL;

export const handlers = [
  http.post<never, ILogin, IResponse<IAuthResponse>>(
    baseURL + `login`,
    async ({ request }) => {
      const { username, password } = await request.json();
      logger.info(`login with username: ${username}, password: ${password}`);
      const mockUser = {
        username: "Huangjx",
        password: "",
        accessToken:
          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjIzLCJwaWQiOjMwLCJwcm8iOjMsImNpZCI6MSwiY3JvIjoyLCJwbGYiOjMsImV4cCI6MTcxNDI5MDYxMX0.U1fICk_BtHAXYrRfxAR126ruzeKy_IcUk2AKdeo7lfA",
        refreshToken: "",
        role: Role.Admin,
      };
      // 成功登录，返回accessToken、refreshToken和role
      return HttpResponse.json(
        {
          data: {
            accessToken: mockUser.accessToken,
            refreshToken: mockUser.refreshToken,
            context: {
              queue: "",
              roleQueue: Role.Admin,
              rolePlatform: mockUser.role,
            },
          },
          code: 0,
          msg: "",
        },
        { status: 200 },
      );
    },
  ),
];
