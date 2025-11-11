/**
 * 是否已在当前浏览器同意隐私政策
 * true 表示用户已在本设备上确认过，登录页可默认勾选
 */
import { atomWithStorage } from 'jotai/utils'

export const atomPrivacyAccepted = atomWithStorage<boolean>('crater.privacyAccepted', false)
