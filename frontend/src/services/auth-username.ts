import { z } from 'zod'

import { authText } from './auth-text.ts'

// Usernames also form Kubernetes resource names; keep login and signup rules aligned.
export const usernameSchema = z
  .string()
  .min(1, { message: authText('usernameRequired') })
  .max(20, { message: authText('usernameTooLong') })
  .regex(/^[a-z][a-z0-9-]*[a-z0-9]$/, { message: authText('usernameFormat') })
