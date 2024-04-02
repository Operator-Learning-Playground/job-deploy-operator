## Simple Job Orchestration Operator
<a href="./README.md">English</a> | <a href="./README-zh.md">简体中文</a>
### Project Idea and Design
Design Background: This project aims to implement a simple Job orchestration operator.

### JobFlow
Feature: Native Job resources in Kubernetes do not have native orchestration features with dependencies (e.g., Job a completes -> execute Job b...). To address this requirement, a custom resource controller called JobFlow is implemented based on Kubernetes' extension capabilities. It enables the execution of multiple Job trees in an operator application.
- Support dependencies between jobs
- Support global parameter passing between jobs(label,annotation,env)
- Supports scheduling between jobs to the same node [example](./yaml/jobflow/example-sameNode.yaml)
- Support shared data volumes between jobs [example](./yaml/jobflow/example-shareVolume.yaml)
- Supports Job references to JobTemplate objects [example](./yaml/jobflow/example-jobTemplate.yaml)

![](./image/jobflow.png?raw=true)

- The CRD (Custom Resource Definition) resource object is as follows. For more information, please refer to reference. [reference](yaml/jobflow/example.yaml)
    - globalParams: Global parameters that will be automatically rendered in each job.
    - name: flow name, multiple flow names cannot be repeated
    - dependencies: Define dependencies. If there are multiple dependencies, we can fill in multiple
    - jobTemplate: Job template that supports Kubernetes native job spec fields.
    - jobTemplateRef: Job template instance, supports all fields of k8s native job spec, needs to fill in the JobTemplate name
    - shareVolumes: job shared data volume
    - shareVolumeMounts: job shared data volume mount
```yaml
apiVersion: api.practice.com/v1alpha1
kind: JobFlow
metadata:
  name: jobflow-example
spec:
  # Configurable global parameters in the job flow, which will take effect in each job and pod
  globalParams:
    # Determines that all jobs run on the same node
    nodeName: minikube
    # Add parameters required by the container
    env:
      - name: "FOO"
        value: "bar"
      - name: "QUE"
        value: "pasa"
    # Annotations for job pods
    annotations:
      key1: value1
      key2: value2
    # Labels for job pods
    labels:   
      key1: value1
      key2: value2
  # You can specify multiple flow processes
  # Important fields in each flow are:
  # name: Flow name. Multiple flow names must be unique.
  # dependencies: Defines the dependencies. Multiple dependencies can be specified.
  # jobTemplate: Job template that supports all native Kubernetes job spec fields.
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
        - job1  # job2 depends on job1 to complete before starting
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
        # job3 depends on job1 and job2 to complete before starting
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
        # job5 depends on job2 and job4 to complete before starting
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

### Features
1. Supports job dependencies in JobFlow tasks.
2. View the status of the job flow.
```yaml
[root@vm-0-12-centos jobflow]# kubectl get jobflows.api.practice.com
NAME                    STATUS    AGE
jobflow-example         Running   117s
jobflow-example-error   Failed    117s
[root@vm-0-12-centos jobflow]# kubectl get jobs | grep jobflow-example
jobflow-example-error-job1                   1/1           13s        3d18h
jobflow-example-error-job2-error-container   0/1           3d18h      3d18h
jobflow-example-error-job4                   1/1           12s        3d18h
jobflow-example-job1                         1/1           12s        2m44s
jobflow-example-job2                         1/1           102s       2m31s
jobflow-example-job3                         0/1           48s        48s
jobflow-example-job4                         1/1           12s        2m44s
jobflow-example-job5                         1/1           12s        48s
[root@vm-0-12-centos jobflow]# kubectl  get pods | grep jobflow-example
jobflow-example-error-job1-x9tn8                   0/1     Completed   0          3d18h
jobflow-example-error-job2-error-container-wxpvb   0/1     Error       0          3d18h
jobflow-example-error-job4-p286g                   0/1     Completed   0          3d18h
jobflow-example-job1-4zq22                         0/1     Completed   0          3m8s
jobflow-example-job2-qsvr9                         0/1     Completed   0          2m55s
jobflow-example-job3-4f6lb                         1/1     Running     0          72s
jobflow-example-job4-7ngpd                         0/1     Completed   0          3m8s
jobflow-example-job5-m8cwg                         0/1     Completed   0          72s
```
- Note: The restartPolicy field in the Pod specification is not allowed to be used, 
and even if defined, it will not take effect (it will be forcibly set to "Never").


### DaemonJob
Functionality: When native Job resources are executed in Kubernetes, there is no workload similar to the daemonset workload in Pods. 
Based on this requirement, we aim to implement a custom resource controller called DaemonJob using Kubernetes' extension capabilities. 
This controller will enable the execution of jobs on selected nodes, providing an operator application that can execute jobs on designated nodes.


![](./image/daemonjob.png?raw=true)

- The CRD (Custom Resource Definition) resource object is as follows. For more information, please refer to reference. [reference](yaml/daemonjob/example.yaml)
  - globalParams: Global parameters that will be automatically rendered in each job.
  - excludeNodeList: Node names where the job should not be executed, separated by commas.
  - template: Job template that supports Kubernetes native job spec fields.
```yaml
apiVersion: api.practice.com/v1alpha1
kind: DaemonJob
metadata:
  name: daemonjob-example
spec:
  # Configurable global parameters in the job flow, which will take effect in each job and pod
  globalParams:
    # Add parameters required by the container
    env:
      - name: "FOO"
        value: "bar"
      - name: "QUE"
        value: "pasa"
    # job pod 的 annotations
    annotations:
      key1: value1
      key2: value2
    # job pod 的 labels
    labels:
      key1: value1
      key2: value2
  # Except these nodes are not created, jobs must be created, separated by ","
  excludeNodeList: node1, node2
  # Support native job Template template
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - image: busybox:1.28
          command:
            - sh
            - -c
            - sleep 100s
          imagePullPolicy: IfNotPresent
          name: nginx
```

### Features
1. Support job execution in DaemonJob tasks
2. View the status of the job flow
```bash
[root@vm-0-12-centos jobflow]# kubectl get daemonjobs.api.practice.com
NAME                STATUS    AGE
daemonjob-example   Running   117s
[root@vm-0-12-centos jobflow]# kubectl get daemonjobs.api.practice.com
NAME                STATUS    AGE
daemonjob-example   Succeed   3m21s
[root@vm-0-12-centos jobflowoperator]# kubectl  get jobs | grep daemonjob-example
daemonjob-example-vm-0-12-centos             1/1           102s       3d17h
daemonjob-example-vm-0-17-centos             1/1           102s       3d17h

```
- Note: The restartPolicy field in the Pod specification is not allowed to be used,
  and even if defined, it will not take effect (it will be forcibly set to "Never").


### Install
Helm deployment is currently supported, please refer to [here](helm)