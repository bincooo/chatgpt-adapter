
![image](https://github.com/user-attachments/assets/93be2041-8ebc-466a-9fd4-939f4f9082f2)

具体配置请 [查阅文档](https://bincooo.github.io/chatgpt-adapter)

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
