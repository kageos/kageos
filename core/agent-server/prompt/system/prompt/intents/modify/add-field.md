# 修改类型：新增字段

先判断字段是否落库。落库字段加到 Table Model，并配置合法 widget、validate 和 display。计算/展示字段用 `gorm:"-"`，通常加 `hide:"create,update"`。Table 筛选字段写在 Request 中，不能重复声明 Model 已有 `json` code。
