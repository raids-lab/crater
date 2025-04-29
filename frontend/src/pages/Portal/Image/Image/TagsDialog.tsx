import { Input } from "@/components/ui/input";
import { Plus, X, Loader2, Check } from "lucide-react";
import { FC, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  DialogClose,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { UpdateImageTag } from "@/services/api/imagepack";

interface TagsDialogProps {
  initialTags: string[];
  imageID: number;
  onSaveTags: (updateData: UpdateImageTag) => void;
}

export const TagsDialog: FC<TagsDialogProps> = ({
  initialTags,
  imageID: id,
  onSaveTags,
}) => {
  const [tags, setTags] = useState<string[]>(initialTags || []);
  const [newTag, setNewTag] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleAddTag = () => {
    const trimmedTag = newTag.trim();
    if (trimmedTag && !tags.includes(trimmedTag)) {
      setTags([...tags, trimmedTag]);
      setNewTag("");
      inputRef.current?.focus();
    } else if (tags.includes(trimmedTag)) {
      toast.info("标签已存在");
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      handleAddTag();
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setTags(tags.filter((tag) => tag !== tagToRemove));
  };

  const handleSave = async () => {
    try {
      setIsSaving(true);
      await onSaveTags({ id, tags });
      toast.success("标签保存成功");
    } catch (error) {
      toast.error("保存标签失败" + error);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <>
      <DialogHeader>
        <DialogTitle>管理管理</DialogTitle>
      </DialogHeader>

      <div className="my-4 flex items-center space-x-2">
        <Input
          ref={inputRef}
          value={newTag}
          onChange={(e) => setNewTag(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="添加新标签..."
          className="flex-1"
        />
        <Button onClick={handleAddTag} size="icon" disabled={!newTag.trim()}>
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex max-h-[200px] flex-wrap gap-2 overflow-y-auto">
        {tags.length === 0 ? (
          <p className="text-muted-foreground text-sm">
            暂无标签，请添加新标签
          </p>
        ) : (
          tags.map((tag) => (
            <Badge
              key={tag}
              variant="secondary"
              className="group px-3 py-1.5 text-sm"
            >
              {tag}
              <button
                onClick={() => handleRemoveTag(tag)}
                className="hover:bg-muted ml-2 rounded-full p-0.5 transition-colors"
              >
                <X className="h-3 w-3" />
                <span className="sr-only">删除标签 {tag}</span>
              </button>
            </Badge>
          ))
        )}
      </div>

      <DialogFooter className="mt-4">
        <DialogClose asChild>
          <Button variant="outline">
            <X />
            取消
          </Button>
        </DialogClose>
        <Button onClick={handleSave} disabled={isSaving}>
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
    </>
  );
};
