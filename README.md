# k8s-tools

一些 kubectl plugins

# Installation
```
go get -v github.com/knight42/k8s-tools/cmd/...
```

PS: 使用前需要确保 `$GOPATH/bin` 在 `PATH` 环境变量中:
```
export PATH=$GOPATH/bin:$PATH
```

# Plugins
* [kubectl-rm](#kubectl-rm)
* [kubectl-podstatus](#kubectl-podstatus)

### kubectl-rm
删除 Resource 前先备份到 `~/.k8s-wastebin/<cluster>/<namespace>/<kind>/<name>_<time>.yaml` 中。

目前仅支持:
* Deployment
* ConfigMap
* StatefulSet
* DaemonSet
* Service
* Cronjob

例子:
```
$ kubectl rm deploy nginx
$ ls ~/.k8s-wastebin/kops-test.k8s.local/default/deployments/
nginx_2019-02-14T17:54:40+08:00.yaml
```

### kubectl-podstatus
查找相应的 Deployment 或 Statefulset 或 DaemonSet, 并列出其管理的 Pod 的状态。

例子:
```
# 自动查找
$ kubectl podstatus perf
Deployment: default/perf
Selector: -lapp=perf

NAME                    READY   STATUS    RESTARTS   PODIP          HOSTIP          NODE                                           AGE
perf-5fb9999756-d9fhc   1/1     Running   0          100.96.4.144   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   1h


# 指定 Kind
$ kubectl podstatus -n infra deploy/echoserver
Deployment: infra/echoserver
Selector: -lrun=echoserver

NAME                          READY   STATUS    RESTARTS   PODIP          HOSTIP          NODE                                           AGE
echoserver-7dd9469844-tbhgt   1/1     Running   0          100.96.4.139   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   7h


# 通过 label 选择 Pods
$ kubectl podstatus -lcronjob=sleep
Selector: -lcronjob=sleep

NAME                     READY   STATUS      RESTARTS   PODIP         HOSTIP          NODE                                           AGE
sleep-1551877200-57895   0/1     Completed   0          100.96.3.57   172.31.71.175   ip-172-31-71-175.cn-north-1.compute.internal   55m
sleep-1551879000-zwl6l   0/1     Completed   0          100.96.3.87   172.31.71.175   ip-172-31-71-175.cn-north-1.compute.internal   25m
```
