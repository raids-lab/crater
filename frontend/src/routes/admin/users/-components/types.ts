import { IUserAttributes } from '@/services/api/admin/user'

export interface AdminUserRow {
  id: number
  name: string
  role: string
  status: string
  banned: boolean
  bannedAt?: string
  extraBalance?: number
  periodFreeTotal?: number
  totalIssueAmount?: number
  totalAvailable?: number
  attributes: IUserAttributes
}
