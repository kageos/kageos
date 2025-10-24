
首先如果更新了镜像，那么需要重新创建app才能用上最新的镜像
这个是创建app的接口
```shell
curl --location --request POST 'http://127.0.0.1:9090/api/v1/app/create' \
--header 'X-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo1LCJ1c2VybmFtZSI6ImJlaWx1byIsImVtYWlsIjoibGl1MTIxMDIyNzA4MEAxMjYuY29tIiwiaXNzIjoiYWktYWdlbnQtb3MiLCJzdWIiOiI1IiwiZXhwIjoxNzk3MTQwOTk2LCJuYmYiOjE3NjExNDA5OTYsImlhdCI6MTc2MTE0MDk5Nn0.exkdtanr3wzf0vu18-iOTQzk3pEpMFeiFMEf-ctFJsw' \
--header 'Content-Type: application/json' \
--data-raw '{
    "app": "aaa",
    "user":"beiluo"
  }'
```

创建完app后，此时还没有构建容器，需要先调用
```shell

curl --location --request POST 'http://127.0.0.1:9090/api/v1/app/update/bbb' \
--header 'X-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo1LCJ1c2VybmFtZSI6ImJlaWx1byIsImVtYWlsIjoibGl1MTIxMDIyNzA4MEAxMjYuY29tIiwiaXNzIjoiYWktYWdlbnQtb3MiLCJzdWIiOiI1IiwiZXhwIjoxNzk3MTQwOTk2LCJuYmYiOjE3NjExNDA5OTYsImlhdCI6MTc2MTE0MDk5Nn0.exkdtanr3wzf0vu18-iOTQzk3pEpMFeiFMEf-ctFJsw' \
--header 'Content-Type: application/json' \
--data-raw '{
    "user": "beiluo",
    "app": "aaa"
  }'
```
更新app，此时更新完毕就算构建成功了，此时正常来讲已经可以对该应用发起请求了
aaa是对应的应用名称，test/add 是sdk内置的测试路由


```shell
curl --location --request POST 'http://127.0.0.1:9090/api/v1/app/request/aaa/test/add' \
--header 'X-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo1LCJ1c2VybmFtZSI6ImJlaWx1byIsImVtYWlsIjoibGl1MTIxMDIyNzA4MEAxMjYuY29tIiwiaXNzIjoiYWktYWdlbnQtb3MiLCJzdWIiOiI1IiwiZXhwIjoxNzk3MTQwOTk2LCJuYmYiOjE3NjExNDA5OTYsImlhdCI6MTc2MTE0MDk5Nn0.exkdtanr3wzf0vu18-iOTQzk3pEpMFeiFMEf-ctFJsw' \
--header 'Content-Type: application/json' \
--data-raw '{
  "name":"test",
  "value":"test1234"
}'

返回示例：
{
    "code": 0,
    "data": {
        "id": 24,
        "name": "test",
        "value": "test1234"
    },
    "msg": "成功",
    "metadata": {
        "app": "aaa",
        "total_cost_mill": 9,
        "trace_id": "f3539939-63dc-4fcd-82fe-9bed587e6e8a",
        "version": "v1"
    }
}

```

此时如果再调用更新接口时候，会升级成v2版本，这时候我们再发起请求返回的是v2，此时容器里应该是 v1和v2进程共存
如果询检正常执行的话，过一会就会把v1停掉，只保留v2进程