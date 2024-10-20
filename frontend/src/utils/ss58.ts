import { blake2AsU8a, encodeAddress } from "@polkadot/util-crypto";

// 将用户名转换为 SS58 地址
export function stringToSS58(name: string): string {
  // 1. 将用户名转换为字节数组 (UTF-8 编码)
  const usernameBytes = new TextEncoder().encode(name);

  // 2. 对字节数组进行 Blake2b 哈希运算，输出长度为 32 字节
  const hashedBytes = blake2AsU8a(usernameBytes, 256);

  // 3. 使用 SS58 编码生成地址，指定网络前缀，0 表示 Substrate 主网
  const ss58Address = encodeAddress(hashedBytes, 0);

  return ss58Address;
}
