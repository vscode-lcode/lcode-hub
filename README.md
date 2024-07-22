## Archive

基于 Bash 的 webdav 始终不够可靠, 还是在服务器上安装一个程序好了, 转向 [lcode](https://github.com/vscode-lcode/lcode)

## 简介

webdav server over bash

一个基于 `bash` 的 `webdav server`, 文件的列出用的是 `ls`, 读取写入用的 `dd`, 所以只要有这三个就可以运行了

#### 用途/目标

使用本地编辑器编辑服务器文件

## 使用/安装

#### 用法 (在服务器上)

直接调用

```sh
<>/dev/tcp/127.0.0.1/4349 bash -i
```

## 安装/设置 (本机)

### 下载 (暂无)

[点击前往 Releases 下载](https://github.com/vscode-lcode/lcode-hub/releases)

其他架构的请从源码 build 或提出 issues 添加需要的架构

### 从源码 build

```sh
make build
# the binanry
./lcode-hub
# output
lcode-hub is running on 127.0.0.1:4349
```

### 设置 ssh config

```conf
# ~/.ssh/config
# config for lcode
Host *
  # 转发 hub 端口
  RemoteForward 127.0.0.1:4349 127.0.0.1:4349
  # 避免多次端口转发
  ControlMaster auto
  ControlPath /tmp/ssh_control_socket_%lcodeh_%p_%r
  # 启动lcode-hub
  LocalCommand ~/go/bin/lcode-hub >/dev/null &
  PermitLocalCommand yes
```

### 进阶设置/设计/FAQ

#### 是如何确定服务器的唯一性?

使用 `hostname`, 获取来源是 `/proc/sys/kernel/hostname`

#### 是如何区分同一台服务器上的多个`lcode`

这是在 `client` 内存数据库表中维护的, 每个`lcode`连接和`workdir`都会记录在该表中

如果请求的路径在该表中不存在, 那么就会返回 403 错误

注: 内存数据库, 每次启动都会清空

## 一些开发中用到的技巧

#### 将 tcp socket 用作管道

```sh
echo 0 | 4>&0 5>/dev/tcp/127.0.0.1/4349 3> >(>&5 cat <&4) cat <&5 | cat
```
