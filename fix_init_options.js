const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', 'utf8');

// 在打开对话框时，通过 by_value 请求完后，也需要构建 dialogSuggestions，否则列表可能是空的
let newContent = content.replace(
  /await handleSearch\(props\.value\.raw, true\) \/\/ by_value 搜索\n\s*\} else \{/m,
  `await handleSearch(props.value.raw, true) // by_value 搜索
      // 为了让弹窗展示该选项，需要填充 dialogSuggestions
      dialogSuggestions.value = options.value.map((opt) => ({
        label: opt.label,
        value: opt.value,
        displayInfo: toDisplayInfoRecord(opt.displayInfo),
        display_info: toDisplayInfoRecord(opt.displayInfo),
        icon: opt.icon,
        rich_text: opt.richText,
        files: opt.files
      }))
    } else {`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', newContent);
console.log('done');
