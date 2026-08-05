const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

// 补回丢失的 import
let newContent = content.replace(
  /import \{ Search, Loading, InfoFilled, ArrowRight, Check \} from '@element-plus\/icons-vue'/,
  `import { Search, Loading, InfoFilled, ArrowRight, Check } from '@element-plus/icons-vue'
import SelectFuzzyPresentation from './SelectFuzzyPresentation.vue'`
);

// 修复 @click 问题（Vue warn 里提到点击选中不能切换可能是因为有些阻止冒泡或者绑定的问题）
newContent = newContent.replace(
  /class="suggestion-item"\s+:class="\{ 'active': selectedIndex === index, 'selected': isItemSelected\(item\) \}"\s+@click="handleItemClick\(item\)"\s+@mouseenter="selectedIndex = index"/g,
  `class="suggestion-item"
            :class="{ 'active': selectedIndex === index, 'selected': isItemSelected(item) }"
            @click.stop="handleItemClick(item)"
            @mouseenter="selectedIndex = index"`
);


fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
