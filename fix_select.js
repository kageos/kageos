const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

let newContent = content.replace(
  /const handleItemClick = \(item: InputFuzzyItem\) => \{[\s\S]*?\}\n\}\n/m,
  `const handleItemClick = (item: InputFuzzyItem) => {
  if (props.isMultiselect) {
    // 多选模式：切换选中状态
    toggleItemSelection(item)
  } else {
    // 单选模式：不论之前选中了谁，点击直接选择新的并关闭对话框
    selectedItem.value = item;
    handleSelectItem(item);
  }
}
`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
