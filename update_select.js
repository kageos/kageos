const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', 'utf8');

// 把 selectedValues 传给 FuzzySearchDialog
let newContent = content.replace(
  /:loading="loading"\n\s*:is-multiselect="false"/,
  `:loading="loading"
      :is-multiselect="false"
      :selected-values="hasCurrentValue ? [internalValue] : []"`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', newContent);
console.log('done');
