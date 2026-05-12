# MOps 论文初稿 (2026-05-11)

**题目**：面向智算平台的多智能体运维编排框架 MOps 的设计与实现
**语种**：中文（摘要附英文版）
**版本**：Draft v2
**日期**：2026 年 5 月 11 日
**总字数**：约 25,000 字

## 目录与章节统计

| 文件 | 章节 | 预估字数 |
|---|---|---|
| `00-abstract.md` | 摘要 + Abstract | ~1,200 |
| `01-introduction.md` | 第 1 章 绪论 | ~4,000 |
| `02-related-work.md` | 第 2 章 相关概念与技术 | ~3,500 |
| `03-system-design.md` | 第 3 章 MOps 框架设计 | ~6,000 |
| `04-implementation.md` | 第 4 章 系统实现 | ~7,000 |
| `05-experiment.md` | 第 5 章 实验与评估 | ~5,000 |
| `06-conclusion.md` | 第 6 章 总结与展望 | ~1,500 |
| **合计** | | **~28,200** |

## 实验数据来源

- **Exp30 三方法对比**：`results/exp30-qwen-max-per-scenario-20260505/`
  - qwen-max 模型，MOPS/PS/React 三方法，30 场景，7 个有效评分
- **DeepSeek 扩展验证**：`results/sample4-deepseekv4pro-oldkey-historyguard-20260505/`
  - deepseek-v4-pro 模型，MOPS 方法，4 场景

## 与前版 (thesis/) 的主要差异

1. 实验数据全部使用真实运行结果（不再使用预期值）
2. 新增第 5 章基于真实数据的细粒度分析与典型案例
3. 第 4 章增加了 Crater-Bench 评测基础设施的详细描述
4. 更新了所有量化结论以匹配真实实验结果
5. 减少了引用的绝对数量，聚焦核心相关工作

## 拼接为单文件

```bash
cat 00-abstract.md 01-introduction.md 02-related-work.md \
    03-system-design.md 04-implementation.md \
    05-experiment.md 06-conclusion.md > thesis-full.md
```

## 待补充

- 封面信息（作者、学号、指导老师、院系等）
- 图表（文中标注了图 3-1、表 1-1 至表 5-5 等位置，可据此生成）
- 参考文献列表（文中引用编号 [1]-[54]，需整理为正式格式）
- 学校模板要求的程序页、声明页等
