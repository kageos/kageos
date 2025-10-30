form类型模版函数的渲染其实参数的定义也是一模一样的，table的response就可以直接渲染成form

我们的逻辑是这样的，点击服务目录的各个节点，
获取应用的服务目录接口：http://127.0.0.1:9090/api/v1/service_tree?user=luobei&app=test10



```json

{
    "code": 0,
    "data": [
        {
            "id": 41,
            "name": "客户服务",
            "code": "crm",
            "parent_id": 0,
            "type": "package",
            "description": "",
            "tags": "",
            "app_id": 80,
            "ref_id": 0,
            "full_code_path": "/luobei/test10/crm",
            "children": [
                {
                    "id": 42,
                    "name": "工单管理",
                    "code": "crm_ticket",
                    "parent_id": 41,
                    "type": "function",
                    "description": "一个简单的工单管理系统 ........",
                    "tags": "工单管理系统",
                    "app_id": 80,
                    "ref_id": 18,
                    "full_code_path": "/luobei/test10/crm/crm_ticket"
                }
            ]
        },
        {
            "id": 43,
            "name": "效率工具",
            "code": "tools",
            "parent_id": 0,
            "type": "package",
            "description": "",
            "tags": "",
            "app_id": 80,
            "ref_id": 0,
            "full_code_path": "/luobei/test10/tools",
            "children": [
                {
                    "id": 44,
                    "name": "斐波那契数列计算器",
                    "code": "tools_fibonacci",
                    "parent_id": 43,
                    "type": "function",
                    "description": "输入起始位置（1-100）和结束位置（1-100），计算指定区间的斐波那契数列。斐波那契数列：F(1)=1, F(2)=1, F(n)=F(n-1)+F(n-2)。使用大整数计算，避免数值溢出问题。应用场景：数学教育、算法演示、金融分析等。",
                    "tags": "数学,计算器,斐波那契",
                    "app_id": 80,
                    "ref_id": 21,
                    "full_code_path": "/luobei/test10/tools/tools_fibonacci"
                }
            ]
        }
    ],
    "msg": "成功",
    "metadata": null
}
```

点击到function类型的话，会调用获取函数详情接口来获取函数详情 form类型的本质就是提交表单，也就是说，基于函数详情渲染出一个提交的表单

给你看一下form函数的详情 http://127.0.0.1:9090/api/v1/function/get?function_id=21



tools_fibonacci（form类型） 求斐波那契数列函数表单函数详情，每个节点有package，function，这两种类型，我们的table是属于function类型，template_type=form 是属于表单类型，
```json

{
    "code": 0,
    "data": {
        "id": 21,
        "app_id": 80,
        "tree_id": 0,
        "method": "POST",
        "router": "/luobei/test10/tools/tools_fibonacci",
        "has_config": false,
        "create_tables": "",
        "callbacks": "",
        "template_type": "form",
        "request": [
            {
                "callbacks": null,
                "code": "start",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "起始位置",
                "search": "",
                "table_permission": "",
                "validation": "required,min=1,max=100",
                "widget": {
                    "config": {
                        "append": "",
                        "default": "",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "number"
                }
            },
            {
                "callbacks": null,
                "code": "end",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "结束位置",
                "search": "",
                "table_permission": "",
                "validation": "required,min=1,max=100",
                "widget": {
                    "config": {
                        "append": "",
                        "default": "",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "number"
                }
            },
            {
                "callbacks": null,
                "code": "stp",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "分隔符",
                "search": "",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "append": "",
                        "default": ",",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "input"
                }
            }
        ],
        "response": [
            {
                "callbacks": null,
                "code": "sequence",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "斐波那契数列",
                "search": "",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "append": "",
                        "default": "",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "text"
                }
            },
            {
                "callbacks": null,
                "code": "sum",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "数列和",
                "search": "",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "append": "",
                        "default": "",
                        "password": false,
                        "placeholder": "",
                        "prepend": ""
                    },
                    "type": "text"
                }
            }
        ],
        "created_at": "2025-10-30T12:07:39Z",
        "updated_at": "2025-10-30T12:07:39Z",
        "full_code_path": ""
    },
    "msg": "成功",
    "metadata": null
}

```

request和response都有很清晰的输入参数和输出参数，输出参数是可以预知的，具体什么类型，具体是干嘛的，怎么渲染都可以知道，所以当我们点击服务目录的form函数时候，
函数详情要渲染出一个form表单，以这个求斐波那契数列为例

我们输入框有三个
起始位置
结束位置
分隔符
然后有个默认的提交按钮（提交时候我们要根据      
"method": "POST",
"router": "/luobei/test10/tools/tools_fibonacci" 调用/api/v1/run/{router} 这个接口就是运行函数的接口 ）
然后body参数是基于我们的表单参数来构建的，假如我们此时输入框的值是：
起始位置：1
结束位置：10
分隔符：","
那么我们提交时该携带的body应该是：
```json

{
  "start":1,
  "end":10,
  "stp":","
}

```

然后返回的body是

```json
{
  "code": 0,
  "data": {
    "sequence": "1,1,2,3,5,8,13,21,34,55",
    "sum":"143"
  },
  "msg": "成功",
  "metadata": {
    "app": "test10",
    "total_cost_mill": 28,
    "trace_id": "50a8f1b4-882d-4697-8141-36e721a5f79d",
    "version": "v4"
  }
}


```

非常丝滑的一套是不是？






