const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

// 在 suggestions 变化时，如果单选也要找一下 selectedItem，否则搜完就没了高亮
let newContent = content.replace(
  /watch\(\(\) => props\.suggestions, \(newSuggestions\) => \{[\s\S]*?\}\n\}, \{ immediate: true \}\)/m,
  `// 监听 suggestions 变化，更新已选项目
watch(() => props.suggestions, (newSuggestions) => {
  if (visible.value && props.selectedValues) {
    if (props.isMultiselect) {
      selectedItems.value = newSuggestions.filter(item => 
        props.selectedValues.some(val => String(val) === String(item.value))
      )
    } else if (props.selectedValues.length > 0) {
      const targetVal = String(props.selectedValues[0]);
      selectedItem.value = newSuggestions.find(item => String(item.value) === targetVal) || null;
    }
  }
}, { immediate: true })`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
