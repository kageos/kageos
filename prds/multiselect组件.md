
```json
{
  "callbacks": [
    "OnSelectFuzzy"
  ],
  "children": null,
  "code": "category",
  "data": {
    "example": "",
    "format": "",
    "type": "[]string"
  },
  "desc": "",
  "name": "分类",
  "search": "",
  "table_permission": "",
  "validation": "required,min=1",
  "widget": {
    "config": {
      "creatable": false,
      "default": null,
      "max_count": 0,
      "options": null,
      "placeholder": ""
    },
    "type": "multiselect"
  }
}
```

注意多选组件的OnSelectFuzzy 回调返回的数据中如果有max_selections字段的话表示这个组件是有限制最多选择数量的

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
      "max_selections": 3, //这个在多选的情况下才有意义，单选的组件下可以忽略这个参数，多选模式下，这个参数如果存在需要严格控制选项,
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

"max_selections": 1, //这个在多选的情况下才有意义，单选的组件下可以忽略这个参数，多选模式下，这个参数如果存在需要严格控制选项，