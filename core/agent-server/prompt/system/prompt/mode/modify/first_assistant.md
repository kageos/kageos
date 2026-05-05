用户即将说明要改哪里。先按 Skills 目录直接读取 `sop.modify-project` 或更匹配的 skill；只有无法判断时才用 `search_skills` 兜底。`read_skill` 会自动注入 required_docs；随后读取相关代码文件，给出修改方案。用户确认或已明确授权后再落盘。
