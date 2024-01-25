## helm 部署

### 修改配置文件
用户需要**自行修改** [配置文件](./values.yaml)。
- base: 基础配置，镜像需要自行构建 (docker build...)
- rbac: 用于创建 rbac 使用
注：如果部署有任何问题，欢迎提 issue 或 直接联系



- 镜像构建
```bash
[root@vm-0-12-centos jobflowoperator]# docker build -t jobflowoperator:v1 .
Sending build context to Docker daemon  1.469MB
Step 1/15 : FROM golang:1.18.7-alpine3.15 as builder
 ---> 33c97f935029
Step 2/15 : WORKDIR /app
 ---> Using cache
 ---> 4edcb0247ad5
Step 3/15 : COPY go.mod go.mod
 ---> Using cache
 ---> 4debb7a476ee
Step 4/15 : COPY go.sum go.sum
 ---> Using cache
 ---> f234b1886296
Step 5/15 : ENV GOPROXY=https://goproxy.cn,direct
 ---> Using cache
 ---> 16052e806626
Step 6/15 : ENV GO111MODULE=on
 ---> Using cache
 ---> 6c4a62849ce0
Step 7/15 : RUN go mod download
 ---> Using cache
 ---> 8514c4b4083f
Step 8/15 : COPY main.go main.go
 ---> Using cache
 ---> 9b6d0d5fcf25
Step 9/15 : COPY pkg/ pkg/
 ---> Using cache
 ---> c0a865f7d093
Step 10/15 : RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o jobflowoperator main.go
 ---> Using cache
 ---> bae7be4a98e2
Step 11/15 : FROM alpine:3.12
 ---> 24c8ece58a1a
Step 12/15 : WORKDIR /app
 ---> Using cache
 ---> 00cee1a43d13
Step 13/15 : COPY --from=builder /app/jobflowoperator .
 ---> Using cache
 ---> 36117787d603
Step 14/15 : USER 65532:65532
 ---> Using cache
 ---> d11efff3f80b
Step 15/15 : ENTRYPOINT ["./jobflowoperator"]
 ---> Using cache
 ---> d749696f4dc0
Successfully built d749696f4dc0
Successfully tagged jobflowoperator:v1
[root@vm-0-12-centos jobflowoperator]#
```

- helm 部署
```bash
[root@vm-0-12-centos jobflowoperator]# cd helm/
[root@vm-0-12-centos helm]# pwd
/root/jobflowoperator/helm
[root@vm-0-12-centos helm]# helm install jobflowoperator .
NAME: jobflowoperator
LAST DEPLOYED: Thu Jan 25 17:54:24 2024
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
```