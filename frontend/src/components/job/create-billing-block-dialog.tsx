import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface CreateBillingBlockDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateBillingBlockDialog({ open, onOpenChange }: CreateBillingBlockDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>当前额度不足，无法提交作业</DialogTitle>
        </DialogHeader>
        <div className="text-muted-foreground space-y-2 text-sm">
          <p>当前账户免费额度和额外额度都小于等于 0，提交作业已被拦截。</p>
          <p>请联系管理员发放免费额度或额外点数后再试。</p>
        </div>
        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>知道了</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
