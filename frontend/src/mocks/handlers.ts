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
        username: "ganhao",
        password: "",
        accessToken:
          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOjgsInBpZCI6OSwicHJvIjozLCJjaWQiOjEsImNybyI6MiwicGxmIjozLCJleHAiOjE3MTQwMzI0MDZ9.p7DFIv7TLoyToyIQ6sNE7dRSOL09DbqWVmrEHtGZe6I",
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
              projectID: 1,
              projectRole: Role.Admin,
              platformRole: mockUser.role,
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
