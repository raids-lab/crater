/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// 常用百家姓
const SURNAMES = [
  '李',
  '王',
  '张',
  '刘',
  '陈',
  '杨',
  '赵',
  '黄',
  '周',
  '吴',
  '徐',
  '孙',
  '胡',
  '朱',
  '高',
  '林',
  '何',
  '郭',
  '马',
  '罗',
  '梁',
  '宋',
  '郑',
  '谢',
  '韩',
  '唐',
  '冯',
  '于',
  '董',
  '萧',
  '程',
  '曹',
  '袁',
  '邓',
  '许',
  '傅',
  '沈',
  '曾',
  '彭',
  '吕',
]

// 中文数字一到十
const NUMBERS = ['一', '二', '三', '四', '五', '六', '七', '八', '九', '十']

// 常见物品名称
const OBJECTS = [
  '花',
  '树',
  '草',
  '叶',
  '山',
  '水',
  '云',
  '风',
  '雨',
  '雪',
  '石',
  '土',
  '星',
  '月',
  '天',
  '地',
  '江',
  '河',
  '湖',
  '海',
  '鸟',
  '鱼',
  '虎',
  '龙',
  '马',
  '牛',
  '羊',
  '鹿',
  '松',
  '竹',
]

/**
 * 根据用户名生成一个简单的数字哈希值
 */
function hashUsername(username: string): number {
  let hash = 0
  for (let i = 0; i < username.length; i++) {
    hash = (hash << 5) - hash + username.charCodeAt(i)
    hash |= 0 // 转换为32位整数
  }
  return Math.abs(hash)
}

/**
 * 生成用户的假名
 */
export function generatePseudonym(username: string): string {
  const hash = hashUsername(username)

  const surnameIndex = hash % SURNAMES.length
  const numberIndex = (hash >> 8) % NUMBERS.length
  const objectIndex = (hash >> 16) % OBJECTS.length

  return SURNAMES[surnameIndex] + NUMBERS[numberIndex] + OBJECTS[objectIndex]
}

// 用户名到假名的缓存映射
const userPseudonymMap = new Map<string, string>()

/**
 * 获取用户的假名，使用缓存避免重复计算
 */
export function getUserPseudonym(username: string = ''): string {
  if (!userPseudonymMap.has(username)) {
    userPseudonymMap.set(username, generatePseudonym(username))
  }
  return userPseudonymMap.get(username)!
}
