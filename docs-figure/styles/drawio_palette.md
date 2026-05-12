# drawio 样式约定

所有 `.drawio` 源文件遵循以下统一规范，方便后期一次性导出 PNG/SVG 时保持一致。

## 字体

- 主字体：**Source Han Sans SC**（思源黑体）；
- 字号：**9pt**（小五）；标题节点 10pt，泳道名称 10pt 加粗；
- 备用字体链（drawio Style → `fontFamily`）：
  `Source Han Sans SC, Noto Sans CJK SC, PingFang SC, Hiragino Sans GB, Microsoft YaHei, sans-serif`。

## 配色（与 matplotlib 一致）

| 用途 | 十六进制 | drawio fillColor |
|---|---|---|
| MOPS 主调（深青） | `#264653` | `#264653` |
| 次要框（中青） | `#2A9D8F` | `#2A9D8F` |
| 第三层（紫灰） | `#9C89B8` | `#9C89B8` |
| 强调红 | `#E76F51` | `#E76F51` |
| 通过/安全 | `#52B788` | `#52B788` |
| 警告 | `#F4A261` | `#F4A261` |
| 中性背景 | `#F5F6F8` | `#F5F6F8` |
| 边框深灰 | `#374151` | `#374151` |

## 节点统一样式

```
shape=rectangle;rounded=1;arcSize=6;
fillColor=<see above>;strokeColor=#374151;strokeWidth=1.2;
fontFamily=Source Han Sans SC;fontSize=9;fontColor=#1F2937;
align=center;verticalAlign=middle;
spacingTop=2;spacingBottom=2;spacingLeft=6;spacingRight=6;
```

## 连接线

```
endArrow=classic;endFill=1;strokeColor=#374151;strokeWidth=1.1;
fontFamily=Source Han Sans SC;fontSize=8;fontColor=#374151;
edgeStyle=orthogonalEdgeStyle;rounded=1;jettySize=auto;
```

## 时序图

- 演员（actor）间距：160px
- 生命线颜色：`#9CA3AF`，dashed
- 同步消息：实线 + 实心箭头；返回值：虚线 + 开口箭头
- 消息标号：`数字. 描述`（与参考图 5.3 一致）

## 分层背景

```
shape=swimlane;startSize=22;fillColor=#F5F6F8;strokeColor=#CBD5E1;
fontFamily=Source Han Sans SC;fontSize=10;fontStyle=1;
```

## 导出建议

- PNG：`Crop=on, Transparent=on, Border=4px, Scale=2x`
- SVG：`Embed Fonts=on, Math=off, Selection only=off`
