import { Button } from "@/components/ui/button";
import DataList, { DataItem } from "./DataList";
import { BookOpenIcon, CirclePlusIcon } from "lucide-react";

export const models: DataItem[] = [
  {
    id: 1,
    name: "GPT-4",
    tags: ["OpenAI", "语言模型", "先进", "强大"],
    desc: "OpenAI开发的强大语言模型，提供先进的语言理解和生成能力。",
    url: "https://openai.com",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 2,
    name: "BERT",
    tags: ["谷歌", "预训练", "自然语言处理"],
    desc: "谷歌开发的预训练语言模型，用于自然语言处理任务。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 3,
    name: "Transformer",
    tags: ["注意力机制", "深度学习", "自然语言处理", "计算机视觉"],
    desc: "一种基于注意力机制的深度学习模型，广泛应用于自然语言处理和计算机视觉。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 4,
    name: "DALL·E",
    tags: ["OpenAI", "图像生成", "文本描述"],
    desc: "OpenAI开发的图像生成模型，能够根据文本描述生成图像。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 5,
    name: "CLIP",
    tags: ["OpenAI", "多模态", "图像", "文本"],
    desc: "OpenAI开发的多模态模型，能够理解图像和文本之间的关系。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 6,
    name: "Whisper",
    tags: ["OpenAI", "语音识别", "语音转文本"],
    desc: "OpenAI开发的语音识别模型，能够将语音转换为文本。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 7,
    name: "T5",
    tags: ["谷歌", "文本到文本转换", "自然语言处理"],
    desc: "谷歌开发的文本到文本转换模型，用于多种自然语言处理任务。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 8,
    name: "RoBERTa",
    tags: ["Facebook", "预训练", "BERT改进"],
    desc: "Facebook开发的预训练语言模型，对BERT进行了改进。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 9,
    name: "ALBERT",
    tags: ["谷歌", "轻量级", "BERT改进"],
    desc: "谷歌开发的轻量级BERT模型，具有更少的参数和更快的训练速度。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 10,
    name: "XLNet",
    tags: ["卡内基梅隆大学", "谷歌", "自回归", "BERT改进"],
    desc: "卡内基梅隆大学和谷歌联合开发的自回归语言模型，对BERT进行了改进。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 11,
    name: "ELECTRA",
    tags: ["谷歌", "斯坦福大学", "生成对抗网络", "预训练"],
    desc: "谷歌和斯坦福大学联合开发的预训练语言模型，采用了生成对抗网络的思想。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 12,
    name: "DeBERTa",
    tags: ["微软", "改进版BERT", "解耦注意力机制"],
    desc: "微软开发的改进版BERT模型，采用了解耦注意力机制。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 13,
    name: "GPT-3",
    tags: ["OpenAI", "大型语言模型", "强大"],
    desc: "OpenAI开发的大型语言模型，具有强大的语言生成能力。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
  {
    id: 14,
    name: "GPT-2",
    tags: ["OpenAI", "语言模型", "前身"],
    desc: "OpenAI开发的语言模型，是GPT-3的前身。",
    username: "李亦龙",
    owner: { id: 12, name: "李亦龙", nickname: "李亦龙" },
  },
];

const Model = () => {
  return (
    <DataList
      items={models}
      title="模型"
      actionArea={
        <div className="flex flex-row gap-3">
          <Button variant="secondary">
            <BookOpenIcon />
            模型文档
          </Button>
          <Button>
            <CirclePlusIcon />
            导入模型
          </Button>
        </div>
      }
    />
  );
};

export default Model;
