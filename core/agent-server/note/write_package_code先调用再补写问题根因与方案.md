# write_package_code「先调用再补写」问题根因与方案

## 一、现象

模型经常在同一轮中**先**发起 `write_package_code` 调用（此时还没写 `<var>`），报错后**再**在下一轮写 `<var>` 并再次调用 `write_package_code` 才成功，体验差。

## 二、根因（来自社区与文档）

1. **LLM 的生成特性**  
   当模型「决定」要调用工具时，往往会在该轮**直接输出 tool_calls**，同一条消息里的 **content 经常很少或为空**。  
   - OpenAI 社区：["When I get tool_calls back in the response, I do not get content"](https://community.openai.com/t/getting-tool-calls-and-content-back-from-same-api-call/1031026)  
   - 即：**tool_calls 与 大段 content 在同一条消息里「同时出现」并不是默认保证的**，很多情况下是「要么 tool_calls，要么 content」。

2. **顺序不可控**  
   即便同一轮里既有 content 又有 tool_calls，**流式/非流式 API 都不保证「先 content 后 tool_calls」的生成顺序**，所以无法单靠 prompt 要求「先写 `<var>` 再写 tool_calls」来 100% 约束模型。

3. **Prompt 的局限**  
   单靠系统提示词/工具描述强调「必须先写 `<var>...</var>` 再调用」可以减轻问题，但无法从机制上杜绝「先调用再补写」，因为模型在「决定调用工具」时往往已经倾向于先输出 tool_calls。

## 三、可选方案

| 方案 | 做法 | 优点 | 缺点 |
|------|------|------|------|
| **A. 兜底：从历史 assistant 解析 \<var\>** | 解析变量时，若**本条**消息没有 `<var>`，则从**本会话中最近一条包含 `<var>` 的 assistant 消息**里解析。 | 实现简单；能覆盖「上一轮写了 \<var\>、本轮只调 write_package_code」的情况，减少一次失败。 | **同一轮**「先调 write_package_code 再补写 \<var\>」仍会失败（当时还没有含 \<var\> 的历史）。 |
| **B. 两阶段：首轮不提供 write_package_code** | 检测到用户要「生成并写入代码」时，**第一轮**不把 write_package_code 加入 tools（或 tool_choice 排除），让模型只能输出文本（含 \<var\>）；第二轮再提供 write_package_code。 | 从机制上保证「先有 \<var\> 再出现 write_package_code」。 | 要改调用链、工具列表/ tool_choice 的动态逻辑，实现和产品设计都更重。 |
| **C. 两轮调用** | 第一轮 `tool_choice: "none"` 或仅文本，拿到带 \<var\> 的回复；第二轮再带 tools 让模型调 write_package_code，或由服务端从 \<var\> 解析后直接执行写入。 | 行为可控。 | 多一次请求、编排复杂。 |

## 四、当前实现（方案 A 兜底）

- 在 **executeToolCalls** 中解析变量时：  
  - 先用 **本条** assistant 的 content 解析 `vars`；  
  - 若 **vars 为空**，则从 **本会话消息列表**里取「**最近一条 role=assistant 且 content 包含 `<var>`**」的消息，用其 content 再解析一次，得到 vars。  
- 这样：  
  - **同一轮**已写 `<var>` 再调用 → 仍从本条解析，行为不变。  
  - **上一轮**写了 `<var>`（未调或只调了别的），**本轮**只调 write_package_code → 可从上一条 assistant 解析到变量，减少一次「未定义变量」报错。

同一轮「先调用再补写」仍会报错（当时历史里还没有 \<var\>），但至少「先写 \<var\> 下一轮再调」的体验会更好；若要彻底避免先调再补写，需要上方案 B 或 C。
