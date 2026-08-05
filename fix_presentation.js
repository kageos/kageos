const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/SelectFuzzyPresentation.vue', 'utf8');

// 修复多余的内容
let newContent = content.replace(
  /\.select-fuzzy-presentation-deleted \{[\s\S]*?\}\n/,
  ``
);

fs.writeFileSync('web/src/architecture/presentation/widgets/SelectFuzzyPresentation.vue', newContent);
console.log('done');
