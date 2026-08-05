const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', 'utf8');

// 在单选组件 SelectWidget 的 openDialog 中
let newContent = content.replace(
  /    \/\/ 🔥 如果已有值，通过 by_value 搜索获取对应的选项和 label\n    if \(props\.value\?\.raw !== null && props\.value\?\.raw !== undefined && props\.value\?\.raw !== ''\) \{[\s\S]*?\} else \{\n      \/\/ 没有值，触发空搜索加载初始选项\n      await handleDialogSearch\(''\)\n    \}/m,
  `    // 🔥 即使有选中值，用户打开弹窗也意味着想重新选择，直接进行空搜索拉取全部选项
    await handleDialogSearch('')`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/SelectWidget.vue', newContent);
console.log('done');
