#!/bin/bash

# 存储服务测试脚本

BASE_URL="http://localhost:8083/api/v1/storage"
JWT_TOKEN="your-jwt-token-here"  # 需要先登录获取

echo "==> 1. 获取上传凭证"
UPLOAD_RESPONSE=$(curl -s -X POST "$BASE_URL/upload-token" \
  -H "Content-Type: application/json" \
  -H "X-Token: $JWT_TOKEN" \
  -d '{
    "file_name": "test.txt",
    "content_type": "text/plain",
    "file_size": 100
  }')

echo "$UPLOAD_RESPONSE" | jq .

# 提取上传 URL 和 Key
UPLOAD_URL=$(echo "$UPLOAD_RESPONSE" | jq -r '.data.url')
FILE_KEY=$(echo "$UPLOAD_RESPONSE" | jq -r '.data.key')

if [ "$UPLOAD_URL" = "null" ]; then
  echo "❌ 获取上传凭证失败"
  exit 1
fi

echo ""
echo "==> 2. 上传文件到 MinIO（直接上传）"
echo "Hello, MinIO!" > /tmp/test.txt
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: text/plain" \
  --data-binary "@/tmp/test.txt"

echo ""
echo ""
echo "==> 3. 获取文件信息"
curl -s -X GET "$BASE_URL/files/$FILE_KEY/info" \
  -H "X-Token: $JWT_TOKEN" | jq .

echo ""
echo "==> 4. 获取下载链接"
DOWNLOAD_RESPONSE=$(curl -s -X GET "$BASE_URL/download/$FILE_KEY" \
  -H "X-Token: $JWT_TOKEN")
echo "$DOWNLOAD_RESPONSE" | jq .

DOWNLOAD_URL=$(echo "$DOWNLOAD_RESPONSE" | jq -r '.data.url')

echo ""
echo "==> 5. 下载文件并验证内容"
curl -s "$DOWNLOAD_URL"

echo ""
echo ""
echo "==> 6. 删除文件"
curl -s -X DELETE "$BASE_URL/files/$FILE_KEY" \
  -H "X-Token: $JWT_TOKEN" | jq .

echo ""
echo "✅ 测试完成"

