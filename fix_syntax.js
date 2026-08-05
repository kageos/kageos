const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

// 修复多出来的大括号
let newContent = content.replace(
  /    handleSelectItem\(item\);\n  \}\n\}\n\}\n/m,
  `    handleSelectItem(item);\n  }\n}\n`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
