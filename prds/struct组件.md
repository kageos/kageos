


struct类型的定义


```json
{
    "callbacks": null,
    "children": [
        {
            "callbacks": null,
            "children": null,
            "code": "address",
            "data": {
                "example": "",
                "format": "",
                "type": "string"
            },
            "desc": "",
            "name": "收货地址",
            "search": "",
            "table_permission": "",
            "validation": "required",
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
            "children": null,
            "code": "phone",
            "data": {
                "example": "",
                "format": "",
                "type": "string"
            },
            "desc": "",
            "name": "联系电话",
            "search": "",
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
            "children": null,
            "code": "note",
            "data": {
                "example": "",
                "format": "",
                "type": "string"
            },
            "desc": "",
            "name": "备注",
            "search": "",
            "table_permission": "",
            "validation": "",
            "widget": {
                "config": {
                    "default": "",
                    "placeholder": ""
                },
                "type": "text_area"
            }
        }
    ],
    "code": "detail",
    "data": {
        "example": "",
        "format": "",
        "type": "struct"
    },
    "desc": "",
    "name": "订单详情",
    "search": "",
    "table_permission": "",
    "validation": "required",
    "widget": {
        "config": null,
        "type": "form"
    }
}
```

这个完全可以把组件渲染成一个form的表单一样