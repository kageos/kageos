# Brand Availability Checker

Brand Availability Checker 是一个面向创业命名的轻量 demo，用来验证“输入一个品牌名，一键检查常见平台占用情况，并生成可用性报告”的业务模板形态。

## MVP 范围

- 输入品牌名和业务描述。
- 使用 RDAP 优先检查常见域名后缀：`.com`、`.ai`、`.io`、`.dev`、`.app`、`.co`，RDAP 不可用时再用 DNS 兜底。
- 默认检查开发者平台：GitHub、npm、PyPI、Docker Hub。
- 输出可用性评分、结论、平台检查表、替代名称建议和 Markdown 报告。
- 自动保存检查历史，便于后续复盘候选名。

## 不做的事

- 不承诺域名一定可注册，RDAP/DNS 只作为第一轮信号，最终以注册商和商标检索为准。
- 不做商标法律判断，只提供 USPTO/WIPO 人工复查入口。
- 不做社媒平台的强自动判断，避免被反爬、登录墙和地区策略误导。
- 不做定时监控，第一版只做手动运行。

## 路由

- `POST /brandcheck/brand_availability.form`
- `GET /brandcheck/brand_check_history.table`

## 适合验证的问题

- 用户是否愿意用这个工具筛选创业项目名字。
- 用户是否觉得“报告导出 + 历史记录”比普通 name checker 更有价值。
- 这个模板是否适合做 KageOS 的免费引流 demo 或低价模板商品。
