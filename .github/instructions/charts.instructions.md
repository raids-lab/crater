---
applyTo: "charts/**"
---

# Helm Chart 开发与审查规范

此文档针对 `charts/` 目录下的 Helm Chart 配置及相关文档变更提供指引。

## 核心规范 (Core Requirements)
- **版本晋升**: 凡涉及 `values.yaml` 配置逻辑、模板修改或依赖更新，必须同步增加 `Chart.yaml` 中的 `version` 版本号。
- **配置同步**: 
  - 修改 `values.yaml` 参数后，必须在对应的 `README.md` 中同步更新参数说明。
  - 新增配置参数必须包含完整的含义描述、默认值及使用示例。
- **注释规范**: `values.yaml` 中新增的配置项必须包含准确、清晰且能够有效辅助用户理解其用途的**英文注释**。

## 优化建议 (Optimization Suggestions)
- **文档自动化**: 建议通过 `helm-docs` 维护文档。若发现 README 手动维护且格式混乱，应建议使用该工具。
- **配置最佳实践**: 针对变量命名语义、注释清晰度或资源限额（Resources Quota）设置提出优化建议。
- **模板优雅性**: 发现更简洁或健壮的 Helm 模板编写方式时提出改进方案。
