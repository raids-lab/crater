import React, { forwardRef } from 'react';
import { useFormContext, Controller } from 'react-hook-form';
import { Input } from '@/components/ui/input';

interface TagsInputProps {
  name: string;
}

const TagsInput = forwardRef<HTMLInputElement, TagsInputProps>(({ name }, ref) => {
  const { control, setError, clearErrors } = useFormContext();

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>, onChange: (value: string[]) => void) => {
    const tags = event.target.value.split(',').map(tag => tag.trim());
    if (tags.some(tag => tag.length === 0)) {
      setError(name, { type: 'manual', message: '标签不能为空' });
    } else {
      clearErrors(name);
      onChange(tags);
    }
  };

  return (
    <Controller
      name={name}
      control={control}
      render={({ field, fieldState }) => (
        <>
          <Input
            type="text"
            ref={ref}
            value={Array.isArray(field.value) ? field.value.join(', ') : ''}
            onChange={(e) => handleChange(e, field.onChange)}
            //placeholder="请输入标签，用逗号分隔"
          />
          {fieldState.error && <p className="text-red-500">{fieldState.error.message}</p>}
        </>
      )}
    />
  );
});

export default TagsInput;