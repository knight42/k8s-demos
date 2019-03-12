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

### kubectl-rm
删除 Resource 前先备份到 `~/.k8s-wastebin/<cluster>/<namespace>/<time>_<args list>.yaml` 中。

例子:
```
$ kubectl rm deploy nginx
$ ls ~/.k8s-wastebin/kops-test.k8s.local/default
2019-03-11T15:01:34+08:00_rm_deploy_echo.yaml  2019-03-11T15:06:22+08:00_rm_cm_-lfoo.yaml
```

### kubectl-podstatus
查找相应的 Deployment 或 Statefulset 或 DaemonSet, 并列出其管理的 Pod 的状态。

例子:
```sh
# 自动查找
$ kubectl podstatus perf
Deployment: default/perf
Selector: -lapp=perf

NAME                    READY   STATUS    RESTARTS   PODIP          HOSTIP          NODE                                           AGE
perf-5fb9999756-d9fhc   1/1     Running   0          100.96.4.144   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   1h
```

```sh
# 指定 Kind
$ kubectl podstatus -n infra deploy/echoserver
Deployment: infra/echoserver
Selector: -lrun=echoserver

NAME                          READY   STATUS    RESTARTS   PODIP          HOSTIP          NODE                                           AGE
echoserver-7dd9469844-tbhgt   1/1     Running   0          100.96.4.139   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   7h
```

```sh
# 通过 label 选择 Pods
$ kubectl podstatus -lcronjob=sleep
Selector: -lcronjob=sleep

NAME                     READY   STATUS      RESTARTS   PODIP         HOSTIP          NODE                                           AGE
sleep-1551877200-57895   0/1     Completed   0          100.96.3.57   172.31.71.175   ip-172-31-71-175.cn-north-1.compute.internal   55m
sleep-1551879000-zwl6l   0/1     Completed   0          100.96.3.87   172.31.71.175   ip-172-31-71-175.cn-north-1.compute.internal   25m
```

```sh
# 跟踪 Deployment 的 Pod 的变化
$ kubectl podstatus -w perf
Deployment: default/perf
Selector: -lapp=perf

NAME                    READY   STATUS    RESTARTS   PODIP          HOSTIP          NODE                                           AGE
perf-6566dbff9f-847g7   1/1     Running   0          100.96.4.150   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   3h
perf-6566dbff9f-847g7   1/1   Terminating   0     100.96.4.150   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   3h
perf-6566dbff9f-m9525   0/1   Pending   0     <none>   <none>   <none>   <none>
perf-6566dbff9f-m9525   0/1   Pending   0     <none>   <none>   ip-172-31-67-191.cn-north-1.compute.internal   <none>
perf-6566dbff9f-m9525   0/1   ContainerCreating   0     <none>   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   0s
perf-6566dbff9f-847g7   0/1   Terminating   0     100.96.4.150   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   3h
perf-6566dbff9f-847g7   0/1   Terminating   0     <none>   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   3h
perf-6566dbff9f-847g7   0/1   Terminating   0     <none>   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   3h
perf-6566dbff9f-847g7   0/1   Terminating   0     <none>   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   3h
perf-6566dbff9f-m9525   1/1   Running   0     100.96.4.151   172.31.67.191   ip-172-31-67-191.cn-north-1.compute.internal   11s
```

### kubectl-nodestat
查看 Node 的 CPU usage/allocatable/requests/limits, Memory usage/allocatable/requests/limits。

注: 需要 Kubernetes 集群中部署了 [metrics-server](https://github.com/kubernetes-incubator/metrics-server)

例子:
```sh
# 注意：当集群中 Node 或 Pod 数比较多的话，会比较慢，推荐使用 `-l <label>` 筛选 Node 或者指定 Node Name
$ kubectl nodestat
Progress: 100.00% (5/5)
NAME                                           CPU(USAGE/TOTAL)   REQUESTS/LIMITS           MEMORY(USAGE/TOTAL)   REQUESTS/LIMITS
ip-172-31-67-0.cn-north-1.compute.internal     106m/1000m/10.6%   900m(90.0%)/0m(0.0%)      1930Mi/3666Mi/52.6%   150Mi(4.1%)/100Mi(2.7%)
ip-172-31-67-191.cn-north-1.compute.internal   49m/2000m/2.5%     360m(18.0%)/250m(12.5%)   2426Mi/3854Mi/63.0%   130Mi(3.4%)/230Mi(6.0%)
ip-172-31-67-84.cn-north-1.compute.internal    97m/1000m/9.7%     850m(85.0%)/0m(0.0%)      1814Mi/3666Mi/49.5%   100Mi(2.7%)/100Mi(2.7%)
ip-172-31-71-175.cn-north-1.compute.internal   64m/2000m/3.2%     840m(42.0%)/0m(0.0%)      2477Mi/3854Mi/64.3%   330Mi(8.6%)/440Mi(11.4%)
ip-172-31-78-247.cn-north-1.compute.internal   76m/1000m/7.6%     850m(85.0%)/0m(0.0%)      1892Mi/3666Mi/51.6%   100Mi(2.7%)/100Mi(2.7%)
```

```sh
# 通过 label 筛选 Node
$ kubectl nodestat -lrole=gw
Progress: 100.00% (9/9)
NAME                                           CPU(USAGE/TOTAL)    REQUESTS/LIMITS              MEMORY(USAGE/TOTAL)   REQUESTS/LIMITS
ip-172-31-68-82.cn-north-1.compute.internal    524m/4000m/13.1%    1610m(40.2%)/4100m(102.5%)   5281Mi/7382Mi/71.5%   1148Mi(15.6%)/2843Mi(38.5%)
ip-172-31-71-6.cn-north-1.compute.internal     776m/4000m/19.4%    2010m(50.2%)/5100m(127.5%)   5074Mi/7382Mi/68.7%   748Mi(10.1%)/2543Mi(34.4%)
...
```

```sh
# 指定 Node Name
$ kubectl nodestat ip-172-31-68-82.cn-north-1.compute.internal
Progress: 100.00% (1/1)
NAME                                          CPU(USAGE/TOTAL)   REQUESTS/LIMITS              MEMORY(USAGE/TOTAL)   REQUESTS/LIMITS
ip-172-31-68-82.cn-north-1.compute.internal   522m/4000m/13.1%   1610m(40.2%)/4100m(102.5%)   5280Mi/7382Mi/71.5%   1148Mi(15.6%)/2843Mi(38.5%)
```
