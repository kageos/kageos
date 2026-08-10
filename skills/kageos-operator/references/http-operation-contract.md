# Direct HTTP operation contract

Send native HTTP requests to the KageOS gateway. Do not generate or interpret an execution-plan JSON file.

## Headers

Send these headers on every request in one run:

```http
Accept: application/json
Content-Type: application/json
X-Client-Source: kageos-operator
X-Source-Type: agent_tool
X-Source-Ref: <stable run reference>
X-Trace-Id: <stable run trace ID>
```

Add exactly one authentication header:

```http
Authorization: Bearer <OpenAPI token>
```

or, temporarily for local/test verification:

```http
X-Token: <access JWT>
```

## Discovery

Call both endpoints immediately before operating the directory:

| Purpose | Method | Endpoint |
| --- | --- | --- |
| Discover functions and schemas | `GET` | `/workspace/api/v1/service_tree/search_functions?full_code_path=<directory>&page=1&page_size=100` |
| Discover packaged automations | `POST` | `/workspace/api/v1/service_tree/export_capability_bundle` |

Use `{"source_directory_path":"<directory>"}` for the export request. Follow pagination for function discovery. Operate only functions whose normalized path is below the requested directory.

## Operations

Append the discovered function `full_code_path` directly to the endpoint prefix:

| Operation | Method | Endpoint prefix | Input |
| --- | --- | --- | --- |
| `form.submit` | `POST` | `/workspace/api/v1/form/submit` | JSON body |
| `table.search` | `GET` | `/workspace/api/v1/table/search` | Query parameters |
| `table.create` | `POST` | `/workspace/api/v1/table/create` | JSON body |
| `table.update` | `PUT` | `/workspace/api/v1/table/update` | JSON body |
| `table.delete` | `DELETE` | `/workspace/api/v1/table/delete` | JSON body |
| `chart.query` | `GET` | `/workspace/api/v1/chart/query` | Query parameters |
| `select_fuzzy.query` | `POST` | `/workspace/api/v1/callback/on_select_fuzzy` | JSON body |

Use Table writes only when discovery exposes the matching callback:

- `OnTableAddRow` permits `table.create`.
- `OnTableUpdateRow` permits `table.update`.
- `OnTableDeleteRows` permits `table.delete`.

Treat `form.submit`, all Table writes, and any callback with external side effects as writes. Obtain authorization before invoking them.

Before calling an operation, inspect widget types in the discovered request or editable fields. Resolve `files`, `user`, `users`, and rich-text assets according to `complex-input-contract.md`; resolve dynamic selects through `OnSelectFuzzy`.

## Choosing calls

| Operation | Use it when |
| --- | --- |
| `form.submit` | The Form is a real user action and its effects are approved. Assert its business response or subsequent state. |
| `table.search` | Read the current state, find a created record, verify an update, or poll an asynchronous outcome. |
| `table.create` | Row creation belongs to the Table workflow and `OnTableAddRow` exists. Prefer the real Form when the Form owns creation. |
| `table.update` | A meaningful editable lifecycle exists and `OnTableUpdateRow` is exposed. Read before and after. |
| `table.delete` | Deletion is the behavior under test or cleanup of a record created by this run. Never delete pre-existing data. |
| `chart.query` | The Chart represents a metric affected by the scenario. Assert a meaningful value or series. |
| `select_fuzzy.query` | A discovered field exposes `OnSelectFuzzy`; validate search and hydration before reusing its values. |

## OnSelectFuzzy

Call the callback with the discovered function path and field code:

```json
{
  "code": "room_id",
  "type": "by_keyword",
  "value": "测试",
  "request": {},
  "value_type": "int"
}
```

- Always call `by_keyword` for every discovered dynamic field.
- Pass the current Form, row, or filter object in `request` when the field has dependencies.
- Capture `data.items[].value` from the real response.
- Call `by_value` for a scalar select or `by_values` for a multiselect before using the captured value elsewhere.
- Assert an empty `error_msg`, the actual item shape, readable labels when items exist, and declared value types.
- Treat a valid empty result as evidence of current empty state, not proof that populated-option rendering works.

## Response-driven execution

Inspect the real envelope and data shape after every call. Require HTTP success, API `code: 0` when present, and business evidence. Adapt later requests to actual fields such as `data.items` and `data.paginated`; never impose example response shapes.

Keep captured identifiers in memory or ephemeral shell variables. Do not persist credentials or raw responses. Never retry writes automatically. Retry only read operations when an asynchronous outcome is expected.

Before approved writes, state the direct call sequence in plain language. Include the synthetic marker, local files to upload, records and storage objects that may be created, expected side effects, and exact cleanup actions. One approval covers that unchanged sequence.
