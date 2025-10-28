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
import { zodResolver } from '@hookform/resolvers/zod'
import { Check, Loader2, X } from 'lucide-react'
import { FC, useState } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Form } from '@/components/ui/form'

import { TagsInput } from '@/components/form/tags-input'

import { ImageDefaultTags, UpdateImageTag } from '@/services/api/imagepack'

const formSchema = z.object({
  tags: z.array(
    z.object({
      value: z.string(),
      label: z.string().optional(),
    })
  ),
})

type FormSchema = z.infer<typeof formSchema>

interface TagsDialogProps {
  initialTags: string[]
  imageID: number
  description: string
  imageLink: string
  onSaveTags: (updateData: UpdateImageTag) => void
}

export const TagsDialog: FC<TagsDialogProps> = ({
  initialTags,
  imageID: id,
  description,
  imageLink,
  onSaveTags,
}) => {
  const [isSaving, setIsSaving] = useState(false)

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      tags: (initialTags || []).map((tag) => ({ value: tag, label: tag })),
    },
  })

  const handleSave = async (values: FormSchema) => {
    try {
      setIsSaving(true)
      const tags = values.tags.map((tag) => tag.value)
      await onSaveTags({ id, tags })
    } catch (error) {
      toast.error('保存标签失败' + error)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSave)} className="space-y-4">
        <DialogHeader>
          <DialogTitle>标签管理</DialogTitle>
        </DialogHeader>
        <DialogDescription className="space-y-4 pt-2">
          <div className="bg-muted/50 rounded-md px-4 py-3">
            <p className="text-muted-foreground">{description}</p>
            <p className="mt-1 font-medium break-all">『{imageLink}』</p>
          </div>
        </DialogDescription>

        <TagsInput
          form={form}
          tagsPath="tags"
          label="镜像标签"
          description="为镜像添加标签，便于分类和搜索"
          customTags={ImageDefaultTags}
        />

        <DialogFooter className="mt-4">
          <DialogClose asChild>
            <Button type="button" variant="outline">
              <X />
              取消
            </Button>
          </DialogClose>
          <Button type="submit" disabled={isSaving}>
            {isSaving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                保存中...
              </>
            ) : (
              <>
                <Check className="h-4 w-4" />
                确认
              </>
            )}
          </Button>
        </DialogFooter>
      </form>
    </Form>
  )
}
