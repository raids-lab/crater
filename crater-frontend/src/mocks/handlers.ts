// 1. Import the "HttpResponse" class from the library.
import { AccessMode, IAuthResponse, ILogin, Role } from "@/services/api/auth";
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
        username: "ganhao",
        password: "",
        accessToken:
          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aSI6NiwicWkiOjIzLCJ1biI6ImdhbmhhbyIsInFuIjoicS0yMyIsInJxIjozLCJycCI6MywiYW0iOjMsInBhIjoxLCJleHAiOjE3MTg4ODg3MDl9.-v3oQ_X9L1cwW96eRkk-SRPUGcJsRgHApM011BRTlWs",
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
              roleQueue: Role.Guest,
              rolePlatform: Role.Guest,
              accessQueue: AccessMode.NotAllowed,
              accessPublic: AccessMode.NotAllowed,
              space: "",
            },
            user: {
              name: "",
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
