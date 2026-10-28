import i18next from 'i18next'

const fallback = {
  timeout: '请求超时，请稍后重试',
  passwordUnavailable: '请切换登录方式或联系管理员',
  methodDisabled: '请切换登录方式',
  registrationDisabled: '暂未开放注册，请联系管理员',
  noProject: '暂无可用项目，请联系管理员',
  initializationFailed: '账号初始化失败，请联系管理员',
  identityInvalid: '账号信息异常，请联系管理员',
  sessionInvalid: '登录已失效，请重新登录',
  credentialsInvalid: '账号或密码错误',
  accountMissing: '未找到该账号，请检查输入',
  registrationRequired: '请先注册或使用统一身份登录',
  locked: '账号已锁定，请联系管理员',
  pending: '账号未激活，请联系管理员',
  disabled: '账号已停用，请联系管理员',
  banned: '账号已封禁，请联系管理员',
  alreadyRegistered: '该账号已注册，请直接登录',
  usePassword: '请使用账号和密码登录',
  invalidInput: '输入有误，请检查后重试',
  verificationFailed: '账号验证失败，请重试',
  operationUnavailable: '此操作不可用，请联系管理员',
  accountExists: '账号已存在，请直接登录',
  rateLimited: '操作过于频繁，请稍后重试',
  serviceUnavailable: '服务暂不可用，请稍后重试',
  network: '网络异常，请检查网络连接',
  failed: '操作失败，请稍后重试',
  resetPassword: '请联系管理员重置密码',
  loginServiceUnavailable: '登录服务暂不可用，请稍后重试',
  loginDisabled: '暂未开放登录，请联系管理员',
  usernameTooLong: '用户名最多 64 个字符',
  usernameFormat: '字母或数字开头，可含 ._- ',
  passwordTooLong: '密码过长，请缩短后重试',
} as const

export function authText(key: keyof typeof fallback): string {
  return i18next.t(`authFeedback.${key}`, { defaultValue: fallback[key] }) || fallback[key]
}
