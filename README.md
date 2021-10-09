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

* 返回结果：
  {"code":200,"msg":"ok","data":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzI2Mzk5ODYsImlzcyI6ImFkbWluIn0.lGyxvdi027cl9J512-jdw1fr33ujGfjeN8-OvNN_7nA"}
  
  {"code":500,"msg":"用户名或密码错误,请重新输入","data":null}

### 2、刷新

* 接口：/api/firefly/refresh

* 请求方式：GET

* 请求参数：无,需要Header带token

* 返回结果：

  {"code":200,"msg":"ok","data":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MzI2NDAwNDUsImlzcyI6ImFkbWluIn0.EwejmIy1gapQvwxaqADdHxXqvA8Nqh6uKYvN29uZPog"}

  {"code":10002,"msg":"无效token","data":null}

### 3、重启机器

* 接口：/api/firefly/reboot

* 请求方式：GET

* 请求参数：无

* 返回结果：无

### 4、JSON配置查询

* 接口：/api/firefly/config

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| node | string | 如：network.trap

* 返回结果：

  {"code":200,"msg":"ok","data":{"ac":"YNLIJ-jm-lojoo-001","id":"2","ip":"218.202.50.250","port":1620}}

### 5、JSON配置编辑

* 接口：/api/firefly/config/edit

* 请求方式：POST

* 请求参数：

  {"node":"network.trap","data":{"ac":"YNLIJ-jm-lojoo-001","id":"1","ip":"218.202.50.250","port":1620}}

* 返回结果：

  {"code":200,"msg":"ok","data":"success"}

###  6、网络查询

_* 接口：/api/firefly/config/tcp

* 请求方式：GET

* 请求参数：无

* 返回结果：

  {"code":200,"msg":"ok","data":{"inet":"dhcp"}}

### 7、网络设置

* 接口：/api/firefly/config/tcp/edit

* 请求方式：POST

* 请求参数：

  {"node":"network.trap","data":{"ac":"YNLIJ-jm-lojoo-001","id":"3","ip":"218.202.50.250","port":1620}}

* 返回结果：
  
  {"code":200,"msg":"ok","data":"success"}
  

### 8、网络Ping

* 接口：/api/firefly/config/ping

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| ipaddr | string | 如：192.168.0.110

* 返回结果：

  {"code":200,"msg":"ok","data":"success"}
  
  {"code":500,"msg":"error","data":null}

### 9、点播功能

* 接口：/vod/*

* 请求方式：GET

* 请求参数：

  访问 http://192.168.0.110:8080/vod/live/hk-2021-08-23-181514.flv 

  将会读取对应的flv文件

### 10、查询所有Flv文件

* 接口：/api/record/list

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| date | string | 2021-09-28 |

* 返回结果：

  {"code":200,"msg":"ok","data":[{"url":"live/hw/2021-10-08/15-25-37.mp4","size":17413943,"timestamp":1633677937,"duration":60},{"url":"live/hw/2021-10-08/15-26-37.mp4","size":17441018,"timestamp":1633677997,"duration":60},{"url":"live/hw/2021-10-08/15-27-37.mp4","size":17194436,"timestamp":1633678057,"duration":60}]}


### 11、开始录制

* 接口：/api/record/start

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| streamPath | string | |

* 返回结果：

  {"code":200,"msg":"ok","data":"success"}

  {"code":500,"msg":"error","data":null}

### 12、停止录制

* 接口：/api/record/stop

* 请求方式：GET

* 请求参数：

| 字段  | 类型   | 说明   |
|------|------: | :-----|
| streamPath | string | |

* 返回结果：

  {"code":200,"msg":"ok","data":"success"}

  {"code":500,"msg":"error","data":null}


### 13、查看SD卡信息

* 接口：/api/firefly/storage

* 请求方式：GET

* 请求参数：无

* 返回结果：

  {"code":200,"msg":"ok","data":{"path":"/mnt/sd","fstype":"msdos","total":62528684032,"free":51366330368,"used":11162353664,"usedPercent":17.851572980949825,"inodesTotal":0,"inodesUsed":0,"inodesFree":0,"inodesUsedPercent":0}}
