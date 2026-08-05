const fs = require('fs');

const mTypePath = 'web/src/architecture/presentation/widgets/multiSelectWidgetTypes.ts';
const sTypePath = 'web/src/architecture/presentation/widgets/selectWidgetTypes.ts';

function addRichAndFiles(path) {
  let content = fs.readFileSync(path, 'utf8');
  if (!content.includes('rich_text?:')) {
    content = content.replace(
      /richText\?: string\n\s*files\?: string/g,
      `richText?: string\n  files?: string\n  rich_text?: string`
    );
    fs.writeFileSync(path, content);
  }
}

addRichAndFiles(mTypePath);
addRichAndFiles(sTypePath);

const multiPath = 'web/src/architecture/presentation/widgets/MultiSelectWidget.vue';
let mContent = fs.readFileSync(multiPath, 'utf8');
mContent = mContent.replace(
  /rich_text: opt\.richText,([\s\n]+)files: opt\.files/g,
  `rich_text: opt.richText || opt.rich_text,$1files: opt.files`
).replace(
  /richText: item\.rich_text,([\s\n]+)files: item\.files/g,
  `richText: item.rich_text,
        rich_text: item.rich_text,$1files: item.files`
);
fs.writeFileSync(multiPath, mContent);

const singlePath = 'web/src/architecture/presentation/widgets/SelectWidget.vue';
let sContent = fs.readFileSync(singlePath, 'utf8');
sContent = sContent.replace(
  /rich_text: opt\.richText,([\s\n]+)files: opt\.files/g,
  `rich_text: opt.richText || opt.rich_text,$1files: opt.files`
).replace(
  /richText: item\.rich_text,([\s\n]+)files: item\.files/g,
  `richText: item.rich_text,
        rich_text: item.rich_text,$1files: item.files`
);
fs.writeFileSync(singlePath, sContent);

console.log('done');
