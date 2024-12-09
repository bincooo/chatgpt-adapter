## server-less 自动化程序下载

[下载地址](https://github.com/bincooo/chatgpt-adapter/tree/hel)

将下载后的`bin.zip`解压放入项目同级目录即可



ioc sdk 示例项目

[iocgo/sdk](https://www.github.com/iocgo/sdk)

## 执行前置

安装中间编译工具

```shell
go install ./cmd/iocgo

# or 

make install
```

## 使用

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

## 运行本项目三部曲 (linux / macos / window)

```shell
# 1: 安装
make install

# 2: 编译
make

# 3: 启动
./bin/linux/server config.yaml
```
