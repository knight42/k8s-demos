# k8s-tools

一些 kubectl 的 plugins:

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
name=perf-fc679db49-fxqck status=Running restart=0 hostIP=172.31.71.228 nodeName=ip-172-31-71-228.cn-north-1.compute.internal
```
