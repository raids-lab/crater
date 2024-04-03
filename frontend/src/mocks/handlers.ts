// 1. Import the "HttpResponse" class from the library.
import { IAuthResponse, ILogin, Role } from "@/services/api/auth";
import { logger } from "@/utils/loglevel";
import { http, HttpResponse } from "msw";

const baseURL = import.meta.env.VITE_API_BASE_URL;

export const handlers = [
  http.post<never, ILogin, IAuthResponse>(
    baseURL + `login`,
    async ({ request }) => {
      const { username, password } = await request.json();
      logger.info(`login with username: ${username}, password: ${password}`);
      const mockUser = {
        username: "lyl",
        password: "000000",
        accessToken: "mockAccessToken123",
        refreshToken: "mockRefreshToken456",
        role: Role.Admin,
      };
      // 成功登录，返回accessToken、refreshToken和role
      return HttpResponse.json(
        {
          accessToken: mockUser.accessToken,
          refreshToken: mockUser.refreshToken,
          context: {
            projectID: 1,
            projectRole: Role.Admin,
            platformRole: mockUser.role,
          },
          projects: [
            {
              id: 1,
              name: "project1",
              role: Role.Admin,
              isPersonal: true,
            },
          ],
        },
        { status: 200 },
      );
    },
  ),
];
