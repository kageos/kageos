#!/bin/bash

# Swagger文档生成脚本
# 用于生成AI Agent OS所有服务的API文档

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

# 显示swag版本
swag_version() {
    swag --version 2>/dev/null || swag -v 2>/dev/null || echo "unknown"
}

echo -e "${BLUE}使用swag版本: $(swag_version)${NC}"
echo ""

# 定义需要生成Swagger的服务列表
declare -a services=(
    "app-server:core/app-server/cmd/app/main.go:core/app-server/docs:core/app-server"
    "app-storage:core/app-storage/cmd/app/main.go:core/app-storage/docs:core/app-storage"
    "api-gateway:core/api-gateway/cmd/app/main.go:core/api-gateway/docs:core/api-gateway"
    "agent-server:core/agent-server/cmd/app/main.go:core/agent-server/docs:core/agent-server"
)

# 统计变量
success_count=0
fail_count=0

# 遍历所有服务
for service_config in "${services[@]}"; do
    IFS=':' read -r service_name main_go_path docs_path service_dir <<< "$service_config"
    
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}📦 生成 $service_name 的 Swagger 文档${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # 检查main.go文件是否存在
    if [ ! -f "$main_go_path" ]; then
        echo -e "${RED}❌ 错误: 找不到 $main_go_path${NC}"
        ((fail_count++))
        continue
    fi
    
    # 检查服务目录是否存在
    if [ ! -d "$service_dir" ]; then
        echo -e "${RED}❌ 错误: 找不到服务目录 $service_dir${NC}"
        ((fail_count++))
        continue
    fi
    
    # 创建docs目录（如果不存在）
    mkdir -p "$docs_path"
    
    # 构建排除目录列表（排除其他服务）
    exclude_dirs=""
    for other_service in "app-server" "app-storage" "api-gateway" "app-runtime" "agent-server" "hr-server" "backup-service" "control-service"; do
        service_base_name=$(basename "$service_dir")
        if [ "$other_service" != "$service_base_name" ]; then
            if [ -n "$exclude_dirs" ]; then
                exclude_dirs="$exclude_dirs,core/$other_service"
            else
                exclude_dirs="core/$other_service"
            fi
        fi
    done
    
    # 排除 hub 和 namespace 目录（这些目录在项目根目录，不在 core 目录下）
    # namespace 目录包含用户生成的内容，可能包含非 Go 代码文件
    for exclude_dir in "hub" "namespace"; do
        if [ -n "$exclude_dirs" ]; then
            exclude_dirs="$exclude_dirs,$exclude_dir"
        else
            exclude_dirs="$exclude_dir"
        fi
    done

    # enterprise Hub backend 使用独立路由，不应混入核心服务 Swagger
    if [ -n "$exclude_dirs" ]; then
        exclude_dirs="$exclude_dirs,enterprise_impl/hub/backend"
    else
        exclude_dirs="enterprise_impl/hub/backend"
    fi
    
    echo -e "${YELLOW}正在生成 Swagger 文档...${NC}"
    echo -e "${BLUE}服务目录: $service_dir${NC}"
    if [ -n "$exclude_dirs" ]; then
        echo -e "${BLUE}排除目录: $exclude_dirs${NC}"
    fi
    
    # 在项目根目录执行，但使用 --dir 限制扫描范围到当前服务目录
    # 使用 --parseDependency=false 避免扫描外部依赖包
    # 使用 --parseInternal=false 避免扫描内部包（如 pkg、sdk 等）
    # 使用 --exclude 排除其他服务的目录
    if [ -n "$exclude_dirs" ]; then
        swag_cmd="swag init -g \"$main_go_path\" -o \"$docs_path\" --parseDependency=false --parseInternal=false --exclude \"$exclude_dirs\""
    else
        swag_cmd="swag init -g \"$main_go_path\" -o \"$docs_path\" --parseDependency=false --parseInternal=false"
    fi
    
    if eval "$swag_cmd" 2>&1; then
        echo -e "${GREEN}✅ $service_name Swagger文档生成成功！${NC}"
        echo -e "${BLUE}📁 文档位置: $docs_path/${NC}"
        
        # 显示生成的文件
        if [ -f "$docs_path/swagger.json" ]; then
            # 统计API端点数量（支持多种路径格式）
            endpoint_count=$(grep -o '"/[^"]*"' "$docs_path/swagger.json" 2>/dev/null | grep -v '"/definitions' | sort -u | wc -l | tr -d ' ' || echo "0")
            if [ "$endpoint_count" -gt 0 ]; then
                echo -e "${BLUE}📊 发现 $endpoint_count 个API端点${NC}"
            fi
        fi
        
        ((success_count++))
    else
        echo -e "${RED}❌ $service_name Swagger文档生成失败！${NC}"
        ((fail_count++))
    fi
    
    echo ""
done

# 显示总结
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📊 生成总结${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ 成功: $success_count 个服务${NC}"
if [ $fail_count -gt 0 ]; then
    echo -e "${RED}❌ 失败: $fail_count 个服务${NC}"
fi

echo ""
echo -e "${BLUE}🌐 访问地址:${NC}"
echo -e "  - 网关聚合: http://localhost:9090/swagger"
echo -e "  - app-server: http://localhost:9091/swagger/index.html"
echo -e "  - app-storage: http://localhost:9092/swagger/index.html"
echo -e "  - agent-server: http://localhost:9095/swagger/index.html"

echo ""
echo -e "${GREEN}=== 完成 ===${NC}"
