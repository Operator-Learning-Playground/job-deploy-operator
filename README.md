## jobflow-operator 简易型 Job 树状引擎控制器

### 项目思路与设计
设计背景：k8s 当中原生的 Job 资源对象执行时，并没有相互依赖的编排特性(ex: Job a 完成后 -> 再执行Job b ...)。
本项目在此需求上，基于 k8s 的扩展功能，实现 JobFlow 的自定义资源控制器，实现一个能执行多 Job 树状引擎的 operator 应用。

![](./image/%E6%97%A0%E6%A0%87%E9%A2%98-2023-08-10-2343.png?raw=true)

- crd 资源对象如下，更多信息可以参考 [参考](./yaml/example.yaml)
    - name: flow 名称，多个 flow 名称不能重复
    - dependencies: 定义依赖项，如果有多个依赖可以填写多个
    - jobTemplate: job 模版，支持 k8s 原生 job spec 全部字段
```yaml
apiVersion: api.practice.com/v1alpha1
kind: JobFlow
metadata:
  name: jobflow-example
spec:
  # 可填写多个 flow 流程
  # 每个 flow 中重要字段 分别为：
  # name: flow 名称，多个 flow 名称不能重复
  # dependencies: 定义依赖项，如果有多个依赖可以填写多个
  # jobTemplate: job 模版，支持 k8s 原生 job spec 全部字段
  flows:
    - name: job1
      dependencies: []
      jobTemplate:
        template:
          spec:
            containers:
              - image: busybox:1.28
                command:
                  - sh
                  - -c
                  - sleep 10s
                imagePullPolicy: IfNotPresent
                name: nginx
    - name: job2
      jobTemplate:
        template:
          spec:
            containers:
              - image: busybox:1.28
                command:
                  - sh
                  - -c
                  - sleep 100s
                imagePullPolicy: IfNotPresent
                name: nginx
      dependencies:
        - job1  # 代表 job2 依赖 job1 完成后才开始启动
    - name: job3
      jobTemplate:
        template:
          spec:
            containers:
              - image: busybox:1.28
                command:
                  - sh
                  - -c
                  - sleep 100s
                imagePullPolicy: IfNotPresent
                name: nginx
      dependencies:
        # 代表 job3 依赖 job1 job2 完成后才开始启动
        - job1
        - job2
    - name: job4
      jobTemplate:
        template:
          spec:
            containers:
              - image: busybox:1.28
                command:
                  - sh
                  - -c
                  - sleep 10s
                imagePullPolicy: IfNotPresent
                name: nginx
    - name: job5
      dependencies:
        # 代表依赖 job2 job4 后才执行
        - job4
        - job2
      jobTemplate:
        template:
          spec:
            containers:
              - image: busybox:1.28
                command:
                  - sh
                  - -c
                  - sleep 10s
                imagePullPolicy: IfNotPresent
                name: nginx
```

### 项目功能
1. 支持 JobFlow 任务中的 job 依赖执行
2. 查看任务流状态
- 注：pod 字段中的 **restartPolicy**  不允许被使用，就算定义后也不会生效(都会被强制设为"Never")