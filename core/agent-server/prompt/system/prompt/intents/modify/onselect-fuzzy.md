# 修改类型：新增 OnSelectFuzzy

回调字段必须是 `type:select` 或 `type:multiselect`，并在字段上写 `callback:"OnSelectFuzzy"`。`OnSelectFuzzyMap` 的 key 必须等于字段 `json` code。依赖字段用 `depend_on` 并在回调中 `BindCurrentFormData`。
