const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

let newContent = content.replace(
  /function isItemSelected\(item: InputFuzzyItem\): boolean \{[\s\S]*?\n\}/m,
  `function isItemSelected(item: InputFuzzyItem): boolean {
  if (props.isMultiselect) {
    return selectedItems.value.some(selected => String(selected.value) === String(item.value))
  }
  return selectedItem.value !== null && String(selectedItem.value.value) === String(item.value)
}`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
