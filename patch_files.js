const fs = require('fs');
const content = fs.readFileSync('web/src/architecture/presentation/widgets/SelectFuzzyPresentation.vue', 'utf8');

// 把 files 部分从模板中删除
let newContent = content.replace(
  /<div v-if="files" class="presentation-files">[\s\S]*?<\/div>\s*<\/div>\s*<\/template>/,
  `</div>\n</template>`
);

// 把 files 的逻辑也删除
newContent = newContent.replace(
  /const hasContent = computed\(\(\) => Boolean\(props\.richText \|\| props\.files\)\)/,
  `const hasContent = computed(() => Boolean(props.richText || props.files))` // 保持逻辑不变，或者改为 props.richText
);

newContent = newContent.replace(
  /<div v-if="files" class="presentation-files">[\s\S]*?<\/div>/,
  ``
);

// 只渲染 files 的情况
newContent = content.replace(
  /<template>[\s\S]*?<\/template>/,
  `<template>
  <div
    v-if="hasContent"
    class="select-fuzzy-presentation"
    :class="{ 'is-compact': compact, 'has-files-only': files && !richText }"
  >
    <div v-if="richText" class="presentation-rich-text-container" :class="{ 'is-expanded': isExpanded, 'is-collapsible': shouldShowToggle }">
      <div class="presentation-rich-text" ref="richTextRef">
        <RichTextResponseWidget
          :field="richTextField"
          :value="richTextValue"
          mode="detail"
          field-path="__select_fuzzy_rich_text"
        />
      </div>
      <div v-if="shouldShowToggle && !isExpanded" class="rich-text-fade-mask"></div>
      <div v-if="shouldShowToggle" class="rich-text-toggle-btn" @click.stop="toggleExpand">
        <span class="toggle-text">{{ isExpanded ? '收起' : '展开全文' }}</span>
        <el-icon class="toggle-icon"><ArrowDown v-if="!isExpanded" /><ArrowUp v-else /></el-icon>
      </div>
    </div>
    <div v-if="files" class="presentation-files">
      <FilesWidget
        :field="filesField"
        :value="filesValue"
        mode="table-cell"
        field-path="__select_fuzzy_files"
      />
    </div>
  </div>
</template>`
);


fs.writeFileSync('web/src/architecture/presentation/widgets/SelectFuzzyPresentation.vue', newContent);
console.log('done');
