# AI-Agent-OS Agent 工作流流程图

> 这张图描述的是工作台 Agent 的主流程：用户在当前目录提出需求后，Agent 如何识别目标角色、读取上下文、选择工具、生成或执行能力，并最终把结果沉淀为平台资产。

```mermaid
flowchart TD
    U[用户在工作台输入需求] --> Ctx[注入工作台上下文<br/>当前目录 / 可执行函数 / 附件 / 会话历史 / 模式]
    Ctx --> Route{识别目标角色}

    Route -->|product_manager / app_developer| Create[创建应用流程]
    Route -->|maintenance_engineer| Modify[修改应用流程]
    Route -->|qa_engineer| Execute[执行应用流程]
    Route -->|data_operator| Misc[通用工具流程]
    Route -->|reviewer| Explain[解释项目流程]

    Create --> Boundary[读取能力边界<br/>判断是否能映射到 Form / Table / Chart]
    Boundary --> CanBuild{平台能承载吗?}
    CanBuild -->|否| Fallback[说明边界<br/>给降级方案<br/>record_workspace_event]
    CanBuild -->|是| ReadCreateDocs[读取 SDK 文档<br/>读取匹配案例]
    ReadCreateDocs --> PRD[输出 PRD / 字段 / 列表 / 业务规则]
    PRD --> Confirm{用户确认?}
    Confirm -->|否| Refine[按反馈调整方案]
    Refine --> PRD
    Confirm -->|是| WriteCode[create_directory / write_go_file<br/>生成业务代码]
    WriteCode --> Build[build_workspace<br/>编译并部署]
    Build --> Validate[调用 run_table_search / run_form_submit / run_chart_query 验证]
    Validate --> Fix{验证通过?}
    Fix -->|否| Debug[read_app_log / read_go_file<br/>修复后重新编译]
    Debug --> Build
    Fix -->|是| AppReady[应用可用<br/>进入 ServiceTree]

    Modify --> ReadExisting[读取现有目录与代码<br/>read_dir / read_go_file]
    ReadExisting --> ModifyPlan[输出修改方案]
    ModifyPlan --> ModifyConfirm{用户确认?}
    ModifyConfirm -->|否| ModifyPlan
    ModifyConfirm -->|是| Patch[search_replace_file 或 write_go_file]
    Patch --> Build

    Execute --> CurrentFns{当前目录有目标函数?}
    CurrentFns -->|是| PickRunTool[选择执行工具<br/>Table / Form / Chart]
    CurrentFns -->|否| ReadDir[read_dir 确认子目录或函数路径]
    ReadDir --> PickRunTool
    PickRunTool --> RunTool[run_table_search / run_table_create / run_table_update<br/>run_form_submit / run_chart_query]
    RunTool --> RenderResult[返回表格 / 表单结果 / 图表 / 文件]

    Misc --> LocalTool{当前目录已有相关工具?}
    LocalTool -->|是| RunTool
    LocalTool -->|否| SearchTools[search_tools<br/>按关键词和 template_type 搜索标准工具]
    SearchTools --> ToolFound{找到已注册工具?}
    ToolFound -->|是| Schema[读取 request schema<br/>构造参数]
    Schema --> RunTool
    ToolFound -->|否| SearchHub[search_hub_directory<br/>搜索 Hub 模板或工具]
    SearchHub --> HubFound{Hub 有可用目录?}
    HubFound -->|是| CopyHub[copy_directory<br/>复制到当前工作区]
    CopyHub --> RunTool
    HubFound -->|否| NeedCreate[询问是否创建新工具]
    NeedCreate --> Create

    Explain --> ReadProject[read_dir / read_doc<br/>读取目录、函数、文档]
    ReadProject --> ExplainResult[用业务语言解释当前项目能力]

    AppReady --> Asset{是否发布或复用?}
    RenderResult --> Asset
    Asset -->|发布| Publish[publish_to_hub / push_to_hub]
    Asset -->|继续使用| Done[交付结果给用户]
    Publish --> Done
    Fallback --> Done
    ExplainResult --> Done
```

## 简化版

适合放在官网或演示 PPT 里：

```mermaid
flowchart LR
    A[用户描述需求] --> B[工作台理解上下文]
    B --> C{已有能力?}
    C -->|有| D[搜索工具 / 选择函数]
    D --> E[按 schema 执行]
    E --> F[返回文件 / 表格 / 图表]

    C -->|没有| G[读边界 / SDK / 案例]
    G --> H[生成 PRD]
    H --> I[用户确认]
    I --> J[写代码并编译]
    J --> K[执行验证]
    K --> L[进入 ServiceTree]
    L --> M[发布到 Hub / 复用]
```

## 核心原则

- **先复用，再生成**：当前目录有能力就直接执行；没有再 `search_tools`；还没有再搜 Hub；最后才创建。
- **先方案，再落盘**：创建和修改应用必须先出 PRD 或修改方案，用户确认后再写代码。
- **先验证，再交付**：生成或修改后必须 `build_workspace`，有可执行函数时要调用 `run_*` 验证。
- **标准能力沉淀**：每个生成的 Form/Table/Chart 都会进入 ServiceTree，后续可搜索、可执行、可复制、可发布。
