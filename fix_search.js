const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', 'utf8');

// handleSearch 里，如果 keyword 为空，也要执行
let newContent = content.replace(
  /if \(!keyword \&\& !isByValue\) \{[\s\S]*?\}\n/,
  ``
);

fs.writeFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', newContent);
console.log('done');
