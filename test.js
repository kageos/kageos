const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/MultiSelectWidget.vue', 'utf8');

console.log(content.match(/function handleDialogSelectMultiple[\s\S]*?\n\}/)[0]);
