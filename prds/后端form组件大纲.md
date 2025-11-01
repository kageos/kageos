
```go


// 测试用的嵌套结构体
type OrderItem struct {
	ID       int     `json:"id" widget:"name:商品ID;type:ID"`
	Name     string  `json:"name" widget:"name:商品名称;type:input"`
	Price    float64 `json:"price" widget:"name:价格;type:float"`
	Quantity int     `json:"quantity" widget:"name:数量;type:number"`
}

type OrderDetail struct {
	Address string `json:"address" widget:"name:收货地址;type:input"`
	Phone   string `json:"phone" widget:"name:联系电话;type:input"`
	Note    string `json:"note" widget:"name:备注;type:text_area"`
}

type Order struct {
	ID     int          `json:"id" widget:"name:订单ID;type:ID"`
	Title  string       `json:"title" widget:"name:订单标题;type:input"`
	Status string       `json:"status" widget:"name:订单状态;type:select;options:待发货,已发货,已收货"`
	Items  []OrderItem  `json:"items" widget:"name:订单项;type:table"`  // 明确指定为table
	Detail *OrderDetail `json:"detail" widget:"name:订单详情;type:form"` // 明确指定为form
	Remark string       `json:"remark" widget:"name:备注;type:text_area"`
}

```


解析后的request是

```json
[
    {
        "code": "id",
        "desc": "",
        "name": "订单ID",
        "search": "",
        "data": {
            "type": "int",
            "format": "",
            "example": ""
        },
        "widget": {
            "type": "ID",
            "config": {

            }
        },
        "children": null,
        "callbacks": null,
        "table_permission": "",
        "validation": ""
    },
    {
        "code": "title",
        "desc": "",
        "name": "订单标题",
        "search": "",
        "data": {
            "type": "string",
            "format": "",
            "example": ""
        },
        "widget": {
            "type": "input",
            "config": {
                "placeholder": "",
                "password": false,
                "prepend": "",
                "append": "",
                "default": ""
            }
        },
        "children": null,
        "callbacks": null,
        "table_permission": "",
        "validation": ""
    },
    {
        "code": "status",
        "desc": "",
        "name": "订单状态",
        "search": "",
        "data": {
            "type": "string",
            "format": "",
            "example": ""
        },
        "widget": {
            "type": "select",
            "config": {
                "options": null,
                "placeholder": "",
                "default": ""
            }
        },
        "children": null,
        "callbacks": null,
        "table_permission": "",
        "validation": ""
    },
    {
        "code": "items",
        "desc": "",
        "name": "订单项",
        "search": "",
        "data": {
            "type": "[]struct",
            "format": "",
            "example": ""
        },
        "widget": {
            "type": "table",
            "config": null
        },
        "children": [
            {
                "code": "id",
                "desc": "",
                "name": "商品ID",
                "search": "",
                "data": {
                    "type": "int",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "ID",
                    "config": {

                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            },
            {
                "code": "name",
                "desc": "",
                "name": "商品名称",
                "search": "",
                "data": {
                    "type": "string",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "input",
                    "config": {
                        "placeholder": "",
                        "password": false,
                        "prepend": "",
                        "append": "",
                        "default": ""
                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            },
            {
                "code": "price",
                "desc": "",
                "name": "价格",
                "search": "",
                "data": {
                    "type": "float",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "float",
                    "config": {
                        "placeholder": "",
                        "password": false,
                        "prepend": "",
                        "append": "",
                        "default": ""
                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            },
            {
                "code": "quantity",
                "desc": "",
                "name": "数量",
                "search": "",
                "data": {
                    "type": "int",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "number",
                    "config": {
                        "placeholder": "",
                        "password": false,
                        "prepend": "",
                        "append": "",
                        "default": ""
                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            }
        ],
        "callbacks": null,
        "table_permission": "",
        "validation": ""
    },
    {
        "code": "detail",
        "desc": "",
        "name": "订单详情",
        "search": "",
        "data": {
            "type": "struct",
            "format": "",
            "example": ""
        },
        "widget": {
            "type": "form",
            "config": null
        },
        "children": [
            {
                "code": "address",
                "desc": "",
                "name": "收货地址",
                "search": "",
                "data": {
                    "type": "string",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "input",
                    "config": {
                        "placeholder": "",
                        "password": false,
                        "prepend": "",
                        "append": "",
                        "default": ""
                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            },
            {
                "code": "phone",
                "desc": "",
                "name": "联系电话",
                "search": "",
                "data": {
                    "type": "string",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "input",
                    "config": {
                        "placeholder": "",
                        "password": false,
                        "prepend": "",
                        "append": "",
                        "default": ""
                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            },
            {
                "code": "note",
                "desc": "",
                "name": "备注",
                "search": "",
                "data": {
                    "type": "string",
                    "format": "",
                    "example": ""
                },
                "widget": {
                    "type": "text_area",
                    "config": {
                        "placeholder": "",
                        "default": ""
                    }
                },
                "children": null,
                "callbacks": null,
                "table_permission": "",
                "validation": ""
            }
        ],
        "callbacks": null,
        "table_permission": "",
        "validation": ""
    },
    {
        "code": "remark",
        "desc": "",
        "name": "备注",
        "search": "",
        "data": {
            "type": "string",
            "format": "",
            "example": ""
        },
        "widget": {
            "type": "text_area",
            "config": {
                "placeholder": "",
                "default": ""
            }
        },
        "children": null,
        "callbacks": null,
        "table_permission": "",
        "validation": ""
    }
]
```