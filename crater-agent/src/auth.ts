import { timingSafeEqual } from "node:crypto";

export function matchesBearerToken(
  header: string | undefined,
  expected: string,
): boolean {
  if (!header?.startsWith("Bearer ")) {
    return false;
  }
  const received = Buffer.from(header.slice("Bearer ".length));
  const wanted = Buffer.from(expected);
  return received.length === wanted.length && timingSafeEqual(received, wanted);
}
