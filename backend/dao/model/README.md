# dao/model

在这里定义数据库模型。

## 如何为新枚举值添加 String() 方法，从而支持打印枚举值

> https://pkg.go.dev/golang.org/x/tools/cmd/stringer

首先安装 stringer 的二进制版本：

```bash
go install golang.org/x/tools/cmd/stringer
```

在 `./const.go` 中添加新枚举值类型后，编辑文档尾部的注释，添加新定义的枚举值：

```go
//go:generate stringer -type=Role,Status,AccessMode,JobStatus,{YOUR_NEW_ENUM_TYPE} -output=const_string.go
```

在项目根目录，运行 `go generate dao/model/const.go`，将会更新生成的代码。

打印枚举值：

```go
fmt.Printf("%v", role)
```