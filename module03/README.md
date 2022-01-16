# 模块三作业：
- 构建本地镜像编写 Dockerfile 将练习 2.2 编写的 httpserver 容器化
- 将镜像推送至 docker 官方镜像仓库
- 通过 docker 命令本地启动 httpserver
- 通过 nsenter 进入容器查看 IP 配置

- 构建本地镜像

    ```
    docker build . -t httpserver:0.0.1
    ```

- 编写 Dockerfile 将练习 2.2 编写的 httpserver 容器化（请思考有哪些最佳实践可以引入到 Dockerfile 中来）

  > [Dockerfile](./Dockerfile)
    1. 使用 Multi-stage build 减少层数
    2. 合并Run命令 减少层数


- 将镜像推送至 Docker 官方镜像仓库

    ```
    docker push httpserver:0.0.1
    ```

- 通过 Docker 命令本地启动 httpserver

    ```
    docker run -d httpserver：0.0.1
    ```

- 通过 nsenter 进入容器查看 IP 配置

    ```
    PID=$(docker inspect --format "{{ .State.Pid }}" nervous_shannon)
    ```
  ```
  nsenter -t $PID -n ip a
  ```

    ```
- 1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00    inet 127.0.0.1/8 scope host lo       valid_lft forever  preferred_lft forever
- 21: eth0@if22: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff link-netnsid 0    inet 172.17.0.2/16 brd 172.17.255.255 scope global eth0       valid_lft forever preferred_lft forever
    ```