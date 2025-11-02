我们的timestamp 组件涵盖了整个时间日期的选择，注意整个时间日期的值都是时间戳，且是毫秒级别的时间戳，整个系统只有这一个时间相关的组件
这个timestamp涵盖了所有
包括 时间选择，日期选择，年份选择，月份选择，时间日期，年份日期等等所有形式，只是通过format来进行格式化
分为两种时间类型
一种是绝对时间：例如
YYYY-MM-DD HH:mm:ss 这种是精确的时间，需要传递精确的时间戳

HH:mm:ss 这种是相对时间，不关心具体的年份月日等等，这种传递时间戳，需要传递 1970 1月1日的 HH:mm:ss 的时间，
为啥要这样干？因为这样后端才能根据这个时间进行排序，假如这个字段存储的是下课时间，那么我们需要根据下课时间进行排序的话就非常方便了
如果用绝对时间戳的方式存储的话，后面新增的记录即使下课早也会被排在后面，这是一个点，
所以这里先分清楚哪些是绝对时间，哪些是相对时间




因为有table_permission：read 说明在新增数据时候这个字段是不显示的
```json

{
                "callbacks": null,
                "children": null,
                "code": "created_at",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "创建时间",
                "search": "",
                "table_permission": "read",
                "validation": "",
                "widget": {
                    "config": {
                        "disabled": false, //这个字段标识是否能进行时间的选择
                        "format": "YYYY-MM-DD HH:mm:ss"
                    },
                    "type": "timestamp"
                }
            }
```

这个字段无论是在table的新增修改还是在form函数的表单中都是可以调出时间日期选择器来选择时间的，
```json

{
                "callbacks": null,
                "children": null,
                "code": "push_at",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "发布时间",
                "search": "",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "disabled": false,
                        "format": "YYYY-MM-DD HH:mm:ss"
                    },
                    "type": "timestamp"
                }
            }
```