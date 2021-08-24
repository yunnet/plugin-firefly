# 萤火虫插件
firefly plugin for monibuca


### 1、登陆

* 接口：/api/firefly/login

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| username | string |   |
| password | string |   |

### 2、重启机器

* 接口：/api/firefly/reboot

* 请求方式：GET

* 请求参数：无

### 3、JSON配置查询

* 接口：/api/firefly/config

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| node | string | 如：network.trap

### 4、JSON配置编辑

* 接口：/api/firefly/config/edit

* 请求方式：POST

* 请求参数：

  {"node":"network.trap","data":{"ac":"YNLIJ-jm-lojoo-001","id":"1","ip":"218.202.50.250","port":1620}}

###  5、网络查询

* 接口：/api/firefly/config/tcp

* 请求方式：GET

* 请求参数：无

### 6、网络设置

* 接口：/api/firefly/config/tcp/edit

* 请求方式：POST

* 请求参数：

  {"address":"192.168.0.110","dns-nameservers":"10.8.201.6","gateway":"192.168.0.1","netmask":"255.255.255.0"}

### 7、网络Ping

* 接口：/api/firefly/config/ping

* 请求方式：GET

* 请求参数：无

### 8、点播功能

* 接口：/api/firefly/config/ping

* 请求方式：GET

* 请求参数：

  访问 http://192.168.0.110:8080/vod/live/hk-2021-08-23-181514.flv 

  将会读取对应的flv文件

### 9、查询所有Flv文件

* 接口：/api/record/list

* 请求方式：GET

* 请求参数：无

### 10、开始录制

* 接口：/api/record/start

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| streamPath | string | |

### 11、停止录制

* 接口：/api/record/stop

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| streamPath | string | |

### 12、将某个flv文件读取并发布成一个直播流

* 接口：/api/record/play

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| streamPath | string  | 文件名|

### 13、删除某个flv文件

* 接口：/api/record/delete

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| streamPath | string  | 文件名 |