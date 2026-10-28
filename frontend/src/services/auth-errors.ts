import { HTTPError, TimeoutError } from 'ky'

import { authText } from './auth-text.ts'
import {
  ERROR_ACCOUNT_DISABLED,
  ERROR_ACCOUNT_INITIALIZATION_FAILED,
  ERROR_ACCOUNT_LOCKED,
  ERROR_ACCOUNT_NOT_ACTIVATED,
  ERROR_IDENTITY_MAPPING_FAILED,
  ERROR_INVALID_CREDENTIALS,
  ERROR_LOGIN_METHOD_DISABLED,
  ERROR_MUST_REGISTER,
  ERROR_NO_PROJECT_ACCESS,
  ERROR_PASSWORD_LOGIN_UNAVAILABLE,
  ERROR_REGISTER_NOT_FOUND,
  ERROR_REGISTRATION_DISABLED,
  ERROR_RESOURCE_ALREADY_EXISTS,
  ERROR_TOKEN_EXPIRED,
  ERROR_TOKEN_INVALID,
  ERROR_USER_BANNED,
} from './error_code.ts'
import {
  LEGACY_ERROR_INVALID_CREDENTIALS,
  LEGACY_ERROR_LDAP_USER_NOT_FOUND,
  LEGACY_ERROR_LEGACY_TOKEN_NOT_SUPPORTED,
  LEGACY_ERROR_MUST_REGISTER,
} from './error_code_legacy.ts'
import type { IErrorResponse } from './types'

// Keep authentication hints short and user-facing. Technical diagnostics remain
// available in the API response instead of being printed beneath the form.
export function getAuthErrorMessage(error: unknown): string {
  if (error instanceof TimeoutError) {
    return authText('timeout')
  }
  if (error instanceof HTTPError) {
    const data = (error as HTTPError & { data?: IErrorResponse }).data
    switch (data?.code) {
      case ERROR_PASSWORD_LOGIN_UNAVAILABLE:
        return authText('passwordUnavailable')
      case ERROR_LOGIN_METHOD_DISABLED:
        return authText('methodDisabled')
      case ERROR_REGISTRATION_DISABLED:
        return authText('registrationDisabled')
      case ERROR_NO_PROJECT_ACCESS:
        return authText('noProject')
      case ERROR_ACCOUNT_INITIALIZATION_FAILED:
        return authText('initializationFailed')
      case ERROR_IDENTITY_MAPPING_FAILED:
        return authText('identityInvalid')
      case ERROR_TOKEN_INVALID:
      case ERROR_TOKEN_EXPIRED:
        return authText('sessionInvalid')
      case ERROR_INVALID_CREDENTIALS:
      case LEGACY_ERROR_INVALID_CREDENTIALS:
        return authText('credentialsInvalid')
      case ERROR_REGISTER_NOT_FOUND:
      case LEGACY_ERROR_LDAP_USER_NOT_FOUND:
        return authText('accountMissing')
      case ERROR_MUST_REGISTER:
      case LEGACY_ERROR_MUST_REGISTER:
        return authText('registrationRequired')
      case ERROR_ACCOUNT_LOCKED:
        return authText('locked')
      case ERROR_ACCOUNT_NOT_ACTIVATED:
        return authText('pending')
      case ERROR_ACCOUNT_DISABLED:
        return authText('disabled')
      case ERROR_USER_BANNED:
        return authText('banned')
      case ERROR_RESOURCE_ALREADY_EXISTS:
        return authText('alreadyRegistered')
      case LEGACY_ERROR_LEGACY_TOKEN_NOT_SUPPORTED:
        return authText('usePassword')
    }
    switch (error.response.status) {
      case 400:
        return authText('invalidInput')
      case 401:
        return authText('verificationFailed')
      case 403:
        return authText('operationUnavailable')
      case 409:
        return authText('accountExists')
      case 429:
        return authText('rateLimited')
      case 504:
        return authText('timeout')
      default:
        return authText('serviceUnavailable')
    }
  }
  if (error instanceof TypeError) {
    return authText('network')
  }
  return authText('failed')
}
