# 萤火虫插件
firefly plugin for monibuca


### 1、登陆

* 接口：/api/firefly/login

* 请求方式：GET

* 请求参数：

​	| username  | string

​	| password  | string

### 2、重启机器

* 接口：/api/firefly/reboot

* 请求方式：GET

* 请求参数：无

### 3、JSON配置查询

* 接口：/api/firefly/config

* 请求方式：GET

* 请求参数：

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