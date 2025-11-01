select

静态options
select有两种选项一种是options
```json
 {
                "callbacks": null,
                "children": null,
                "code": "category",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "string"
                },
                "desc": "",
                "name": "商品分类",
                "search": "in",
                "table_permission": "",
                "validation": "required",
                "widget": {
                    "config": {
                        "creatable": false, //是否支持创建选项？
                        "default": "",//默认值
                        "options": [ //静态选项，无需回调
                            "饮料",
                            "零食",
                            "日用品",
                            "其他"
                        ],
                        "placeholder": ""
                    },
                    "type": "select"
                }
            }
```

动态回调的选项
组件里的callback里假如有OnSelectFuzzy 回调说明啥？
说明这个select的值不是静态的options里的选项，是需要触发回调的，当选选项时候，需要触发OnSelectFuzzy 回调

```json
{
                "callbacks": [
                    "OnSelectFuzzy"
                ],
                "children": null,
                "code": "member_id",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "会员卡",
                "search": "",
                "table_permission": "",
                "validation": "required",
                "widget": {
                    "config": {
                        "creatable": false,
                        "default": "",
                        "options": null,
                        "placeholder": ""
                    },
                    "type": "select"
                }
}
```

回调示例


```json
curl --location --request POST 'http://127.0.0.1:9090/api/v1/callback/luobei/test999/tools/cashier_desk?_type=OnSelectFuzzy&_method=POST' \
--header 'X-Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjo1LCJ1c2VybmFtZSI6ImJlaWx1byIsImVtYWlsIjoibGl1MTIxMDIyNzA4MEAxMjYuY29tIiwiaXNzIjoiYWktYWdlbnQtb3MiLCJzdWIiOiI1IiwiZXhwIjoxNzk3MTQwOTk2LCJuYmYiOjE3NjExNDA5OTYsImlhdCI6MTc2MTE0MDk5Nn0.exkdtanr3wzf0vu18-iOTQzk3pEpMFeiFMEf-ctFJsw' \
--header 'Content-Type: application/json' \
--data-raw '{
"code":"product_id",
"type":"by_keyword",
"value":"输入框的关键字",
"request":{},//当前整个表单的值
"value_type":"int(这个code的类型)"
}'
```
http://127.0.0.1:9090/api/v1/callback 是回调接口
/luobei/test999/tools/cashier_desk 是这个函数的路由
_type=OnSelectFuzzy 是回调的类型
_method=POST 是这个函数的method


下面是回调接口的返回值：
```json
{
    "code": 0,
    "data": {
        "error_msg": "", //如果不为空说明有错误，提示错误，不允许选择
        "items": [
            {
                "display_info": {
                    "价格": 5,
                    "分类": "饮料",
                    "商品名称": "薯条",
                    "库存": 100
                },
                "icon": "",
                "label": "薯条 - ¥5.00 (库存:100)",
                "value": 1
            }
        ],
        "statistics": {
            "优惠金额(元)": "sum(价格,*quantity,*0.1)",
            "会员折扣": "9折优惠",
            "会员折扣后价格(元)": "sum(价格,*quantity,*0.9)",
            "商品原价总额(元)": "sum(价格,*quantity)",
            "商品总数量(件)": "sum(quantity)",
            "商品种类数": "count(价格)",
            "配送说明": "满99元包邮，不满99元运费10元"
        }
    },
    "msg": "成功",
    "metadata": {
        "app": "test999",
        "total_cost_mill": 13,
        "trace_id": "2ba2e0f1-9b6f-4eeb-805f-c929b4c8892f",
        "version": "v2"
    }
}
```