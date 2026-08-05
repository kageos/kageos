const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', 'utf8');

// 修复属性错误
let newContent = content.replace(
  /<SelectFuzzyPresentation\s+v-if="item\.rich_text"\s+:rich-text="item\.rich_text"\s+compact\s+\/>/,
  `<SelectFuzzyPresentation
                :files="item.files"
                compact
              />`
);

// 把富文本放回到原来位置
newContent = newContent.replace(
  /<\/div>\s*<!-- 选择指示器 -->/m,
  `</div>
              <!-- 富文本展示在下方 -->
              <SelectFuzzyPresentation
                v-if="item.rich_text"
                :rich-text="item.rich_text"
                compact
              />
            </div>
            
            <!-- 选择指示器 -->`
);

fs.writeFileSync('web/src/architecture/presentation/widgets/FuzzySearchDialog.vue', newContent);
console.log('done');
