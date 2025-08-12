

## use
启动
```shell
./NetWeakTest -N=node[1-3] -L=[debug, info] -T=30ns -options=[] -- bash
```
查询
```shell
./NetWeakTest -N=node[1-3] -L=[debug, info] -type=show
```
重置
```shell
./NetWeakTest -N=node[1-3] -L=[debug, info] -type=reset
```

options:
* loss x%
* limit x
* delay xms yms
* corrupt x%
* duplicate x%
* reorder x%
* rate X[k,m]bit
