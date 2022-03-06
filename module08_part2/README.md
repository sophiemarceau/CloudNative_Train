# 模块8作业：
-除了将 httpServer 应用优雅的运行在 Kubernetes 之上，我们还应该考虑如何将服务发布给对内和对外的调用方。
来尝试用 Service, Ingress 将你的服务发布给集群外部的调用方吧。
- 在第一部分的基础上提供更加完备的部署 spec，包括（不限于）：
- Service
- 可以考虑的细节
- 探活
- 如何确保整个应用的高可用。
- 如何通过证书保证 httpServer 的通讯安全。



Service

-https://kubernetes.io/zh/docs/concepts/services-networking/service/

-四层负载均衡，由kube-proxy来配置，早期默认iptable, 在1.11之后变成ipvs，这两个都是netfilter框架的产物
-在 ipvs 模式下，kube-proxy 监视 Kubernetes 服务和端点，调用 netlink 接口相应地创建 IPVS 规则， 并定期将 IPVS 规则与 Kubernetes 服务和端点同步。 该控制循环可确保IPVS 状态与所需状态匹配。访问服务时，IPVS 将流量定向到后端Pod之一。
-IPVS代理模式基于类似于 iptables 模式的 netfilter 挂钩函数， 但是使用哈希表作为基础数据结构，并且在内核空间中工作。 这意味着，与 iptables 模式下的 kube-proxy 相比，IPVS 模式下的 kube-proxy 重定向通信的延迟要短，并且在同步代理规则时具有更好的性能。 与其他代理模式相比，IPVS 模式还支持更高的网络流量吞吐量。
-IPVS 提供了更多选项来平衡后端 Pod 的流量。 这些是：rr：轮替（Round-Robin）lc：最少链接（Least Connection），即打开链接数量最少者优先dh：目标地址哈希（Destination Hashing）sh：源地址哈希（Source Hashing）sed：最短预期延迟（Shortest Expected Delay）nq：从不排队（Never Queue）

四种类型Cluster IP 集群内访问NodePort 暴露给外部访问 (kubenode:port) 30000~33000loadBalancer 暴露给外部访问 （ 依赖外部负载均衡器 ）metallb(https://github.com/kubernetes/cloud-provider-aws)ExternalName 类似cname，可以实现跨namespace的ingress


apiVersion: v1
kind: Service
metadata:
    name: my-service
    namespace: prod
spec:
    type: ExternalName
    externalName: my.database.example.com



当查找主机 my-service.prod.svc.cluster.local 时，集群 DNS 服务返回 CNAME 记录， 其值为 my.database.example.com

-ingress社区推荐 nginx-ingress通俗意义上的七层负载均衡，service 和 pod 仅可在集群内部网络中通过 IP 地址访问。所有到达边界路由器的流量或被丢弃或被转发到其他地方。从概念上讲，可能像下面这样：

internet        |  ------------  [Services]

-Ingress 是授权入站连接到达集群服务的规则集合。


internet        |   [Ingress]   --|-----|--   [Services]


-你可以给 Ingress 配置提供外部可访问的 URL、负载均衡、SSL、基于名称的虚拟主机等。用户通过 POST Ingress 资源到 API server 的方式来请求 ingress。 Ingress controller 负责实现 Ingress，通常使用负载均衡器，它还可以配置边界路由和其他前端，这有助于以高可用的方式处理流量。 

-安装：https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/



```
~ ❯❯❯ helm repo add nginx-stable https://helm.nginx.com/stable~ ❯❯❯ helm install ingress-nginx ingress-nginx/ingress-nginx --create-namespace --namespace ingressNAME: ingress-nginxLAST DEPLOYED: Sun Feb 27 02:44:22 2022NAMESPACE: ingressSTATUS: deployedREVISION: 1TEST SUITE: NoneNOTES:The ingress-nginx controller has been installed.It may take a few minutes for the LoadBalancer IP to be available.You can watch the status by running 'kubectl --namespace ingress get services -o wide -w ingress-nginx-controller'An example Ingress that makes use of the controller:  apiVersion: networking.k8s.io/v1  kind: Ingress  metadata:    name: example    namespace: foo  spec:    ingressClassName: nginx    rules:      - host: www.example.com        http:          paths:            - backend:                service:                  name: exampleService                  port:                    number: 80              path: /    # This section is only required if TLS is to be enabled for the Ingress    tls:      - hosts:        - www.example.com        secretName: example-tlsIf TLS is enabled for the Ingress, a Secret containing the certificate and key must also be provided:  apiVersion: v1  kind: Secret  metadata:    name: example-tls    namespace: foo  data:    tls.crt: <base64 encoded cert>    tls.key: <base64 encoded key>  type: kubernetes.io/tls~ ❯❯❯ kubectl get pod -n ingressNAME                                      READY   STATUS    RESTARTS   AGEingress-nginx-controller-54bfb9bb-vgdtm   1/1     Running   0          12h~ ❯❯❯ kubectl get svc -n ingressNAME                                 TYPE           CLUSTER-IP    EXTERNAL-IP      PORT(S)                      AGEingress-nginx-controller             LoadBalancer   10.0.78.161   52.170.175.195   80:31909/TCP,443:32228/TCP   12hingress-nginx-controller-admission   ClusterIP      10.0.222.50   <none>           443/TCP                      12h~ ❯❯❯
```


-Ingress 是对集群中服务的外部访问进行管理的 API 对象，典型的访问方式是 HTTP。



```
apiVersion: networking.k8s.io/v1kind: Ingressmetadata:  name: httpserver-80spec:  ingressClassName: nginx  rules:    - host: mod8.51.cafe      http:        paths:          - backend:              service:                name: httpsvc                port:                  number: 80            path: /            pathType: Prefix

```


-cert-manager

-https://cert-manager.io/docs/installation/
-https://cert-manager.io/docs/installation/helm/

-install


```
helm repo add jetstack https://charts.jetstack.iohelm repo updatekubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.7.1/cert-manager.crds.yamlhelm install \  cert-manager jetstack/cert-manager \  --namespace cert-manager \  --create-namespace \  --version v1.7.1 \
```


-https://cert-manager.io/docs/configuration/vault/


-Issuer/cluster-issuer

-ca 颁发机构支持acme的，你都可以用cert-manager，https://datatracker.ietf.org/doc/html/rfc8555

-签发证书CA的配置


```
apiVersion: cert-manager.io/v1kind: Issuermetadata:  generation: 1  name: letsencrypt-prodspec:  acme:    email: xxx@example.com    preferredChain: ""    privateKeySecretRef:      name: letsencrypt-prod    server: https://acme-v02.api.letsencrypt.org/directory    solvers:    - http01:        ingress:          class: nginxShell  SessionapiVersion: networking.k8s.io/v1kind: Ingressmetadata:  annotations:    cert-manager.io/issuer: letsencrypt-prod  name: fmengspec:  ingressClassName: nginx  rules:    - host: mod8-ssl.51.cafe      http:        paths:          - backend:              service:                name: httpsvc                port:                  number: 80            path: /            pathType: Prefix  tls:    - hosts:        - mod8-ssl.51.cafe      secretName: mod8-tls

```

-配置证书


```
~/g/s/c/模块8 ❯❯❯ kubectl get cert -n mod8NAME       READY   SECRET     AGEmod8-tls   True    mod8-tls   5m58s~/g/s/c/模块8 ❯❯❯ kubectl get CertificateRequest -n mod8NAME             APPROVED   DENIED   READY   ISSUER             REQUESTOR                                         AGEmod8-tls-sqgh9   True                True    letsencrypt-prod   system:serviceaccount:cert-manager:cert-manager   6m9sShell  Session

```


```

apiVersion: networking.k8s.io/v1kind: Ingressmetadata:  annotations:    cert-manager.io/issuer: letsencrypt-prod  name: fmengspec:  ingressClassName: nginx  rules:    - host: mod8-ssl.51.cafe      http:        paths:          - backend:              service:                name: httpsvc                port:                  number: 80            path: /            pathType: Prefix  tls:    - hosts:        - mod8-ssl.51.cafe      secretName: mod8-tls

```


