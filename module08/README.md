# 模块8作业：
- 将httpserver部署到kubernetes集群中，做到如下要求：
- 优雅启动
- 优雅中止
- 资源需求和QoS
- 探活
- 日常运维需求，日志等级
- 配置和代码分离

- 镜像管理
- 推送镜像

推送镜像先把，httpserver打成image，推入docker hub，这里用的Azure容器注册表服务。
默认是私有镜像仓库，所以这里要用docker login登录一次。

`root@ubuntuguest:~/images# docker login cloudnativestg1.azurecr.io
Username: cloudnativestg1
Password:
WARNING! Your password will be stored unencrypted in /root/snap/docker/1458/.docker/config.json.
Configure a credential helper to remove this warning. See
https://docs.docker.com/engine/reference/commandline/login/#credentials-store
root@ubuntuguest:~/images# docker push cloudnativestg1.azurecr.io/httpserver:1.0
The push refers to repository [cloudnativestg1.azurecr.io/httpserver]
c1a8e2b858f1: Pushing  67.07kB/6.077MBd31505fd5050: Preparing`
推上去之后看下

ImagePullSecrets
https://kubernetes.io/zh/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod

`cat /root/snap/docker/1458/.docker/config.json | base64 -w0`


```
apiVersion: v1data:  .dockerconfigjson: ewoJImF1dGhzIjogewoJCSJjbG91ZG5hdGl2ZXN0ZzEuYXp1cmVjci5pbyI6IHsKCQkJImF1dGgiOiAiWTJ....yIKCQl9Cgl9Cn0=kind: Secretmetadata:  name: cloudnativetype: kubernetes.io/dockerconfigjson
```

- 第一部分
- Deployment
- deployment controller生成rs，rs controller去启动配置。
```
apiVersion: apps/v1kind: Deploymentmetadata:  name: httpserver  labels:    app: httpserverspec:  replicas: 1  selector:    matchLabels:      app: httpserver  template:    metadata:      labels:        app: httpserver    spec:      imagePullSecrets:      - name: cloudnative      containers:      - name: httpserver        image: fmeng.azurecr.io/httpserver:1.0        ports:        - containerPort: 8080
```
- 优雅启动
- 1. 一个服务刚启动，可能会有一堆东西要加载，比如需要load大量数据等等，此时程序启动了，但是并未准备好处理外部请求，所以利用一些探针来测试程序是否启动完成，然后进入下一步
- Probe 探针
- 1. https://kubernetes.io/zh/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
- Readiness Probe
- ```
readinessProbe:    httpGet:      path: /healthz      port: 8080      scheme: HTTP    initialDelaySeconds: 5    periodSeconds: 3
```
initialDelaySeconds 字段告诉 kubelet 在执行第一次探测前应该等待 3 秒 periodSeconds 字段指定了 kubelet 每隔 3 秒执行一次存活探测

Startup Probe 慢启动容器

应用程序在启动时需要较多的初始化时间。 要不影响对引起探测死锁的快速响应，这种情况下，设置存活探测参数是要技巧的。 技巧就是使用一个命令来设置启动探测，针对HTTP 或者 TCP 检测，可以通过设置 failureThreshold * periodSeconds 参数来保证有足够长的时间应对糟糕情况下的启动时间。

startupProbe:  httpGet:    path: /healthz    port: liveness-port  failureThreshold: 30  periodSeconds: 10

应用程序将会有最多 5 分钟(30 * 10 = 300s) 的时间来完成它的启动

livenessProbe
```
livenessProbe:  exec:    command:    - cat    - /tmp/healthy  initialDelaySeconds: 5  periodSeconds: 5---livenessProbe:  httpGet:    path: /healthz    port: 8080    httpHeaders:    - name: Custom-Header      value: Awesome  initialDelaySeconds: 3  periodSeconds: 3
```

init container
通常pod有一些初始化操作，创建文件夹，初始化磁盘，检查某些依赖服务是不是正常，这些操作放在代码中会污染代码，写在启动命令中不方便管理，出问题也不方便排查，更优雅的方式是使用init container

https://kubernetes.io/zh/docs/concepts/workloads/pods/init-containers/

postStart

https://kubernetes.io/zh/docs/tasks/configure-pod-container/attach-handler-lifecycle-event/#%E5%AE%9A%E4%B9%89-poststart-%E5%92%8C-prestop-%E5%A4%84%E7%90%86%E5%87%BD%E6%95%B0

postStart 操作执行完成之前，kubelet 会锁住容器，不让应用程序的进程启动，只有在 postStart 操作完成之后容器的状态才会被设置成为 RUNNING。

优雅中止

https://kubernetes.io/zh/docs/tasks/configure-pod-container/attach-handler-lifecycle-event/

有优雅启动，就有优雅中止，我们先看中止的流程：

容器终止流程
- 1. Pod 被删除，状态置为 Terminating。
- 2. kube-proxy 更新转发规则，将 Pod 从 service 的 endpoint 列表中摘除掉，新的流量不再转发到该 Pod。
- 3. 如果 Pod 配置了 preStop Hook ，将会执行。
- 4. kubelet 对 Pod 中各个 container 发送 SIGTERM 信号以通知容器进程开始优雅停止。
- 5. 等待容器进程完全停止，如果在  terminationGracePeriodSeconds 内 (默认 30s) 还未完全停止，就发送 SIGKILL 信号强制杀死进程。
- 6. 所有容器进程终止，清理 Pod 资源。

SIGTERM & SIGKILL
9 -  SIGKILL 强制终端15 - SIGTEM 请求中断
容器内的进程不能用SIGTEM中断的，都不能算优雅中止，所以这里有个常见问题，为什么有些容器SIGTEM信号不起作用。
如果容器启动入口使用了  shell，比如使用了类似  /bin/sh -c my-app  或  /docker-entrypoint.sh  这样的 ENTRYPOINT 或 CMD，这就可能就会导致容器内的业务进程收不到 SIGTERM 信号，原因是:
容器主进程是 shell，业务进程是在 shell 中启动的，成为了 
shell 进程的子进程。shell 进程默认不会处理 SIGTERM 信号，自己不会退出，也不会将信号传递给子进程，导致业务进程不会触发停止逻辑。
当等到 K8S 优雅停止超时时间 (terminationGracePeriodSeconds，默认 30s)，发送 SIGKILL 强制杀死 shell 及其子进程。


资源需求和QoSQoS 服务质量GuaranteedBurstableBestEffortQoS 类为 Guaranteed 的 Pod：Pod 中的每个容器都必须指定内存限制和内存请求。对于 Pod 中的每个容器，内存限制必须等于内存请求。Pod 中的每个容器都必须指定 CPU 限制和 CPU 请求。对于 Pod 中的每个容器，CPU 限制必须等于 CPU 请求。YAML

 limits:    memory: "200Mi"    cpu: "700m"  requests:    memory: "200Mi"    cpu: "700m"

QoS 类为 Burstable 的 PodPod 不符合 Guaranteed QoS 类的标准。Pod 中至少一个容器具有内存或 CPU 请求。

resources:  limits:    memory: "200Mi"  requests:    memory: "100Mi"

QoS 类为 BestEffort 的 Pod没有设置内存和 CPU 限制或请求

配置和代码分离，日志等级

https://kubernetes.io/zh/docs/concepts/configuration/configmap/

```
apiVersion: v1kind: ConfigMapmetadata:  name: game-demodata:  # 类属性键；每一个键都映射到一个简单的值  player_initial_lives: "3"  ui_properties_file_name: "user-interface.properties"  loglevel: debug  # 类文件键  game.properties: |    enemy.types=aliens,monsters    player.maximum-lives=5      user-interface.properties: |    color.good=purple    color.bad=yellow    allow.textmode=true    volumes:# 你可以在 Pod 级别设置卷，然后将其挂载到 Pod 内的容器中- name: config  configMap:    # 提供你想要挂载的 ConfigMap 的名字    name: game-demoenv:# 定义环境变量- name: PLAYER_INITIAL_LIVES # 请注意这里和 ConfigMap 中的键名是不一样的  valueFrom:    configMapKeyRef:      name: game-demo           # 这个值来自 ConfigMap      key: player_initial_lives # 需要取值的键- name: UI_PROPERTIES_FILE_NAME  valueFrom:    configMapKeyRef:      name: game-demo      key: ui_properties_file_name
```

Downward API

https://kubernetes.io/zh/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/

下面这些信息可以通过环境变量和 downwardAPI 卷提供给容器：能通过 fieldRef 获得的：metadata.name - Pod 名称metadata.namespace - Pod 名字空间metadata.uid - Pod 的 UIDmetadata.labels['<KEY>'] - Pod 标签 <KEY> 的值 (例如, metadata.labels['mylabel']）metadata.annotations['<KEY>'] - Pod 的注解 <KEY> 的值（例如, metadata.annotations['myannotation']）能通过 resourceFieldRef 获得的：容器的 CPU 约束值容器的 CPU 请求值容器的内存约束值容器的内存请求值容器的巨页限制值（前提是启用了 DownwardAPIHugePages特性门控）容器的巨页请求值（前提是启用了 DownwardAPIHugePages特性门控）容器的临时存储约束值容器的临时存储请求值

此外，以下信息可通过 downwardAPI 卷从 fieldRef 获得：metadata.labels - Pod 的所有标签，以 label-key="escaped-label-value" 格式显示，每行显示一个标签metadata.annotations - Pod 的所有注解，以 annotation-key="escaped-annotation-value" 格式显示，每行显示一个标签以下信息可通过环境变量获得：status.podIP - 节点 IPspec.serviceAccountName - Pod 服务帐号名称, 版本要求 v1.4.0-alpha.3spec.nodeName - 节点名称, 版本要求 v1.4.0-alpha.3status.hostIP - 节点 IP, 版本要求 v1.7.0-alpha.1

其他revisionHistoryLimithttps://stackoverflow.com/questions/60699867/about-k8s-deployments-spec-revisionhistorylimitprogressDeadlineSeconds标示 Deployment 进展停滞之前，需要等待所给的时长。检测此状况的一种方法是在 Deployment 规约中指定截止时间参数： （[.spec.progressDeadlineSeconds]（#progress-deadline-seconds））。 .spec.progressDeadlineSeconds 给出的是一个秒数值，Deployment 控制器在（通过 Deployment 状态）


`apiVersion: apps/v1kind: Deploymentmetadata:  labels:    app: httpserver  name: httpserverspec:  progressDeadlineSeconds: 600  replicas: 2  revisionHistoryLimit: 10  selector:    matchLabels:      app: httpserver  strategy:    rollingUpdate:      maxSurge: 25%      maxUnavailable: 25%    type: RollingUpdate  template:    metadata:      creationTimestamp: null      labels:        app: httpserver    spec:      containers:        - env:            - name: httpport              valueFrom:                configMapKeyRef:                  key: httpport                  name: myenv          image: cloudnativestg1.azurecr.io/httpserver:1.1          imagePullPolicy: IfNotPresent          livenessProbe:            failureThreshold: 3            httpGet:              path: /healthz              port: 8080              scheme: HTTP            initialDelaySeconds: 5            periodSeconds: 10            successThreshold: 1            timeoutSeconds: 1          name: httpserver          readinessProbe:            failureThreshold: 3            httpGet:              path: /healthz              port: 8080              scheme: HTTP            initialDelaySeconds: 5            periodSeconds: 10            successThreshold: 1            timeoutSeconds: 1          resources:            limits:              cpu: 200m              memory: 100Mi            requests:              cpu: 20m              memory: 20Mi          terminationMessagePath: /dev/termination-log          terminationMessagePolicy: File      dnsPolicy: ClusterFirst      imagePullSecrets:        - name: cloudnative      restartPolicy: Always      schedulerName: default-scheduler      securityContext: {}      terminationGracePeriodSeconds: 30`