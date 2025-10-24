#!/bin/bash

# Swagger文档生成脚本
# 用于生成AI Agent OS的API文档

set -e  # 遇到错误立即退出

echo "=== 生成Swagger API文档 ==="

# 切换到项目根目录
cd "$(dirname "$0")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查swag命令是否存在
if ! command -v swag &> /dev/null; then
    echo -e "${RED}错误: swag命令未找到，请先安装swag${NC}"
    echo -e "${YELLOW}安装命令: go install github.com/swaggo/swag/cmd/swag@latest${NC}"
    exit 1
fi

# 检查main.go文件是否存在
if [ ! -f "core/app-server/cmd/app/main.go" ]; then
    echo -e "${RED}错误: 找不到main.go文件${NC}"
    exit 1
fi

# 显示swag版本
echo -e "${BLUE}使用swag版本: $(swag version)${NC}"

# 创建docs目录（如果不存在）
mkdir -p core/app-server/docs

# 执行swag init命令
echo -e "${YELLOW}正在生成Swagger文档...${NC}"
swag init -g core/app-server/cmd/app/main.go -o core/app-server/docs

# 检查生成结果
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Swagger文档生成成功！${NC}"
    echo -e "${BLUE}📁 文档位置: core/app-server/docs/${NC}"
    echo -e "${BLUE}🌐 访问地址: http://localhost:9090/swagger/index.html${NC}"
    
    # 显示生成的文件
    echo ""
    echo -e "${YELLOW}生成的文件:${NC}"
    ls -la core/app-server/docs/
    
    # 显示API数量统计
    echo ""
    echo -e "${YELLOW}API统计:${NC}"
    api_count=$(grep -c '"paths"' core/app-server/docs/swagger.json)
    if [ $api_count -gt 0 ]; then
        endpoint_count=$(grep -o '"/api/v1/[^"]*"' core/app-server/docs/swagger.json | wc -l | tr -d ' ')
        echo -e "${BLUE}发现 $endpoint_count 个API端点${NC}"
        
        # 显示主要API端点
        echo ""
        echo -e "${YELLOW}主要API端点:${NC}"
        grep -o '"/api/v1/[^"]*"' core/app-server/docs/swagger.json | sort | uniq | head -10
    else
        echo -e "${BLUE}API端点信息已生成${NC}"
    fi
else
    echo -e "${RED}❌ Swagger文档生成失败！${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}=== 完成 ===${NC}"
