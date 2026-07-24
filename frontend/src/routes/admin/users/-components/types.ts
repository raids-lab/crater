import { IUserAttributes, IUserBanRestrictions } from '@/services/api/admin/user'

export interface AdminUserRow {
  id: number
  name: string
  role: string
  status: string
  banned: boolean
  permanentBanned: boolean
  bannedTimestamp?: string
  banRestrictions: IUserBanRestrictions
  extraBalance?: number
  periodFreeTotal?: number
  totalIssueAmount?: number
  totalAvailable?: number
  attributes: IUserAttributes
}
