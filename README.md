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
* [kubectl-nodestat](#kubectl-nodestat)
* [kubectl-scaleig](#kubectl-scaleig)

### kubectl-rm
删除 Resource 前先备份到 `~/.k8s-wastebin/<cluster>/<namespace>/<time>_<args list>.yaml` 中。

例子:
```
$ kubectl rm deploy nginx
$ ls ~/.k8s-wastebin/kops-test.k8s.local/default
2019-03-11T15:01:34+08:00_rm_deploy_echo.yaml  2019-04-16T20:21:59+08:00_rm_svc_perf_deploy_perf.yaml
```

### kubectl-podstatus
查找相应的 Deployment 或 Statefulset 或 DaemonSet, 并列出其管理的 Pod 的状态。

例子:
```sh
# 自动查找
$ kubectl podstatus perf
Deployment: default/perf
Selector: -lapp=perf

NAME                    READY   STATUS    LAST STATUS     RESTARTS   PODIP           HOSTIP         NODE                                          AGE
perf-5fb9999756-d9fhc   1/1     Running   OOMKilled:137   2          100.96.25.224   172.31.77.41   ip-172-31-77-41.cn-north-1.compute.internal   5d
```

```sh
# 指定 Kind
$ kubectl podstatus -n infra deploy/echoserver
Deployment: infra/echoserver
Selector: -lrun=echoserver

NAME                          READY   STATUS    LAST STATUS   RESTARTS   PODIP           HOSTIP          NODE                                           AGE
echoserver-7dd9469844-hlm5v   1/1     Running   <none>        0          100.96.10.195   172.31.70.116   ip-172-31-70-116.cn-north-1.compute.internal   7d
```

```sh
# 通过 label 选择 Pods
$ kubectl podstatus -lcronjob=sleep
Selector: -lcronjob=sleep

NAME                     READY   STATUS      LAST STATUS   RESTARTS   PODIP           HOSTIP          NODE                                           AGE
sleep-1551877200-57895   0/1     Completed   <none>        0          100.96.18.190   172.31.64.115   ip-172-31-64-115.cn-north-1.compute.internal   1d
```

```sh
# 跟踪 Deployment 的 Pod 的变化
$ kubectl podstatus -w perf
Deployment: default/perf
Selector: -lapp=perf

NAME                  READY   STATUS    LAST STATUS   RESTARTS   PODIP         HOSTIP         NODE                                          AGE
perf-7cbf7bf8-bznqv   1/1     Running   <none>        0          100.96.9.78   172.31.74.30   ip-172-31-74-30.cn-north-1.compute.internal   22d
perf-fc679db49-7jgqs   0/1   Pending   <none>   0     <none>   <none>   <none>   <none>
perf-fc679db49-7jgqs   0/1   Pending   <none>   0     <none>   <none>   ip-172-31-74-18.cn-north-1.compute.internal   <none>
perf-fc679db49-7jgqs   0/1   ContainerCreating   <none>   0     <none>   172.31.74.18   ip-172-31-74-18.cn-north-1.compute.internal   0s
perf-fc679db49-7jgqs   1/1   Running   <none>   0     100.96.10.146   172.31.74.18   ip-172-31-74-18.cn-north-1.compute.internal   11s
perf-7cbf7bf8-bznqv   1/1   Terminating   <none>   0     100.96.9.78   172.31.74.30   ip-172-31-74-30.cn-north-1.compute.internal   22d
perf-7cbf7bf8-bznqv   0/1   Terminating   <none>   0     100.96.9.78   172.31.74.30   ip-172-31-74-30.cn-north-1.compute.internal   22d
```

### kubectl-nodestat
查看 Node 的 CPU usage/allocatable/requests/limits, Memory usage/allocatable/requests/limits。

注: 需要 Kubernetes 集群中部署了 [metrics-server](https://github.com/kubernetes-incubator/metrics-server)

例子:
```sh
# 注意：当集群中 Node 或 Pod 数目比较多的话会比较慢。推荐使用 `-l <label>` 筛选 Node 或者指定 Node Name
$ kubectl nodestat
Progress: 100.00% (5/5)
NAME                                           CPU(USAGE/TOTAL)    REQUESTS/LIMITS           MEMORY(USAGE/TOTAL)    REQUESTS/LIMITS
ip-172-31-67-0.cn-north-1.compute.internal     106m(10.6%)/1000m   900m(90.0%)/0m(0.0%)      1930Mi(52.6%)/3666Mi   150Mi(4.1%)/100Mi(2.7%)
ip-172-31-67-191.cn-north-1.compute.internal   49m(2.5%)/2000m     360m(18.0%)/250m(12.5%)   2426Mi(63.0%)/3854Mi   130Mi(3.4%)/230Mi(6.0%)
ip-172-31-67-84.cn-north-1.compute.internal    97m(9.7%)/1000m     850m(85.0%)/0m(0.0%)      1814Mi(49.5%)/3666Mi   100Mi(2.7%)/100Mi(2.7%)
ip-172-31-71-175.cn-north-1.compute.internal   64m(3.2%)/2000m     840m(42.0%)/0m(0.0%)      2477Mi(64.3%)/3854Mi   330Mi(8.6%)/440Mi(11.4%)
ip-172-31-78-247.cn-north-1.compute.internal   76m(7.6%)/1000m     850m(85.0%)/0m(0.0%)      1892Mi(51.6%)/3666Mi   100Mi(2.7%)/100Mi(2.7%)
```

```sh
# 通过 label 筛选 Node
$ kubectl nodestat -lrole=gw
Progress: 100.00% (9/9)
NAME                                           CPU(USAGE/TOTAL)     REQUESTS/LIMITS              MEMORY(USAGE/TOTAL)    REQUESTS/LIMITS
ip-172-31-68-82.cn-north-1.compute.internal    524m(13.1%)/4000m    1610m(40.2%)/4100m(102.5%)   5281Mi(71.5%)/7382Mi   1148Mi(15.6%)/2843Mi(38.5%)
ip-172-31-71-6.cn-north-1.compute.internal     776m(19.4%)/4000m    2010m(50.2%)/5100m(127.5%)   5074Mi(68.7%)/7382Mi   748Mi(10.1%)/2543Mi(34.4%)
...
```

```sh
# 指定 Node Name
$ kubectl nodestat ip-172-31-68-82.cn-north-1.compute.internal
Progress: 100.00% (1/1)
NAME                                          CPU(USAGE/TOTAL)    REQUESTS/LIMITS              MEMORY(USAGE/TOTAL)    REQUESTS/LIMITS
ip-172-31-68-82.cn-north-1.compute.internal   522m(13.1%)/4000m   1610m(40.2%)/4100m(102.5%)   5280Mi(71.5%)/7382Mi   1148Mi(15.6%)/2843Mi(38.5%)
```

### kubectl-scaleig
用于平滑地给 [kops](https://github.com/kubernetes/kops) 创建出来的 instance group 缩容。

默认情况下, kops 通过修改与 instance group 对应的 AWS Auto Scaling Group 的 max size 来缩容, 这样会导致在 terminate EC2 实例的时候, 其实仍然有 Pod 在上面运行, 粗暴地关机会造成服务波动。

比较理想的做法是在关机前先 drain 要删掉的节点, 把上面的 Pod 调度到其他节点, 然后把要删掉的节点从 Auto Scaling Group 中 detach, 最后在分别 terminate 要删掉的节点。

由于手工 detach 跟 terminate 比较繁琐, 且容易出错, 所以写了这个 kubectl plugin 来自动化这一系列过程。

用法:
```sh
# kubectl scaleig -c <kops cluster name> --size <desired size> <instance group name>
# eg
$ kubectl scaleig -c kops-test.k8s.local --size 1 nodes
```
