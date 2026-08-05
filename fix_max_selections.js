const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

// 当 maxSelections === 1 时的特殊处理，相当于单选，直接替换而不是无操作
let newContent = content.replace(
  /function toggleItemSelection\(item: InputFuzzyItem\) \{[\s\S]*?\n\}\n/m,
  `function toggleItemSelection(item: InputFuzzyItem) {
  const index = selectedItems.value.findIndex(selected => String(selected.value) === String(item.value))
  if (index >= 0) {
    // 已选中，取消选择
    selectedItems.value.splice(index, 1)
  } else {
    // 未选中
    if (props.maxSelections === 1) {
      // 限制为1个时，直接替换
      selectedItems.value = [item]
    } else if (props.maxSelections > 0 && selectedItems.value.length >= props.maxSelections) {
      // 达到上限，不再添加
      return
    } else {
      selectedItems.value.push(item)
    }
  }
}
`
);

// handleItemCheckboxChange 也做类似处理
newContent = newContent.replace(
  /function handleItemCheckboxChange\(item: InputFuzzyItem, checked: boolean\) \{[\s\S]*?\n\}\n/m,
  `function handleItemCheckboxChange(item: InputFuzzyItem, checked: boolean) {
  if (checked) {
    if (props.maxSelections === 1) {
      selectedItems.value = [item]
    } else if (props.maxSelections > 0 && selectedItems.value.length >= props.maxSelections) {
      return
    } else if (!selectedItems.value.some(selected => String(selected.value) === String(item.value))) {
      selectedItems.value.push(item)
    }
  } else {
    const index = selectedItems.value.findIndex(selected => String(selected.value) === String(item.value))
    if (index >= 0) {
      selectedItems.value.splice(index, 1)
    }
  }
}
`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
