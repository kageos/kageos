现在是v7版本
我点击一下更新返回
```json

{
    "code": 0,
    "data": {
        "user": "beiluo",
        "app": "test777",
        "old_version": "v7",
        "new_version": "v8",
        "status": "updated",
        "diff": {
            "diff": {
                "added_apis": null,
                "deleted_apis": null,
                "updated_apis": [
                    {
                        "added_version": "v7",
                        "code": "add",
                        "create_tables": null,
                        "desc": "",
                        "method": "POST",
                        "name": "",
                        "request": null,
                        "response": null,
                        "router": "/test/add",
                        "tags": null,
                        "update_versions": [
                            "v8"
                        ]
                    },
                    {
                        "added_version": "v7",
                        "code": "get",
                        "create_tables": null,
                        "desc": "",
                        "method": "POST",
                        "name": "",
                        "request": null,
                        "response": null,
                        "router": "/test/get",
                        "tags": null,
                        "update_versions": [
                            "v8"
                        ]
                    }
                ]
            },
            "message": "API diff completed successfully",
            "status": "success",
            "timestamp": "2025-10-26T15:26:03.521100719+08:00",
            "trigger": {
                "trigger": "update_callback"
            },
            "version": "v8"
        }
    },
    "msg": "成功",
    "metadata": null
}
```


然后现在变成v8了，我没变代码，又更新了一下
```json
{
    "code": 0,
    "data": {
        "user": "beiluo",
        "app": "test777",
        "old_version": "v8",
        "new_version": "v9",
        "status": "updated",
        "diff": {
            "diff": {
                "added_apis": null,
                "deleted_apis": null,
                "updated_apis": [
                    {
                        "added_version": "v8",
                        "code": "add",
                        "create_tables": null,
                        "desc": "",
                        "method": "POST",
                        "name": "",
                        "request": null,
                        "response": null,
                        "router": "/test/add",
                        "tags": null,
                        "update_versions": [
                            "v9"
                        ]
                    },
                    {
                        "added_version": "v8",
                        "code": "get",
                        "create_tables": null,
                        "desc": "",
                        "method": "POST",
                        "name": "",
                        "request": null,
                        "response": null,
                        "router": "/test/get",
                        "tags": null,
                        "update_versions": [
                            "v9"
                        ]
                    }
                ]
            },
            "message": "API diff completed successfully",
            "status": "success",
            "timestamp": "2025-10-26T15:26:55.593788362+08:00",
            "trigger": {
                "trigger": "update_callback"
            },
            "version": "v9"
        }
    },
    "msg": "成功",
    "metadata": null
}
```

明显不对，代码没变update_versions为啥里面变了？另外此次返回的diff下面应该全是null啊，因为没没新增，没删除，没修改啊，另外update_versions是假如我这次编译的时候做diff发现
api发生了变更才会把此次的version追加进去的，懂吗？明显有错误，另外，data下面套了两个diff字段是不是不好？
