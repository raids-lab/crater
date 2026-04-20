import { getDefaultStore } from 'jotai'

import { apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { globalJobUrl } from '@/utils/store'

import { IResponse } from '../types'

const store = getDefaultStore()
const JOB_URL = store.get(globalJobUrl)

export interface BillingSummaryResp {
  periodFreeBalance: number
  extraBalance: number
  totalAvailable: number
  lastIssuedAt?: string
  nextIssueAt?: string
  effectiveIssueAmount: number
  effectiveIssuePeriodMinutes: number
}

export interface UserBillingAccount {
  accountId: number
  accountName: string
  accountNickname: string
  periodFreeBalance: number
  extraBalance: number
  totalAvailable: number
  lastIssuedAt?: string
  nextIssueAt?: string
  effectiveIssueAmount: number
  effectiveIssuePeriodMinutes: number
}

export interface UserBillingSummary {
  userId: number
  username: string
  extraBalance: number
  periodFreeTotal: number
  totalIssueAmount: number
  totalAvailable: number
}

export interface AccountBillingConfig {
  issueAmount?: number
  issuePeriodMinutes?: number
  effectiveIssueAmount: number
  effectiveIssuePeriodMinutes: number
  lastIssuedAt?: string
}

export interface AccountBillingMember {
  issueAmountOverride?: number | null
  userId: number
  username: string
  nickname: string
  periodFreeBalance: number
  extraBalance: number
  totalAvailable: number
  lastIssuedAt?: string
  nextIssueAt?: string
  effectiveIssueAmount: number
  effectiveIssuePeriodMinutes: number
}

export interface BillingPriceResource {
  id: number
  name: string
  label: string
  unitPrice: number
}

export interface JobBillingInfo {
  jobName: string
  name: string
  billedPointsTotal: number
}

export interface AdjustUserExtraBalanceResp {
  userId: number
  username: string
  beforeBalance: number
  delta: number
  afterBalance: number
}

export interface AdjustUserExtraBalanceReq {
  delta: number
  reason?: string
}

export interface UpdateAccountBillingMemberIssueAmountReq {
  issueAmountOverride: number | null
}

export const apiContextBillingSummary = () =>
  apiV1Get<IResponse<BillingSummaryResp>>('context/billing/summary')

export const apiAdminAdjustUserExtraBalance = (userName: string, data: AdjustUserExtraBalanceReq) =>
  apiV1Post<IResponse<AdjustUserExtraBalanceResp>>(
    `admin/users/${userName}/billing/extra-balance`,
    data
  )

export const apiAdminGetUserBillingAccounts = (userName: string) =>
  apiV1Get<IResponse<UserBillingAccount[]>>(`admin/users/${userName}/billing/accounts`)

export const apiAdminGetUserBillingSummary = () =>
  apiV1Get<IResponse<UserBillingSummary[]>>('admin/users/billing/summary')

export const apiAdminGetAccountBillingConfig = (accountId: number) =>
  apiV1Get<IResponse<AccountBillingConfig>>(`admin/accounts/${accountId}/billing/config`)

export const apiGetAccountBillingConfig = (accountId: number) =>
  apiV1Get<IResponse<AccountBillingConfig>>(`accounts/${accountId}/billing/config`)

export const apiAdminUpdateAccountBillingConfig = (
  accountId: number,
  data: { issueAmount?: number; issuePeriodMinutes?: number }
) => apiV1Put<IResponse<AccountBillingConfig>>(`admin/accounts/${accountId}/billing/config`, data)

export const apiUpdateAccountBillingConfig = (
  accountId: number,
  data: { issueAmount?: number; issuePeriodMinutes?: number }
) => apiV1Put<IResponse<AccountBillingConfig>>(`accounts/${accountId}/billing/config`, data)

export const apiAdminGetAccountBillingMembers = (accountId: number) =>
  apiV1Get<IResponse<AccountBillingMember[]>>(`admin/accounts/${accountId}/billing/members`)

export const apiGetAccountBillingMembers = (accountId: number) =>
  apiV1Get<IResponse<AccountBillingMember[]>>(`accounts/${accountId}/billing/members`)

export const apiAdminUpdateAccountBillingMemberIssueAmount = (
  accountId: number,
  userId: number,
  data: UpdateAccountBillingMemberIssueAmountReq
) =>
  apiV1Put<IResponse<AccountBillingMember>>(
    `admin/accounts/${accountId}/billing/members/${userId}`,
    data
  )

export const apiUpdateAccountBillingMemberIssueAmount = (
  accountId: number,
  userId: number,
  data: UpdateAccountBillingMemberIssueAmountReq
) =>
  apiV1Put<IResponse<AccountBillingMember>>(`accounts/${accountId}/billing/members/${userId}`, data)

export const apiAdminResetAccountBillingBalance = (accountId: number) =>
  apiV1Post<IResponse<string>>(`admin/accounts/${accountId}/billing/reset`)

export const apiResetAccountBillingBalance = (accountId: number) =>
  apiV1Post<IResponse<string>>(`accounts/${accountId}/billing/reset`)

export const apiAdminResetAllBillingBalances = () =>
  apiV1Post<
    IResponse<{ accountsAffected: number; userAccountsAffected: number; issuedAt: string }>
  >('admin/system-config/billing/reset-all')

export const apiBillingPriceList = () =>
  apiV1Get<IResponse<BillingPriceResource[]>>('resources/billing/prices')

export const apiAdminUpdateResourceUnitPrice = (id: number, unitPrice: number) =>
  apiV1Put<IResponse<unknown>>(`admin/resources/${id}/billing/unit-price`, { unitPrice })

export const apiJobBillingList = () => apiV1Get<IResponse<JobBillingInfo[]>>(`${JOB_URL}/billing`)

export const apiJobAllBillingList = (days?: number) =>
  apiV1Get<IResponse<JobBillingInfo[]>>(`${JOB_URL}/billing/all`, {
    searchParams: days === undefined ? undefined : { days },
  })

export const apiAdminGetJobBillingList = (days: number) =>
  apiV1Get<IResponse<JobBillingInfo[]>>(`admin/${JOB_URL}/billing`, {
    searchParams: { days },
  })

export const apiAdminGetUserJobBillingList = (username: string, days: number = 30) =>
  apiV1Get<IResponse<JobBillingInfo[]>>(`admin/${JOB_URL}/billing/user/${username}`, {
    searchParams: { days },
  })

export const apiGetUserJobBillingList = (username: string, days: number = 30) =>
  apiV1Get<IResponse<JobBillingInfo[]>>(`${JOB_URL}/billing/user/${username}`, {
    searchParams: { days },
  })

export const apiJobBillingDetail = (jobName: string) =>
  apiV1Get<IResponse<JobBillingInfo>>(`${JOB_URL}/${jobName}/billing`)
