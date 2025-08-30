#!/bin/bash

# Monica Proxy Wails版本构建脚本

echo "=========================================="
echo "Monica Proxy Wails版本构建脚本"
echo "=========================================="

# 检查是否安装了Wails
if ! command -v wails &> /dev/null; then
    echo "错误: 未找到Wails，请先安装Wails v2"
    echo "安装命令: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    exit 1
fi

# 检查Go版本
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "检测到Go版本: $GO_VERSION"

# 检查Node.js和npm
if ! command -v node &> /dev/null; then
    echo "错误: 未找到Node.js，请先安装Node.js"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo "错误: 未找到npm，请先安装npm"
    exit 1
fi

NODE_VERSION=$(node --version)
NPM_VERSION=$(npm --version)
echo "检测到Node.js版本: $NODE_VERSION, npm版本: $NPM_VERSION"

echo ""
echo "清除旧版本文件，以便编译的是最新的代码..."
rm -rf ./build/bin
rm -rf ./frontend/dist

echo "更新图标..."
cp Icon.png ./build/appicon.png

# 进入frontend目录并安装依赖
echo ""
echo "正在安装前端依赖..."
cd frontend
if [ ! -d "node_modules" ]; then
    npm install
    if [ $? -ne 0 ]; then
        echo "错误: 前端依赖安装失败"
        exit 1
    fi
else
    echo "前端依赖已安装，跳过..."
fi

cd ..

# 构建Wails应用
echo ""
echo "正在构建Wails应用..."
wails build

if [ $? -eq 0 ]; then
    echo ""
    echo "=========================================="
    echo "构建成功！"
    echo "可执行文件位置: build/bin/monica-proxy-wails"
    echo "=========================================="
else
    echo ""
    echo "=========================================="
    echo "构建失败！"
    echo "=========================================="
    exit 1
fi