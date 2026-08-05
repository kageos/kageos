const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

// 判断 isItemSelected
const func = content.match(/function isItemSelected[\s\S]*?}/)[0];
console.log(func);
