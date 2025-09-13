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
import React from 'react'
import { useTranslation } from 'react-i18next'
import styled, { keyframes } from 'styled-components'

const progressPulse = keyframes`
  0% { left: -50%; }
  100% { left: 150%; }
`

// 下载进度指示器容器
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

// 下载进度条
const DownloadProgress = styled.div<{ $progress: number }>`
  height: 100%;
  width: ${(props) => props.$progress}%;
  background: #10b981;
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

interface DownloadProgressProps {
  progress: number
  isProcessing: boolean
  fileName: string
}

export const FileDownloadProgress: React.FC<DownloadProgressProps> = ({
  progress,
  isProcessing,
  fileName,
}) => {
  const { t } = useTranslation()

  return (
    <ProgressContainer>
      <StatusLabel>
        <span>{isProcessing ? t('fileDownload.processing') : t('fileDownload.downloading')}</span>
        <span>{isProcessing ? '' : `${progress}%`}</span>
      </StatusLabel>

      <ProgressBar>
        {isProcessing ? (
          <ProcessingIndicator>
            <ProcessingBar />
          </ProcessingIndicator>
        ) : (
          <DownloadProgress $progress={progress} />
        )}
      </ProgressBar>

      <FileName>{fileName}</FileName>
    </ProgressContainer>
  )
}
