## ioc sdk 示例项目

[iocgo/sdk](https://www.github.com/iocgo/sdk)

### 执行前置

安装中间编译工具
```shell
go install ./cmd/iocgo

# or 

make install
```
### 使用

正常指令附加
```shell
# ----- go build ------ #
# 原指令 #
go build ./main.go

# 附加指令 #
go build -toolexec iocgo ./main.go


# ----- go run ------ #
# 原指令 #
go run ./main.go

# 附加指令 #
go run -toolexec iocgo ./main.go
```

其它`go`指令同理


### 运行本项目三部曲 (linux / macos)

```shell
make install

make build

./server -h
```

### 其它

避免编译缓存污染其它项目，推荐使用 `go mod vendor` 命令来独立管理依赖


