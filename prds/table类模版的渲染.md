
table类型的模版会被渲染成element类型的表格
我们的逻辑是这样的，点击服务目录的各个节点，
/api/v1/service_tree?user=beiluo&app=testapi21

```json
{
    "code": 0,
    "data": [
        {
            "id": 33,
            "name": "客户服务",
            "code": "crm",
            "parent_id": 0,
            "type": "package",
            "description": "",
            "tags": "",
            "app_id": 70,
            "ref_id": 0,
            "full_code_path": "/beiluo/testapi21/crm",
            "children": [
                {
                    "id": 34,
                    "name": "工单管理",
                    "code": "crm_ticket",
                    "parent_id": 33,
                    "type": "function",
                    "description": "一个简单的工单管理系统 ........",
                    "tags": "工单管理系统",
                    "app_id": 70,
                    "ref_id": 16,
                    "full_code_path": "/beiluo/testapi21/crm/crm_ticket"
                }
            ]
        }
    ],
    "msg": "成功",
    "metadata": null
}
```


每个节点有package，function，这两种类型，我们的table是属于function类型，点击到function类型的话，会调用获取函数详情接口来获取函数详情

/api/v1/function/get?function_id=16




获取函数详情：
```json

{
    "code": 0,
    "data": {
        "id": 16, //函数id
        "app_id": 70, 
        "tree_id": 34, //所属对应的服务树的id
        "method": "GET", //接口的方法
        "router": "/beiluo/testapi21/crm/crm_ticket", //路由
        "has_config": false, //预留字段
        "create_tables": "crm_ticket", //这个函数创建的表名
        "callbacks": "", //包含全局回调，分号分隔,一般table函数没有全局回调，OnPageLoad 这种，这种是加载页面时候回调然后初始化页面参数的
        "template_type": "table", //标识渲染成table类型的
        "request": {}, //table函数一般请求参数是null，因为搜索框在下面的response的search标签里，response里的每个字段都是对应的数据库的表字段
        //一个response就是一个表结构，所以表内字段的搜索可以在search内打标签来支持根据这个字段的查询，但是表外字段的查询就需要在request里了
        //例如，只看自己 这个参数，明显是一个不在表里的逻辑参数，这样的话需要在request里单独渲染,
      //下面的字段显示的非常清楚 id要精确查询，created_at可以搜索大于小于，最终查询的请求参数是这样的，例如你要查询
      //
        "response": [
            {
                "callbacks": null,
                "code": "id",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "ID",
                "search": "eq",
                "table_permission": "read",
                "validation": "",
                "widget": {
                    "config": {},
                    "type": "ID"
                }
            },
            {
                "callbacks": null,
                "code": "created_at",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "timestamp"
                },
                "desc": "",
                "name": "创建时间",
                "search": "get,lte",
                "table_permission": "read",
                "validation": "",
                "widget": {
                    "config": {
                        "disabled": false,
                        "format": "YYYY-MM-DD HH:mm:ss"
                    },
                    "type": "timestamp"
                }
            },
            {
                "callbacks": null,
                "code": "title",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "工单标题",
                "search": "like",
                "table_permission": "",
                "validation": "required,min=2,max=200",
                "widget": {
                    "config": {
                        "append": "",
                        "default": "",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "input"
                }
            },
            {
                "callbacks": null,
                "code": "description",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "问题描述",
                "search": "like",
                "table_permission": "",
                "validation": "required,min=10",
                "widget": {
                    "config": {
                        "default": "",
                        "placeholder": ""
                    },
                    "type": "text_area"
                }
            },
            {
                "callbacks": null,
                "code": "priority",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "优先级",
                "search": "in",
                "table_permission": "",
                "validation": "required,oneof=低,中,高",
                "widget": {
                    "config": {
                        "default": "中",
                        "options": [
                            "低",
                            "中",
                            "高"
                        ],
                        "placeholder": ""
                    },
                    "type": "select"
                }
            },
            {
                "callbacks": null,
                "code": "status",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "工单状态",
                "search": "in",
                "table_permission": "",
                "validation": "required,oneof=待处理,处理中,已完成,已关闭",
                "widget": {
                    "config": {
                        "default": "待处理",
                        "options": [
                            "待处理",
                            "处理中",
                            "已完成",
                            "已关闭"
                        ],
                        "placeholder": ""
                    },
                    "type": "select"
                }
            },
            {
                "callbacks": null,
                "code": "phone",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "联系电话",
                "search": "like",
                "table_permission": "",
                "validation": "required,min=11,max=20",
                "widget": {
                    "config": {
                        "append": "",
                        "default": "",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "input"
                }
            },
            {
                "callbacks": null,
                "code": "remark",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "备注",
                "search": "like",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "default": "",
                        "placeholder": ""
                    },
                    "type": "text_area"
                }
            },
            {
                "callbacks": null,
                "code": "create_by",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "创建用户",
                "search": "like",
                "table_permission": "read",
                "validation": "",
                "widget": {
                    "config": {},
                    "type": "user"
                }
            }
        ],
        "created_at": "2025-10-28T22:34:43Z",
        "updated_at": "2025-10-28T22:34:43Z"
    },
    "msg": "成功",
    "metadata": null
}
```

这个函数会被渲染是这样的
首先我们先在服务目录点击这个函数，然后我们会调用函数详情接口，获取到函数详情的数据，根据详情来渲染界面，函数详情可能是不同的类型，眼下我们先介绍template_type=table的
点击服务目录的table函数后，调用获取函数详情接口，获取到函数的详情，根据response的参数来渲染表格，然后再自动调用运行函数接口（table函数需要自动运行一下）
/api/v1/run/{router} 这个接口就是运行函数的接口，函数详情返回的
"method": "GET", //接口的方法
"router": "/beiluo/testapi21/crm/crm_ticket", //路由
已经说清楚了这个函数怎么运行，直接调用 GET /api/v1/run/beiluo/testapi21/crm/crm_ticket 即可，

假如你要查询优先级高的，然后降序排序参数是这样的
?in=priority:高&sort=id:desc


假如要查询某个时间段内的，然后降序排序
?gte=created_at:${起始的时间戳毫秒}&lte=created_at:${截止的时间戳毫秒}&sort=id:desc

假设要查询id=1的记录
?eq=id:1


假设要根据title字段模糊查询
?like=title:公积金

查询第一页，查询20条
?page=1&page_size=20


支持的搜索类型（目前这么多，后续再扩展）
eq 精确匹配
gte 大于等于
lte 小于等于
like 模糊查询
in 包含




然后返回的table数据是这样的
```json

{
    "code": 0,
    "data": {
        "items": [
          {
            "id": 1,
            "created_at": 1719388800000,
            "title": "网站登录问题",
            "description": "用户反馈无法正常登录系统，提示密码错误，但密码确认正确。需要技术排查登录接口是否存在问题。",
            "priority": "高",
            "status": "处理中",
            "phone": "13800138000",
            "remark": "用户为VIP客户，需要优先处理",
            "create_by": "张三"
          },
          {
            "id": 2,
            "created_at": 1719475200000,
            "title": "订单支付失败",
            "description": "用户在下单支付时多次尝试均提示支付失败，但银行卡余额充足，需要检查支付网关连接状态。",
            "priority": "中",
            "status": "待处理",
            "phone": "13900139000",
            "remark": "用户提供了支付截图和错误信息",
            "create_by": "李四"
          }
        ],
        "paginated": {
            "current_page": 1,
            "page_size": 20,
            "total_count": 2,
            "total_pages": 1
        }
    },
    "msg": "成功",
    "metadata": {
        "app": "testapi21",
        "total_cost_mill": 28,
        "trace_id": "50a8f1b4-882d-4697-8141-36e721a5f79d",
        "version": "v4"
    }
}
```

这样的话我们直接可以把这个table渲染出来了，这样的话，这个table是一个通用的渲染引擎，后续我们字段不一样的话，一样可以用这个渲染引擎来渲染










