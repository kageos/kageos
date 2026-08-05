const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

let newContent = content.replace(
  /if \(props\.isMultiselect && props\.selectedValues\) \{[\s\S]*?\}\n\s*\}/m,
  `if (props.isMultiselect && props.selectedValues) {
      selectedItems.value = props.suggestions.filter(item => 
        props.selectedValues.some(val => String(val) === String(item.value))
      )
    } else if (!props.isMultiselect && props.selectedValues && props.selectedValues.length > 0) {
      const targetVal = String(props.selectedValues[0]);
      selectedItem.value = props.suggestions.find(item => String(item.value) === targetVal) || null;
    }
  }`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
