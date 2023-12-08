// 1. Import the "HttpResponse" class from the library.
import { http, HttpResponse } from "msw";

export const handlers = [
  http.get("/resource", () => {
    // 2. Return a mocked "Response" instance from the handler.
    return HttpResponse.text("Hello world!");
  }),
];
