# 修改类型：字段改名

先读取目标 Model 和相关 Request/Form/Response。改名时同时检查 `json`、`gorm column`、`widget name`、筛选字段、业务代码引用、link 参数和测试数据。若只改前端显示名，只改 `widget:"name:..."`，不要改 `json` 或数据库列。
