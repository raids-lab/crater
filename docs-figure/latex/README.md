# LaTeX 算法与公式编译说明

本目录的 `.tex` 文件均为 fragment，假定宿主文档已加载下列宏包：

```latex
\usepackage{xeCJK}            % 中文支持，XeLaTeX/LuaLaTeX
\setCJKmainfont{Source Han Serif SC}
\setCJKsansfont{Source Han Sans SC}
\usepackage[ruled,vlined,linesnumbered]{algorithm2e}
\usepackage{amsmath, amssymb, mathtools}
\usepackage{xcolor}
\definecolor{algcomment}{HTML}{6B7280}
\SetCommentSty{\color{algcomment}\small\itshape}
```

## 编译入口（独立验证）

每个目录下提供 `_preview.tex`，可直接 `xelatex _preview.tex` 单独编译验证：

```bash
cd latex/algorithms && xelatex _preview.tex
cd latex/equations  && xelatex _preview.tex
```

## 嵌入论文

```latex
\begin{algorithm}[t]
\input{docs-figure/latex/algorithms/alg1-intent-router}
\end{algorithm}

\begin{equation}
\input{docs-figure/latex/equations/eq3-1-confidence-merge}
\end{equation}
```

> 注：每个 `.tex` 算法 fragment 已包含完整 `\begin{algorithm}…\end{algorithm}` 包装；
> 嵌入时按需保留或去掉外层环境，参考各 fragment 顶部注释。
