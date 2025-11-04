眼下组件最麻烦的一个应该就是文件组件了，这个文件类型的组件相当麻烦
这个组件意味着需要承担用户的文件上传和文件的输出展示，关键是这个文件怎么上传，因为我们要支持私有化部署，所以我们要支持多种上传策略
眼下最重要的我觉得应该是通过minio来支持？就是旧版本的逻辑是这样的

先从获取token接口获取到上传的token，然后前端调用七牛云的上传接口把文件上传上去，然后我们把文件地址拿到，
用户files组件的参数是一个结构体，里面有files参数，里面有上传后文件的url，把上传后的文件写进去，提交时候把这个带给后端，
后端如果要处理文件的话是需要直接把文件下载到本地然后处理的，然后输出文件的话，我们后端有files.Files 类型，这个实现了
json的接口，在序列化json的时候会自动调用上传接口把文件上传到七牛云，然后更新上传地址，这样其实后端对上传无感知
，但是旧版本跟七牛云强耦合，不符合私有化部署的策略，私有化部署肯定数据不出域，这样的话我们必需要有自己的存储服务，这样我觉得要minio？
这样的话，我们上传的话可以直接上传到minio然后拿到地址？然后提交到系统里？这样是不是更方便？后续私有化部署也方便？
还有一种场景是无需后端处理文件的，例如工单系统的上传附件，用户上传附件后直接提交后端，后端只是存储在数据库中，不对文件进行处理，这种的话
我们files.Files直接实现接口，保证可以存储成json，然后查询时候查询成json输出，照样可以渲染到表单上即可，这样文件组件即可使用

文件组件的使用场景也是非常广泛的，我给你看看我们旧版本的文件组件 
有很多，例如视频转换，图片转换，excel转换csv，csv转换excel，csv合并，csv分隔，pdf合并，pdf分隔，pdf上水印，ocr文字识别，图片上水印
视频倒放，等等，好多好多的场景，多的很，所以这个组件也是非常重要的一个核心字段

```shell
 tools git:(master) ✗ tree
.
├── csv
│   ├── csv_merge.go
│   └── init_.go
├── database
│   ├── init_.go
│   └── sqlite
│       ├── init_.go
│       └── sqlite_query.go
├── email
│   ├── email_send.go
│   └── init_.go
├── image
│   ├── image_compress.go
│   ├── image_convert.go
│   ├── image_grid.go
│   ├── image_to_pdf.go
│   ├── image_watermark_text.go
│   └── init_.go
├── init_.go
├── keep_.go
├── log
│   ├── init_.go
│   └── log_query.go
├── ocr
│   ├── init_.go
│   ├── ocr_barcode.go
│   ├── ocr_barcode_mark.go
│   ├── ocr_exec_batch_install.go
│   ├── ocr_exec_manage.go
│   ├── ocr_export_structured.go
│   ├── ocr_image.go
│   ├── ocr_pdf_barcode_archive.go
│   ├── ocr_pdf_to_searchable.go
│   ├── ocr_pdf_tools.go
│   └── ocr_table_extract.go
├── office
│   ├── csv
│   │   ├── csv_to_excel.go
│   │   ├── csv_to_sql_insert.go
│   │   └── init_.go
│   ├── excel
│   │   ├── excel_to_csv.go
│   │   ├── excel_to_sql_ddl.go
│   │   ├── excel_to_sql_insert.go
│   │   └── init_.go
│   └── init_.go
├── pdf
│   ├── init_.go
│   ├── keep_.go
│   ├── pdf_encrypt.go
│   ├── pdf_merge.go
│   ├── pdf_optimize.go
│   ├── pdf_rotate.go
│   ├── pdf_split.go
│   ├── pdf_test_split.go
│   ├── pdf_validate.go
│   ├── pdf_watermark_image.go
│   ├── pdf_watermark_text.go
│   ├── pdf_zip.go
│   └── pdf_zip_index.go
├── sft
│   ├── init_.go
│   └── sft_jsonl_manage.go
├── texts
│   └── init_.go
└── video
    ├── README.md
    ├── init_.go
    ├── keep_.go
    ├── video_audio_extract.go
    ├── video_clip.go
    ├── video_concat.go
    ├── video_ffmpeg_manage.go
    ├── video_gif.go
    ├── video_speed.go
    ├── video_subtitle.go
    ├── video_thumbnail.go
    ├── video_transcode.go
    ├── video_transform.go
    └── video_watermark.go

```
