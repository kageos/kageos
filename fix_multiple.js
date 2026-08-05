const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/MultiSelectWidget.vue', 'utf8');

// handleDialogSelectMultiple 里不能合并 selectedValues 否则无法反选
let newContent = content.replace(
  /const newValues = items\.map\(item => item\.value\)[\s\S]*?const allValues = Array\.from\(new Set\(\[\.\.\.selectedValues\.value, \.\.\.newValues\]\)\)/,
  `const newValues = items.map(item => item.value)
  const allValues = newValues`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/MultiSelectWidget.vue', newContent);
console.log('done');
