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
根据名字查找相应的 Deployment 或 Statefulset 或 DaemonSet, 并列出其管理的 Pod 的状态

例子:
```
$ kubectl podstatus perf
+-----------------------+-------+---------+----------+---------------+----------------------------------------------+
|         NAME          | READY | STATUS  | RESTARTS |      IP       |                     NODE                     |
+-----------------------+-------+---------+----------+---------------+----------------------------------------------+
| perf-5fb9999756-d2pjv | 1/1   | Running |        0 | 172.31.67.191 | ip-172-31-67-191.cn-north-1.compute.internal |
+-----------------------+-------+---------+----------+---------------+----------------------------------------------+
```
