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
// i18n-processed-v1.1.0
import { useQueryClient } from '@tanstack/react-query'
import { UploadIcon } from 'lucide-react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import styled, { keyframes } from 'styled-components'

import { Button } from '@/components/ui/button'

import { apiXMLPut } from '@/services/client'

import { FILE_SIZE_LIMITS, isFileSizeExceeded } from '@/utils/file-size'

const progressPulse = keyframes`
  0% { left: -50%; }
  100% { left: 150%; }
`

// 上传进度指示器容器
const ProgressContainer = styled.div`
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 16px;
  width: 240px;
  transition: all 0.3s ease;
`

// 进度条基础样式
const ProgressBar = styled.div`
  height: 6px;
  background: #e0e0e0;
  border-radius: 3px;
  overflow: hidden;
  position: relative;
  margin-top: 8px;
`

// 上传进度条
const UploadProgress = styled.div<{ $progress: number }>`
  height: 100%;
  width: ${(props) => props.$progress}%;
  background: #3b82f6;
  border-radius: 3px;
  transition: width 0.3s ease;
`

// 处理状态指示器
const ProcessingIndicator = styled.div`
  height: 100%;
  width: 100%;
  position: relative;
  overflow: hidden;
`

// 处理动画条
const ProcessingBar = styled.div`
  position: absolute;
  top: 0;
  height: 100%;
  width: 40%;
  background: linear-gradient(90deg, rgba(16, 185, 129, 0.2), #10b981);
  animation: ${progressPulse} 1.5s ease-in-out infinite;
`

// 状态标签
const StatusLabel = styled.div`
  display: flex;
  justify-content: space-between;
  font-size: 14px;
  color: #4b5563;
  margin-bottom: 4px;
`

// 文件名
const FileName = styled.div`
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 8px;
`

interface ProgressProps {
  progress: number
  isProcessing: boolean
  fileName: string
}

const FloatingProgress: React.FC<ProgressProps> = ({ progress, isProcessing, fileName }) => {
  const { t } = useTranslation()

  return (
    <ProgressContainer>
      <StatusLabel>
        <span>{isProcessing ? t('fileUpload.processing') : t('fileUpload.uploading')}</span>
        <span>{isProcessing ? '' : `${progress}%`}</span>
      </StatusLabel>

      <ProgressBar>
        {isProcessing ? (
          <ProcessingIndicator>
            <ProcessingBar />
          </ProcessingIndicator>
        ) : (
          <UploadProgress $progress={progress} />
        )}
      </ProgressBar>

      <FileName>{fileName}</FileName>
    </ProgressContainer>
  )
}

interface FileUploadProps {
  uploadPath: string
  disabled: boolean
}

const FileUpload: React.FC<FileUploadProps> = ({ uploadPath, disabled }) => {
  const { t } = useTranslation()
  const [isUploading, setIsUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [isProcessing, setIsProcessing] = useState(false)
  const [currentFile, setCurrentFile] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const { files } = event.currentTarget
    if (files && files.length) {
      const Files = Array.from(files)

      // 检查文件大小限制
      const oversizedFiles = Files.filter((file) => isFileSizeExceeded(file.size))

      if (oversizedFiles.length > 0) {
        toast.error(t('fileUpload.fileSizeError', { maxSize: FILE_SIZE_LIMITS.MAX_FILE_SIZE_TEXT }))
        return
      }

      handleUpload(Files)
    }
  }

  const handleUpload = async (Files: File[]) => {
    if (Files.length == 0) return
    setIsUploading(true)
    setProgress(0)
    setCurrentFile(Files[0].name)
    const file = Files[0]
    const filename = file.name.split('/').pop()
    if (filename === undefined) return null
    const filedataBuffer = await Files[0].arrayBuffer()

    try {
      await apiXMLPut(`ss/${uploadPath}/${filename}`, filedataBuffer, (progressEvent) => {
        const loaded = progressEvent.loaded || 0
        const total = progressEvent.total || 1
        const percentCompleted = Math.round((loaded * 100) / total)
        setProgress(percentCompleted)
        if (percentCompleted === 100) {
          setIsProcessing(true)
        }
      })

      toast.success(t('fileUpload.successMessage'))
      void queryClient.invalidateQueries({
        queryKey: ['data', 'filesystem', uploadPath],
      })
      setTimeout(() => setProgress(0), 500)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Upload failed')
    } finally {
      setTimeout(() => {
        setIsUploading(false)
        setIsProcessing(false)
        setProgress(0)
      }, 1000)
    }
  }

  const handleClick = () => {
    if (fileInputRef.current) {
      fileInputRef.current.click()
    }
  }

  return (
    <>
      <Button
        variant="outline"
        size="icon"
        className="h-8 w-8"
        title={t('fileUpload.buttonTitle')}
        disabled={isUploading || disabled}
        onClick={handleClick}
      >
        <UploadIcon className="size-4" />
      </Button>
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        style={{ display: 'none' }}
      />
      {(isUploading || isProcessing) && currentFile && (
        <FloatingProgress progress={progress} isProcessing={isProcessing} fileName={currentFile} />
      )}
    </>
  )
}

export default FileUpload
